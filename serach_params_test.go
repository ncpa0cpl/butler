package butler_test

import (
	"testing"

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
