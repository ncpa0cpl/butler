package butler

type MiddlewareRequestHandler func(
	request *Request,
	next func(request *Request),
	respond func(response *Response),
) error

type MiddlewareResponseHandler func(
	request *Request,
	response *Response,
	next func(response *Response),
) error

type Middleware struct {
	Name       string
	OnRequest  MiddlewareRequestHandler
	OnResponse MiddlewareResponseHandler
}

func getReqMiddlewares(middlewares []Middleware) []Middleware {
	handlers := make([]Middleware, 0, len(middlewares))

	for _, md := range middlewares {
		if md.OnRequest != nil {
			handlers = append(handlers, md)
		}
	}

	return handlers
}

func getRespMiddlewares(middlewares []Middleware) []Middleware {
	handlers := make([]Middleware, 0, len(middlewares))

	for _, md := range middlewares {
		if md.OnResponse != nil {
			handlers = append(handlers, md)
		}
	}

	return handlers
}
