package openapi3_test

import (
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/swgen"
	"github.com/swaggest/swgen/swjschema"
)

// ISOWeek is a week identifier.
type ISOWeek string

// SwaggerDef returns swagger definition.
func (ISOWeek) SwaggerDef() swgen.SwaggerData {
	s := swgen.SwaggerData{}

	s.Description = "ISO Week"
	s.Example = "2006-W43"
	s.Type = "string"
	s.Pattern = `^[0-9]{4}-W(0[1-9]|[1-4][0-9]|5[0-2])$`

	return s
}

type WeirdResp interface {
	Boo()
}

type UUID [16]byte

type Resp struct {
	HeaderField string `header:"X-Header-Field" description:"Sample header response."`
	Field1      int    `json:"field1"`
	Field2      string `json:"field2"`
	Info        struct {
		Foo string  `json:"foo" default:"baz" required:"true" pattern:"\\d+"`
		Bar float64 `json:"bar" description:"This is Bar."`
	} `json:"info"`
	Parent               *Resp                  `json:"parent,omitempty"`
	Map                  map[string]int64       `json:"map,omitempty"`
	MapOfAnything        map[string]interface{} `json:"mapOfAnything,omitempty"`
	ArrayOfAnything      []interface{}          `json:"arrayOfAnything,omitempty"`
	Whatever             interface{}            `json:"whatever"`
	NullableWhatever     *interface{}           `json:"nullableWhatever,omitempty"`
	RecursiveArray       []WeirdResp            `json:"recursiveArray,omitempty"`
	RecursiveStructArray []Resp                 `json:"recursiveStructArray,omitempty"`
	CustomType           ISOWeek                `json:"customType"`
	UUID                 UUID                   `json:"uuid"`
}

func (r *Resp) Description() string {
	return "This is a sample response."
}

func (r *Resp) Title() string {
	return "Sample Response"
}

var _ jsonschema.Preparer = &Resp{}

func (r *Resp) PrepareJSONSchema(s *jsonschema.Schema) error {
	s.WithExtraPropertiesItem("x-foo", "bar")

	return nil
}

type Req struct {
	InQuery1       int                   `query:"in_query1" required:"true" description:"Query parameter."`
	InQuery2       int                   `query:"in_query2" required:"true" description:"Query parameter."`
	InQuery3       int                   `query:"in_query3" required:"true" description:"Query parameter."`
	InPath         int                   `path:"in_path"`
	InCookie       string                `cookie:"in_cookie" deprecated:"true"`
	InHeader       float64               `header:"in_header"`
	InBody1        int                   `json:"in_body1"`
	InBody2        string                `json:"in_body2"`
	InForm1        string                `formData:"in_form1"`
	InForm2        string                `formData:"in_form2"`
	File1          multipart.File        `formData:"upload1"`
	File2          *multipart.FileHeader `formData:"upload2"`
	UUID           UUID                  `header:"uuid"`
	ArrayCSV       []string              `query:"array_csv" explode:"false"`
	ArraySwg2CSV   []string              `query:"array_swg2_csv" collectionFormat:"csv"`
	ArraySwg2SSV   []string              `query:"array_swg2_ssv" collectionFormat:"ssv"`
	ArraySwg2Pipes []string              `query:"array_swg2_pipes" collectionFormat:"pipes"`
}

type GetReq struct {
	InQuery1 int     `query:"in_query1" required:"true" description:"Query parameter." json:"q1"`
	InQuery2 ISOWeek `query:"in_query2" required:"true" description:"Query parameter." json:"q2"`
	InQuery3 int     `query:"in_query3" required:"true" description:"Query parameter." json:"q3"`
	InPath   int     `path:"in_path" json:"p"`
	InCookie string  `cookie:"in_cookie" deprecated:"true" json:"c"`
	InHeader float64 `header:"in_header" json:"h"`
}

const (
	apiName    = "SampleAPI"
	apiVersion = "1.2.3"
)

func TestReflector_SetRequest_array(t *testing.T) {
	reflector := openapi3.Reflector{}
	reflector.DefaultOptions = append(reflector.DefaultOptions, jsonschema.InterceptType(swjschema.InterceptType))

	s := reflector.SpecEns()
	s.Info.Title = apiName
	s.Info.Version = apiVersion

	op := openapi3.Operation{}

	err := reflector.SetRequest(&op, new([]GetReq), http.MethodPost)
	assert.NoError(t, err)

	require.NoError(t, s.AddOperation(http.MethodPost, "/somewhere", op))

	b, err := json.MarshalIndent(s, "", " ")
	assert.NoError(t, err)

	require.NoError(t, ioutil.WriteFile("_testdata/openapi_req_array_last_run.json", b, 0600))

	expected, err := ioutil.ReadFile("_testdata/openapi_req_array.json")
	require.NoError(t, err)

	assertjson.Equal(t, expected, b)
}

func TestReflector_SetRequest(t *testing.T) {
	reflector := openapi3.Reflector{}
	reflector.DefaultOptions = append(reflector.DefaultOptions, jsonschema.InterceptType(swjschema.InterceptType))

	s := reflector.SpecEns()
	s.Info.Title = apiName
	s.Info.Version = apiVersion

	op := openapi3.Operation{}

	err := reflector.SetRequest(&op, new(GetReq), http.MethodGet)
	assert.NoError(t, err)

	require.NoError(t, s.AddOperation(http.MethodGet, "/somewhere/{in_path}", op))

	b, err := json.MarshalIndent(s, "", " ")
	assert.NoError(t, err)

	require.NoError(t, ioutil.WriteFile("_testdata/openapi_req_last_run.json", b, 0600))

	expected, err := ioutil.ReadFile("_testdata/openapi_req.json")
	require.NoError(t, err)

	assertjson.Equal(t, expected, b)
}

func TestReflector_SetJSONResponse(t *testing.T) {
	reflector := openapi3.Reflector{}
	reflector.DefaultOptions = append(reflector.DefaultOptions, jsonschema.InterceptType(swjschema.InterceptType))

	// Add custom type mappings
	uuidDef := swgen.SwaggerData{}
	uuidDef.Type = "string"
	uuidDef.Format = "uuid"
	uuidDef.Example = "248df4b7-aa70-47b8-a036-33ac447e668d"

	reflector.AddTypeMapping(UUID{}, uuidDef)

	s := reflector.SpecEns()
	s.Info.Title = apiName
	s.Info.Version = apiVersion

	reflector.AddTypeMapping(new(WeirdResp), new(Resp))

	op := openapi3.Operation{}

	err := reflector.SetRequest(&op, new(Req), http.MethodPost)
	assert.NoError(t, err)

	err = reflector.SetJSONResponse(&op, new(WeirdResp), http.StatusOK)
	assert.NoError(t, err)

	err = reflector.SetJSONResponse(&op, new([]WeirdResp), http.StatusConflict)
	assert.NoError(t, err)

	pathItem := openapi3.PathItem{}
	pathItem.
		WithSummary("Path Summary").
		WithDescription("Path Description")
	s.Paths.WithMapOfPathItemValuesItem("/somewhere/{in_path}", pathItem)

	js := op.RequestBody.RequestBody.Content["application/json"].Schema.ToJSONSchema(s)
	jsb, err := json.MarshalIndent(js, "", " ")
	require.NoError(t, err)
	expected, err := ioutil.ReadFile("_testdata/req_schema.json")
	require.NoError(t, err)
	assertjson.Equal(t, expected, jsb, string(jsb))

	assert.NoError(t, s.AddOperation(http.MethodPost, "/somewhere/{in_path}", op))

	op = openapi3.Operation{}

	err = reflector.SetRequest(&op, new(GetReq), http.MethodGet)
	assert.NoError(t, err)

	err = reflector.SetJSONResponse(&op, new(Resp), http.StatusOK)
	assert.NoError(t, err)

	assert.NoError(t, s.AddOperation(http.MethodGet, "/somewhere/{in_path}", op))

	js = op.Responses.MapOfResponseOrRefValues[strconv.Itoa(http.StatusOK)].Response.Content["application/json"].
		Schema.ToJSONSchema(s)
	jsb, err = json.MarshalIndent(js, "", " ")
	require.NoError(t, err)

	require.NoError(t, ioutil.WriteFile("_testdata/resp_schema_last_run.json", jsb, 0600))

	expected, err = ioutil.ReadFile("_testdata/resp_schema.json")
	require.NoError(t, err)
	assertjson.Equal(t, expected, jsb, string(jsb))

	js = op.Responses.MapOfResponseOrRefValues[strconv.Itoa(http.StatusOK)].Response.Headers["X-Header-Field"].Header.
		Schema.ToJSONSchema(s)
	jsb, err = json.Marshal(js)
	require.NoError(t, err)
	assertjson.Equal(t, []byte(`{"type": "string", "description": "Sample header response."}`), jsb)

	js = op.Parameters[0].Parameter.Schema.ToJSONSchema(s)
	jsb, err = json.Marshal(js)
	require.NoError(t, err)
	assertjson.Equal(t, []byte(`{"type": "integer", "description": "Query parameter."}`), jsb)

	b, err := json.MarshalIndent(s, "", " ")
	assert.NoError(t, err)

	require.NoError(t, ioutil.WriteFile("_testdata/openapi_last_run.json", b, 0600))

	expected, err = ioutil.ReadFile("_testdata/openapi.json")
	require.NoError(t, err)

	assertjson.Equal(t, expected, b)
}

type Identity struct {
	ID string `path:"id"`
}

type Data []string

type PathParamAndBody struct {
	Identity
	Data
}

func TestReflector_SetRequest_pathParamAndBody(t *testing.T) {
	reflector := openapi3.Reflector{}
	op := openapi3.Operation{}

	err := reflector.SetRequest(&op, new(PathParamAndBody), http.MethodPost)
	assert.NoError(t, err)

	s := reflector.SpecEns()
	s.Info.Title = apiName
	s.Info.Version = apiVersion
	assert.NoError(t, s.AddOperation(http.MethodPost, "/somewhere/{id}", op))

	b, err := json.MarshalIndent(s, "", " ")
	assert.NoError(t, err)

	expected := []byte(`{
        	            	 "openapi": "3.0.2",
        	            	 "info": {
        	            	  "title": "SampleAPI",
        	            	  "version": "1.2.3"
        	            	 },
        	            	 "paths": {
        	            	  "/somewhere/{id}": {
        	            	   "post": {
        	            	    "parameters": [
        	            	     {
        	            	      "name": "id",
        	            	      "in": "path",
        	            	      "required": true,
        	            	      "schema": {
        	            	       "type": "string"
        	            	      }
        	            	     }
        	            	    ],
        	            	    "requestBody": {
        	            	     "content": {
        	            	      "application/json": {
        	            	       "schema": {
        	            	        "$ref": "#/components/schemas/Openapi3TestPathParamAndBody"
        	            	       }
        	            	      }
        	            	     }
        	            	    },
        	            	    "responses": {}
        	            	   }
        	            	  }
        	            	 },
        	            	 "components": {
        	            	  "schemas": {
        	            	   "Openapi3TestPathParamAndBody": {
        	            	    "type": "array",
        	            	    "items": {
        	            	     "type": "string"
        	            	    },
        	            	    "nullable": true
        	            	   }
        	            	  }
        	            	 }
        	            	}`)

	assertjson.Equal(t, expected, b, string(b))
}

type WithReqBody PathParamAndBody

func (*WithReqBody) ForceRequestBody() {}

func TestRequestBodyEnforcer(t *testing.T) {
	reflector := openapi3.Reflector{}
	op := openapi3.Operation{}

	err := reflector.SetRequest(&op, new(WithReqBody), http.MethodGet)
	assert.NoError(t, err)

	s := reflector.SpecEns()
	s.Info.Title = apiName
	s.Info.Version = apiVersion
	assert.NoError(t, s.AddOperation(http.MethodGet, "/somewhere/{id}", op))

	b, err := json.MarshalIndent(s, "", " ")
	assert.NoError(t, err)

	expected := []byte(`{
        	            	 "openapi": "3.0.2",
        	            	 "info": {
        	            	  "title": "SampleAPI",
        	            	  "version": "1.2.3"
        	            	 },
        	            	 "paths": {
        	            	  "/somewhere/{id}": {
        	            	   "get": {
        	            	    "parameters": [
        	            	     {
        	            	      "name": "id",
        	            	      "in": "path",
        	            	      "required": true,
        	            	      "schema": {
        	            	       "type": "string"
        	            	      }
        	            	     }
        	            	    ],
        	            	    "requestBody": {
        	            	     "content": {
        	            	      "application/json": {
        	            	       "schema": {
        	            	        "$ref": "#/components/schemas/Openapi3TestWithReqBody"
        	            	       }
        	            	      }
        	            	     }
        	            	    },
        	            	    "responses": {}
        	            	   }
        	            	  }
        	            	 },
        	            	 "components": {
        	            	  "schemas": {
        	            	   "Openapi3TestWithReqBody": {
        	            	    "type": "array",
        	            	    "items": {
        	            	     "type": "string"
        	            	    },
        	            	    "nullable": true
        	            	   }
        	            	  }
        	            	 }
        	            	}`)

	assertjson.Equal(t, expected, b, string(b))
}
