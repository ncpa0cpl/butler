package butler_test

import (
	"testing"
	"time"

	f "github.com/ncpa0cpl/butler"
	"github.com/stretchr/testify/assert"
)

func TestSearchParams(t *testing.T) {
	assert := assert.New(t)

	strParam := f.StringQParam{}
	strParam.Set("value string")
	assert.Equal("value string", strParam.Get())

	numParam := f.NumberQParam{}
	numParam.Set("123")
	assert.Equal(int64(123), numParam.Get())

	boolParam := f.BoolQParam{}
	boolParam.Set("True")
	assert.Equal(true, boolParam.Get())
}

func TestSearchParamsInStruct(t *testing.T) {
	assert := assert.New(t)

	type MyParams struct {
		Search     f.StringQParam
		Limit      f.NumberQParam
		IncludeDel f.BoolQParam
	}

	var params MyParams

	params.Search.Set("value string")
	assert.Equal("value string", params.Search.Get())

	params.Limit.Set("123")
	assert.Equal(int64(123), params.Limit.Get())

	params.IncludeDel.Set("True")
	assert.Equal(true, params.IncludeDel.Get())
}

func TestTagNames(t *testing.T) {
	assert := assert.New(t)

	type QParams struct {
		P1 *f.StringListQParam `name:"foo"`
		P2 *f.NumberQParam     `name:"bar"`
		P3 *f.StringQParam     `name:"baz"`

		UnnamedStringList *f.StringListQParam
		UnnamedNumber     *f.NumberQParam
		UnnamedString     *f.StringQParam
	}

	server := f.CreateServer()
	server.Port = 8080

	loopback := &f.BasicEndpoint[QParams]{
		Method: "GET",
		Path:   "/loopback",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params QParams) *f.Response {
			return f.Respond.Ok().JSON(struct {
				Named1 []string
				Named2 int64
				Named3 string

				Unnamed1 []string
				Unnamed2 int64
				Unnamed3 string
			}{
				Named1: params.P1.Get(),
				Named2: params.P2.Get(),
				Named3: params.P3.Get(),

				Unnamed1: params.UnnamedStringList.Get(),
				Unnamed2: params.UnnamedNumber.Get(),
				Unnamed3: params.UnnamedString.Get(),
			})
		},
	}

	server.Add(loopback)

	go server.Listen()
	defer server.Close()

	body, resp := request("GET",
		"http://localhost:8080/loopback?foo=hello&foo=world&bar=1&baz=lorem&unnamedStringList=1&unnamedStringList=2&unnamedNumber=0&unnamedString=UnnamedString",
		nil)
	assert.Equal(200, resp.StatusCode)
	assert.Equal(
		"{\"Named1\":[\"hello\",\"world\"],\"Named2\":1,\"Named3\":\"lorem\",\"Unnamed1\":[\"1\",\"2\"],\"Unnamed2\":0,\"Unnamed3\":\"UnnamedString\"}",
		string(body))
}

func TestArrayParams(t *testing.T) {
	assert := assert.New(t)

	type QParams struct {
		StrArray   *f.StringListQParam
		IntArray   *f.NumberListQParam
		FloatArray *f.FloatListQParam
	}

	server := f.CreateServer()
	server.Port = 8080

	loopback := &f.BasicEndpoint[QParams]{
		Method: "GET",
		Path:   "/loopback",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params QParams) *f.Response {
			return f.Respond.Ok().JSON(struct {
				Strings []string
				Ints    []int64
				Floats  []float64
			}{
				Strings: params.StrArray.Get(),
				Ints:    params.IntArray.Get(),
				Floats:  params.FloatArray.Get(),
			})
		},
	}

	server.Add(loopback)

	go server.Listen()
	defer server.Close()

	body, resp := request("GET",
		"http://localhost:8080/loopback?strArray=first+str&strArray=second+str&intArray=1&intArray=-2&intArray=500&floatArray=0&floatArray=0.5&floatArray=1.02",
		nil)
	assert.Equal(200, resp.StatusCode)
	assert.Equal("{\"Strings\":[\"first str\",\"second str\"],\"Ints\":[1,-2,500],\"Floats\":[0,0.5,1.02]}", string(body))
}

func TestArrayParamsWhenEmpty(t *testing.T) {
	assert := assert.New(t)

	type QParams struct {
		StrArray   *f.StringListQParam
		IntArray   *f.NumberListQParam
		FloatArray *f.FloatListQParam
	}

	server := f.CreateServer()
	server.Port = 8080

	loopback := &f.BasicEndpoint[QParams]{
		Method: "GET",
		Path:   "/loopback",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params QParams) *f.Response {
			return f.Respond.Ok().JSON(struct {
				Strings []string
				Ints    []int64
				Floats  []float64
			}{
				Strings: params.StrArray.Get(),
				Ints:    params.IntArray.Get(),
				Floats:  params.FloatArray.Get(),
			})
		},
	}

	server.Add(loopback)

	go server.Listen()
	defer server.Close()

	body, resp := request("GET",
		"http://localhost:8080/loopback",
		nil)
	assert.Equal(200, resp.StatusCode)
	assert.Equal("{\"Strings\":[],\"Ints\":[],\"Floats\":[]}", string(body))
}

func TestArrayParamsWithParsingErrors(t *testing.T) {
	assert := assert.New(t)

	type QParams struct {
		StrArray   *f.StringListQParam
		IntArray   *f.NumberListQParam
		FloatArray *f.FloatListQParam
	}

	server := f.CreateServer()
	server.Port = 8080

	loopback := &f.BasicEndpoint[QParams]{
		Method: "GET",
		Path:   "/loopback",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params QParams) *f.Response {
			return f.Respond.Ok().JSON(struct {
				Strings []string
				Ints    []int64
				Floats  []float64
			}{
				Strings: params.StrArray.Get(),
				Ints:    params.IntArray.Get(),
				Floats:  params.FloatArray.Get(),
			})
		},
	}

	server.Add(loopback)

	go server.Listen()
	defer server.Close()

	_, resp := request("GET",
		"http://localhost:8080/loopback?strArray=first+str&strArray=second+str&intArray=1&intArray=-2.05&intArray=500&floatArray=0&floatArray=0.5&floatArray=1.02",
		nil)
	assert.Equal(400, resp.StatusCode)

	_, resp = request("GET",
		"http://localhost:8080/loopback?strArray=first+str&strArray=second+str&intArray=1&intArray=-2&intArray=500&floatArray=0&floatArray=0.5&floatArray=1,02",
		nil)
	assert.Equal(400, resp.StatusCode)
}
