package butler

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

	// Optional. Type assigned to this field will be used to generate the response type in the documentation
	ResponseType any

	bindParams paramBinder[T]
	parent     EndpointParent
}

func (e *BasicEndpoint[T]) GetName() string {
	return e.Name
}

func (e *BasicEndpoint[T]) GetDescription() string {
	return e.Description
}

func (g *BasicEndpoint[T]) GetSubRoutes() []EndpointInterface {
	return []EndpointInterface{}
}

func (e *BasicEndpoint[T]) GetPath() string {
	return pathJoin(e.parent.GetPath(), e.Path)
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

func (e *BasicEndpoint[T]) GetMiddlewares() []Middleware {
	return []Middleware{}
}

func (e *BasicEndpoint[T]) ExecuteHandler(ctx echo.Context, request *Request) (retVal *Response) {
	if e.bindParams == nil {
		e.bindParams = CreateSearchParamsBinder[T]()
	}

	params, err := e.bindParams(ctx)
	if err != nil {
		request.Logger.Error(err.ToString())
		return err.Response()
	}

	response := e.Handler(request, params)
	return response
}

func (e *BasicEndpoint[T]) Register(parent EndpointParent) {
	if e.Handler == nil {
		panic("endpoint has no handler")
	}
	if e.parent != nil {
		panic("endpoint can only be registered once")
	}

	e.parent = parent

	registerEndpoint(e, parent)
}

//

func (g *BasicEndpoint[T]) GetParamsT() any {
	var zeroP T
	return zeroP
}

func (g *BasicEndpoint[T]) GetBodyT() any {
	return nil
}

func (g *BasicEndpoint[T]) GetResponseT() any {
	return g.ResponseType
}
