package butler

import (
	echo "github.com/labstack/echo/v5"
)

type Endpoint[T any, B any] struct {
	Method string
	Path   string
	Auth   AuthHandler
	// One of: `auto`, `none`, `gzip`, `brotli`, `deflate`
	//
	// Default: `auto`
	Encoding string
	// CachePolicy is used to determine the value of the Cache-Control header and the server behavior
	// when receiving a request with a If-None-Match header.
	CachePolicy       *HttpCachePolicy
	StreamingSettings *StreamingSettings
	Handler           func(request *Request, params T, body *B) *Response
	// Optional function for generating an ETag
	//
	// Value returned by this function will be set in the response Etag header.
	//
	// If the returned ETag matches the requests `If-None-Match` header
	// the handler call will be skipped and a 304 response will be sent
	//
	// If this function is nil, Butler will instead read the response body and
	// generate a hash to be used as an ETag
	GetEtag func(request *Request) string

	Description string
	Name        string

	// Optional. Type assigned to this field will be used to generate the response type in the documentation
	ResponseType any
	// Optional. Type assigned to this field will be used to generate the response type in the documentation
	// If ResponseType is provided and this field is empty, it will default to JSON
	ResponseContentType string

	middlewares []Middleware
	bindParams  paramBinder[T]
	parent      EndpointParent
}

func (e *Endpoint[T, B]) GetName() string {
	return e.Name
}

func (e *Endpoint[T, B]) GetDescription() string {
	return e.Description
}

func (e *Endpoint[T, B]) GetSubRoutes() []EndpointInterface {
	return []EndpointInterface{}
}

func (e *Endpoint[T, B]) GetPath() string {
	return pathJoin(e.parent.GetPath(), e.Path)
}

func (e *Endpoint[T, B]) GetMethod() string {
	return e.Method
}

func (e *Endpoint[T, B]) GetAuth() AuthHandler {
	return e.Auth
}

func (e *Endpoint[T, B]) GetEncoding() string {
	return e.Encoding
}

func (e *Endpoint[T, B]) GetCachePolicy() *HttpCachePolicy {
	return e.CachePolicy
}

func (e *Endpoint[T, B]) GetStreamingSettings() *StreamingSettings {
	return e.StreamingSettings
}

func (e *Endpoint[T, B]) GetMiddlewares() []Middleware {
	return e.middlewares
}

func (e *Endpoint[T, B]) Use(middleware Middleware) {
	e.middlewares = append(e.middlewares, middleware)
}

func (e *Endpoint[T, B]) EtagGenerator() func(request *Request) string {
	return e.GetEtag
}

func (e *Endpoint[T, B]) BindParams(request *Request) *ParamParsingError {
	if e.bindParams == nil {
		e.bindParams = CreateSearchParamsBinder[T]()
	}

	params, perr := e.bindParams(request)
	if perr != nil {
		return perr
	}

	request.setParamsInterface(params)

	return nil
}

func (e *Endpoint[T, B]) ExecuteHandler(ctx *echo.Context, request *Request) (retVal *Response) {
	body, err := e.parseBody(ctx)
	if err != nil {
		request.Logger.Error(err)
		return Respond.BadRequest()
	}

	params := request.GetParamsInterface().(T)

	response := e.Handler(request, params, body)
	return response
}

func (e *Endpoint[T, B]) Register(parent EndpointParent) {
	if e.Handler == nil {
		panic("endpoint has no handler")
	}
	if e.parent != nil {
		panic("endpoint can only be registered once")
	}

	e.parent = parent
	registerEndpoint(e, parent)
}

func (e *Endpoint[T, B]) parseBody(ctx *echo.Context) (*B, error) {
	var body B
	err := ctx.Bind(&body)
	return &body, err
}

//

func (g *Endpoint[T, B]) GetParamsT() any {
	var zeroP T
	return zeroP
}

func (g *Endpoint[T, B]) GetBodyT() any {
	var zeroB B
	return zeroB
}

func (g *Endpoint[T, B]) GetResponseT() any {
	return g.ResponseType
}

func (g *Endpoint[T, B]) GetResponseContentType() string {
	if g.ResponseContentType != "" {
		return g.ResponseContentType
	}

	if g.ResponseType != nil {
		return "application/json"
	}

	return ""
}

func (g *Endpoint[T, B]) GetDefaultCachePolicy() *HttpCachePolicy {
	return g.CachePolicy
}

func (g *Endpoint[T, B]) GetDefaultEncoding() string {
	if g.Encoding != "" {
		return g.Encoding
	}
	return "auto"
}
