package httpbutler_test

import (
	"net/http"
	"testing"

	f "github.com/ncpa0cpl/http-butler"
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

type MockRequestContext struct {
	Params map[string]string
}

func (p *MockRequestContext) Param(param string) string {
	v, ok := p.Params[param]
	if ok {
		return v
	}
	return ""
}

func (p *MockRequestContext) QueryParam(param string) string {
	v, ok := p.Params[param]
	if ok {
		return v
	}
	return ""
}

func (p *MockRequestContext) Path() string {
	return ""
}

func (p *MockRequestContext) Cookie(name string) (*http.Cookie, error) {
	return nil, nil
}

func TestParamsBinder(t *testing.T) {
	assert := assert.New(t)

	type MyParams struct {
		Search      *f.StringQParam
		Limit       *f.NumberQParam
		IncludeDel  *f.BoolQParam
		NotProvided *f.StringQParam
	}

	binder := f.CreateSearchParamsBinder[MyParams]()

	mockedParams := &MockRequestContext{
		map[string]string{
			"search":     "foobar",
			"limit":      "69",
			"includedel": "1",
		},
	}

	params, _ := binder(mockedParams)

	assert.Equal("foobar", params.Search.Get())
	assert.Equal(int64(69), params.Limit.Get())
	assert.Equal(true, params.IncludeDel.Get())
	assert.Equal("", params.NotProvided.Get())
}
