package butler

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type NoParams struct{}

type SearchQParam interface {
	Init(req *Request, name string) *ParamParsingError
}

// #region Query Params

type StringQParam struct {
	name  string
	value string
	isSet bool
}

func (p *StringQParam) IsQueryParam() bool {
	return true
}

func (p *StringQParam) AcceptedKind() string {
	return reflect.String.String()
}

// True if the request contained this param
func (p *StringQParam) Has() bool {
	return p.isSet
}

func (p *StringQParam) Name() string {
	return p.name
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

func (p *StringQParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	v := ctx.EchoContext().QueryParam(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type StringListQParam struct {
	name  string
	value []string
	isSet bool
}

func (p *StringListQParam) IsQueryParam() bool {
	return true
}

func (p *StringListQParam) AcceptedKind() string {
	return reflect.String.String()
}

// True if the request contained this param
func (p *StringListQParam) Has() bool {
	return p.isSet
}

func (p *StringListQParam) Name() string {
	return p.name
}

func (p *StringListQParam) Get(defaultValue ...[]string) []string {
	if !p.isSet {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return []string{}
	}
	return p.value
}

func (p *StringListQParam) Set(value []string) *ParamParsingError {
	p.value = value
	p.isSet = true
	return nil
}

func (p *StringListQParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	params := ctx.EchoContext().QueryParams()
	values, ok := params[name]
	if ok {
		return p.Set(values)
	}
	return nil
}

type NumberQParam struct {
	name  string
	value int64
	isSet bool
}

func (p *NumberQParam) IsQueryParam() bool {
	return true
}

func (p *NumberQParam) AcceptedKind() string {
	return reflect.Int64.String()
}

// True if the request contained this param
func (p *NumberQParam) Has() bool {
	return p.isSet
}

func (p *NumberQParam) Name() string {
	return p.name
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
		return &ParamParsingError{400, "Bad Request", "parsing to integer failed", ""}
	}

	p.value = num
	p.isSet = true
	return nil
}

func (p *NumberQParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	v := ctx.EchoContext().QueryParam(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type NumberListQParam struct {
	name  string
	value []int64
	isSet bool
}

func (p *NumberListQParam) IsQueryParam() bool {
	return true
}

func (p *NumberListQParam) AcceptedKind() string {
	return reflect.Int64.String()
}

// True if the request contained this param
func (p *NumberListQParam) Has() bool {
	return p.isSet
}

func (p *NumberListQParam) Name() string {
	return p.name
}

func (p *NumberListQParam) Get(defaultValue ...[]int64) []int64 {
	if !p.isSet {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return []int64{}
	}
	return p.value
}

func (p *NumberListQParam) Set(values []string) *ParamParsingError {
	p.value = make([]int64, len(values))

	for idx, v := range values {
		num, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return &ParamParsingError{400, "Bad Request", "parsing to integer failed", ""}
		}
		p.value[idx] = num
	}

	p.isSet = true
	return nil
}

func (p *NumberListQParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	params := ctx.EchoContext().QueryParams()
	values, ok := params[name]
	if ok {
		return p.Set(values)
	}
	return nil
}

type FloatQParam struct {
	name  string
	value float64
	isSet bool
}

func (p *FloatQParam) IsQueryParam() bool {
	return true
}

func (p *FloatQParam) AcceptedKind() string {
	return reflect.Float64.String()
}

// True if the request contained this param
func (p *FloatQParam) Has() bool {
	return p.isSet
}

func (p *FloatQParam) Name() string {
	return p.name
}

func (p *FloatQParam) Get(defaultValue ...float64) float64 {
	if !p.isSet && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return p.value
}

func (p *FloatQParam) Set(value string) *ParamParsingError {
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return &ParamParsingError{400, "Bad Request", "parsing to float failed", ""}
	}

	p.value = num
	p.isSet = true
	return nil
}

func (p *FloatQParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	v := ctx.EchoContext().QueryParam(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type FloatListQParam struct {
	name  string
	value []float64
	isSet bool
}

func (p *FloatListQParam) IsQueryParam() bool {
	return true
}

func (p *FloatListQParam) AcceptedKind() string {
	return reflect.Float64.String()
}

// True if the request contained this param
func (p *FloatListQParam) Has() bool {
	return p.isSet
}

func (p *FloatListQParam) Name() string {
	return p.name
}

func (p *FloatListQParam) Get(defaultValue ...[]float64) []float64 {
	if !p.isSet {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return []float64{}
	}
	return p.value
}

func (p *FloatListQParam) Set(values []string) *ParamParsingError {
	p.value = make([]float64, len(values))

	for idx, v := range values {
		num, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return &ParamParsingError{400, "Bad Request", "parsing to float failed", ""}
		}
		p.value[idx] = num
	}

	p.isSet = true
	return nil
}

func (p *FloatListQParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	params := ctx.EchoContext().QueryParams()
	values, ok := params[name]
	if ok {
		return p.Set(values)
	}
	return nil
}

type BoolQParam struct {
	name  string
	value bool
	isSet bool
}

func (p *BoolQParam) IsQueryParam() bool {
	return true
}

func (p *BoolQParam) AcceptedKind() string {
	return reflect.Bool.String()
}

// True if the request contained this param
func (p *BoolQParam) Has() bool {
	return p.isSet
}

func (p *BoolQParam) Name() string {
	return p.name
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

func (p *BoolQParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	v := ctx.EchoContext().QueryParam(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

// #endregion Query Params

// #region URL Params

type StringUrlParam struct {
	name  string
	value string
	isSet bool
}

func (p *StringUrlParam) AcceptedKind() string {
	return reflect.String.String()
}

func (p *StringUrlParam) Name() string {
	return p.name
}

func (p *StringUrlParam) Get() string {
	return p.value
}

func (p *StringUrlParam) Set(value string) *ParamParsingError {
	p.value = value
	p.isSet = true
	return nil
}

func (p *StringUrlParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	v := ctx.EchoContext().Param(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type NumberUrlParam struct {
	name  string
	value int64
	isSet bool
}

func (p *NumberUrlParam) AcceptedKind() string {
	return reflect.Int64.String()
}

func (p *NumberUrlParam) Name() string {
	return p.name
}

func (p *NumberUrlParam) Get() int64 {
	return p.value
}

func (p *NumberUrlParam) Set(value string) *ParamParsingError {
	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return &ParamParsingError{400, "Bad Request", "parsing to integer failed", ""}
	}

	p.value = num
	p.isSet = true
	return nil
}

func (p *NumberUrlParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	v := ctx.EchoContext().Param(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type FloatUrlParam struct {
	name  string
	value float64
	isSet bool
}

func (p *FloatUrlParam) AcceptedKind() string {
	return reflect.Float64.String()
}

func (p *FloatUrlParam) Name() string {
	return p.name
}

func (p *FloatUrlParam) Get() float64 {
	return p.value
}

func (p *FloatUrlParam) Set(value string) *ParamParsingError {
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return &ParamParsingError{400, "Bad Request", "parsing to float failed", ""}
	}

	p.value = num
	p.isSet = true
	return nil
}

func (p *FloatUrlParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	v := ctx.EchoContext().Param(name)
	if v != "" {
		return p.Set(v)
	}
	return nil
}

type BoolUrlParam struct {
	name  string
	value bool
	isSet bool
}

func (p *BoolUrlParam) AcceptedKind() string {
	return reflect.Bool.String()
}

func (p *BoolUrlParam) Name() string {
	return p.name
}

func (p *BoolUrlParam) Get() bool {
	return p.value
}

func (p *BoolUrlParam) Set(value string) *ParamParsingError {
	p.value = value == "1" || strings.ToLower(value) == "true"
	p.isSet = true
	return nil
}

func (p *BoolUrlParam) Init(ctx *Request, name string) *ParamParsingError {
	p.name = name
	v := ctx.EchoContext().Param(name)
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
	paramName  string
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
	return fmt.Sprintf("%s: [param='%s'] %s", e.Message, e.paramName, e.LogMessage)
}

type paramBinder[T any] func(ctx *Request) (T, *ParamParsingError)

type internalParamBinder struct {
	paramName string
	bind      func(rval reflect.Value, ctx *Request) *ParamParsingError
}

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
			paramName := fname
			if tagValue := field.Tag.Get("name"); tagValue != "" {
				paramName = tagValue
			}

			if field.Type.Kind() != reflect.Pointer {
				paramKeys = append(paramKeys, internalParamBinder{
					paramName: paramName,
					bind: func(rval reflect.Value, ctx *Request) *ParamParsingError {
						field := rval.FieldByName(fname)
						fieldValue := field.Interface()
						qParam := fieldValue.(SearchQParam)
						return qParam.Init(ctx, paramName)
					},
				})
			} else {
				paramKeys = append(paramKeys, internalParamBinder{
					paramName: paramName,
					bind: func(rval reflect.Value, ctx *Request) *ParamParsingError {
						field := rval.FieldByName(fname)
						v := reflect.New(field.Type().Elem())
						field.Set(v)
						fieldValue := v.Interface()
						qParam := fieldValue.(SearchQParam)
						return qParam.Init(ctx, paramName)
					},
				})
			}
		} else {
			panic("one of query parameters does not implement the required interface")
		}
	}

	return func(ctx *Request) (T, *ParamParsingError) {
		var params T
		paramsT := reflect.ValueOf(&params).Elem()

		for _, binder := range paramKeys {
			err := binder.bind(paramsT, ctx)
			if err != nil {
				err.paramName = binder.paramName
				return params, err
			}
		}

		return params, nil
	}
}

func decapitalize(s string) string {
	if len(s) > 0 {
		char := s[0:1]
		rest := s[1:]
		char = strings.ToLower(char)
		return char + rest
	}
	return s
}

type defaultQParameter[T any] interface {
	Init(req *Request, name string) *ParamParsingError
	Has() bool
	Get(defaultValue ...T) T
	Name() string
	IsQueryParam() bool
	AcceptedKind() string
}

type defaultUrlParameter[T any] interface {
	Init(req *Request, name string) *ParamParsingError
	Get() T
	Name() string
	AcceptedKind() string
}

func unused_assertImplementsInterface() {
	var strqparam defaultQParameter[string]
	var strlistqparam defaultQParameter[[]string]
	var intqparam defaultQParameter[int64]
	var intlistqparam defaultQParameter[[]int64]
	var floatqparam defaultQParameter[float64]
	var floatlistqparam defaultQParameter[[]float64]
	var boolqparam defaultQParameter[bool]

	strqparam = &StringQParam{}
	strlistqparam = &StringListQParam{}
	intqparam = &NumberQParam{}
	intlistqparam = &NumberListQParam{}
	floatqparam = &FloatQParam{}
	floatlistqparam = &FloatListQParam{}
	boolqparam = &BoolQParam{}

	var strurlparam defaultUrlParameter[string]
	var inturlparam defaultUrlParameter[int64]
	var floaturlparam defaultUrlParameter[float64]
	var boolurlparam defaultUrlParameter[bool]

	strurlparam = &StringUrlParam{}
	inturlparam = &NumberUrlParam{}
	floaturlparam = &FloatUrlParam{}
	boolurlparam = &BoolUrlParam{}

	fmt.Println(
		strqparam, strlistqparam, intqparam, intlistqparam,
		floatqparam, floatlistqparam, boolqparam,
		strurlparam, inturlparam, floaturlparam, boolurlparam,
	)
}
