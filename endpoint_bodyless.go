package httpbutler

import echo "github.com/labstack/echo/v4"

type BasicEndpoint[T any] struct {
	Method string
	Path   string
	Auth   AuthHandler
	// Specifies the Content Encoding that should be used for the endpoint responses
	Encoding string
	// CachePolicy is used to determine the value of the Cache-Control header and the server behavior
	// when receiving a request with a If-None-Match header.
	CachePolicy       *HttpCachePolicy
	StreamingSettings *StreamingSettings
	Handler           func(request *Request, params T) *Response

	Description string
	Name        string

	bindParams paramBinder[T]
}

func (e *BasicEndpoint[T]) GetPath() string {
	return e.Path
}

func (e *BasicEndpoint[T]) GetMethod() string {
	return e.Method
}

func (e *BasicEndpoint[T]) GetAuth() AuthHandler {
	return e.Auth
}

func (e *BasicEndpoint[T]) GetEncoding() string {
	return e.Encoding
}

func (e *BasicEndpoint[T]) GetCachePolicy() *HttpCachePolicy {
	return e.CachePolicy
}

func (e *BasicEndpoint[T]) GetStreamingSettings() *StreamingSettings {
	return e.StreamingSettings
}

func (e *BasicEndpoint[T]) ExecuteHandler(ctx echo.Context, request *Request) (retVal *Response) {
	defer func() {
		if r := recover(); r != nil {
			retVal = Respond.InternalError()
		}
	}()

	if e.bindParams == nil {
		e.bindParams = CreateSearchParamsBinder[T]()
	}

	params, err := e.bindParams(ctx)
	if err != nil {
		ctx.Logger().Error(err.ToString())
		return err.Response()
	}

	response := e.Handler(request, params)
	return response
}

func (e *BasicEndpoint[T]) Register(parent EndpointParent) []EndpointInterface {
	if e.Handler == nil {
		panic("endpoint has no handler")
	}

	registerEndpoint(e, parent)
	return []EndpointInterface{e}
}
