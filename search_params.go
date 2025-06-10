package butler

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type NoParams struct{}

type SearchQParam interface {
	Init(ctx RequestContext, name string) *ParamParsingError
}

// #region Query Params

type StringQParam struct {
	value string
	isSet bool
}

// True if the request contained this param
func (p *StringQParam) Has() bool {
	return p.isSet
}

func (p *StringQParam) Get(defaultValue ...string) string {
	if !p.isSet && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return p.value
}

func (p *StringQParam) Set(value string) *ParamParsingError {
	p.value = value
	p.isSet = true
	return nil
}

func (p *StringQParam) Init(ctx RequestContext, name string) *ParamParsingError {
	v := ctx.QueryParam(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type NumberQParam struct {
	value int64
	isSet bool
}

// True if the request contained this param
func (p *NumberQParam) Has() bool {
	return p.isSet
}

func (p *NumberQParam) Get(defaultValue ...int64) int64 {
	if !p.isSet && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return p.value
}

func (p *NumberQParam) Set(value string) *ParamParsingError {
	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return &ParamParsingError{400, "Bad Request", "parsing to number failed"}
	}

	p.value = num
	p.isSet = true
	return nil
}

func (p *NumberQParam) Init(ctx RequestContext, name string) *ParamParsingError {
	v := ctx.QueryParam(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type BoolQParam struct {
	value bool
	isSet bool
}

// True if the request contained this param
func (p *BoolQParam) Has() bool {
	return p.isSet
}

func (p *BoolQParam) Get(defaultValue ...bool) bool {
	if !p.isSet && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return p.value
}

func (p *BoolQParam) Set(value string) *ParamParsingError {
	p.value = value == "1" || strings.ToLower(value) == "true"
	p.isSet = true
	return nil
}

func (p *BoolQParam) Init(ctx RequestContext, name string) *ParamParsingError {
	v := ctx.QueryParam(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

// #endregion Query Params

// #region URL Params

type StringUrlParam struct {
	value string
	isSet bool
}

func (p *StringUrlParam) Get() string {
	return p.value
}

func (p *StringUrlParam) Set(value string) *ParamParsingError {
	p.value = value
	p.isSet = true
	return nil
}

func (p *StringUrlParam) Init(ctx RequestContext, name string) *ParamParsingError {
	v := ctx.Param(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type NumberUrlParam struct {
	value int64
	isSet bool
}

func (p *NumberUrlParam) Get() int64 {
	return p.value
}

func (p *NumberUrlParam) Set(value string) *ParamParsingError {
	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return &ParamParsingError{400, "Bad Request", "parsing to number failed"}
	}

	p.value = num
	p.isSet = true
	return nil
}

func (p *NumberUrlParam) Init(ctx RequestContext, name string) *ParamParsingError {
	v := ctx.Param(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type BoolUrlParam struct {
	value bool
	isSet bool
}

func (p *BoolUrlParam) Get() bool {
	return p.value
}

func (p *BoolUrlParam) Set(value string) *ParamParsingError {
	p.value = value == "1" || strings.ToLower(value) == "true"
	p.isSet = true
	return nil
}

func (p *BoolUrlParam) Init(ctx RequestContext, name string) *ParamParsingError {
	v := ctx.Param(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

// #endregion

type ParamParsingError struct {
	StatusCode int
	Message    string
	LogMessage string
}

func (e *ParamParsingError) Response() *Response {
	resp := &Response{
		Status: e.StatusCode,
	}
	if len(e.Message) > 0 {
		resp.Text(e.Message)
	}
	return resp
}

func (e *ParamParsingError) ToString() string {
	return fmt.Sprintf("%s: %s", e.Message, e.LogMessage)
}

type RequestContext interface {
	Path() string
	Param(name string) string
	QueryParam(name string) string
	Cookie(name string) (*http.Cookie, error)
}

type paramBinder[T any] func(ctx RequestContext) (T, *ParamParsingError)

type internalParamBinder func(rval reflect.Value, ctx RequestContext) *ParamParsingError

func CreateSearchParamsBinder[T any]() paramBinder[T] {
	var paramsType T
	paramsT := reflect.TypeOf(paramsType)
	if paramsT.Kind() == reflect.Pointer {
		paramsT = paramsT.Elem()
	}

	paramInterface := reflect.TypeOf((*SearchQParam)(nil)).Elem()

	paramKeys := make([]internalParamBinder, 0, paramsT.NumField())
	for i := range paramsT.NumField() {
		field := paramsT.Field(i)
		if field.Type.Implements(paramInterface) {
			fname := field.Name
			paramName := strings.ToLower(fname)

			if field.Type.Kind() != reflect.Pointer {
				paramKeys = append(paramKeys, func(rval reflect.Value, ctx RequestContext) *ParamParsingError {
					field := rval.FieldByName(fname)
					fieldValue := field.Interface()
					qParam := fieldValue.(SearchQParam)
					return qParam.Init(ctx, paramName)
				})
			} else {
				paramKeys = append(paramKeys, func(rval reflect.Value, ctx RequestContext) *ParamParsingError {
					field := rval.FieldByName(fname)
					v := reflect.New(field.Type().Elem())
					field.Set(v)
					fieldValue := v.Interface()
					qParam := fieldValue.(SearchQParam)
					return qParam.Init(ctx, paramName)
				})
			}
		} else {
			panic("one of query parameters does not implement the required interface")
		}
	}

	return func(ctx RequestContext) (T, *ParamParsingError) {
		var params T
		paramsT := reflect.ValueOf(&params).Elem()

		for _, bind := range paramKeys {
			err := bind(paramsT, ctx)
			if err != nil {
				return params, err
			}
		}

		return params, nil
	}
}
