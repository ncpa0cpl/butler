package butler

import (
	echo "github.com/labstack/echo/v4"
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

	Description string
	Name        string

	bindParams paramBinder[T]
}

func (e *Endpoint[T, B]) GetPath() string {
	return e.Path
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
	return []Middleware{}
}

func (e *Endpoint[T, B]) ExecuteHandler(ctx echo.Context, request *Request) (retVal *Response) {
	if e.bindParams == nil {
		e.bindParams = CreateSearchParamsBinder[T]()
	}

	body, err := e.parseBody(ctx)
	if err != nil {
		request.Logger.Error(err)
		return Respond.BadRequest()
	}

	params, perr := e.bindParams(ctx)
	if perr != nil {
		request.Logger.Error(perr.ToString())
		return perr.Response()
	}

	response := e.Handler(request, params, body)
	return response
}

func (e *Endpoint[T, B]) Register(parent EndpointParent) []EndpointInterface {
	if e.Handler == nil {
		panic("endpoint has no handler")
	}

	registerEndpoint(e, parent)
	return []EndpointInterface{e}
}

func (e *Endpoint[T, B]) parseBody(ctx echo.Context) (*B, error) {
	var body B
	err := ctx.Bind(&body)
	return &body, err
}
