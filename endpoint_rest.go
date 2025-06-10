package butler

import "fmt"

type RestResource[Q any, B any] interface {
	Get(req *Request, params Q) (payload *B, responseOverride *Response)
	List(req *Request, params Q) (payload []B, responseOverride *Response)
	Create(req *Request, body *B) (repBody *B, responseOverride *Response)
	Update(req *Request, params Q, body *B) (repBody *B, responseOverride *Response)
	Delete(req *Request, params Q) (responseOverride *Response)
}

type RestEndpoints[Q any, B any] struct {
	Path string
	Auth AuthHandler
	// One of: `auto`, `none`, `gzip`, `brotli`, `deflate`
	//
	// Default: `auto`
	Encoding string
	// CachePolicy is used to determine the value of the Cache-Control header and the server behavior
	// when receiving a request with a If-None-Match header.
	CachePolicy       *HttpCachePolicy
	StreamingSettings *StreamingSettings
	Resource          RestResource[Q, B]
	// Optional function that runs before a response is sent back to the client
	//
	// value returned by this function (if not nil) will be sent to the client instead
	OnResponse func(requestType string, body *B) any
	// Optional function that runs before the Resource method is called
	//
	// value returned by this function (if not nil) will be passed to the Resource method instead
	OnRequest func(requestType string, body *B) *B

	Description string
	Name        string

	middlewares []Middleware
	parent      EndpointParent
}

func (g *RestEndpoints[T, B]) GetEcho() EchoServer {
	return g.parent.GetEcho()
}

func (g *RestEndpoints[T, B]) GetMiddlewares() []Middleware {
	return append(g.parent.GetMiddlewares(), g.middlewares...)
}

func (g *RestEndpoints[T, B]) GetPath() string {
	return pathJoin(g.parent.GetPath(), g.Path)
}

func (g *RestEndpoints[T, B]) GetAuthHandlers() []AuthHandler {
	if g.Auth == nil {
		return g.parent.GetAuthHandlers()
	}
	return append(g.parent.GetAuthHandlers(), g.Auth)
}

func (g *RestEndpoints[T, B]) GetServer() *Server {
	return g.parent.GetServer()
}

func (g *RestEndpoints[T, B]) Use(middleware Middleware) {
	g.middlewares = append(g.middlewares, middleware)
}

func (g *RestEndpoints[T, B]) Register(server EndpointParent) []EndpointInterface {
	if g.parent != nil {
		panic("rest endpoints cannot be registered twice")
	}

	g.parent = server

	getEndpoint := &BasicEndpoint[T]{
		Method:            "GET",
		Path:              ":id",
		Auth:              g.Auth,
		Encoding:          g.Encoding,
		CachePolicy:       g.CachePolicy,
		StreamingSettings: g.StreamingSettings,
		Name:              fmt.Sprintf("GET endpoint for the %s resource", g.Name),
		Handler: func(request *Request, params T) *Response {
			payload, err := g.Resource.Get(request, params)

			if err != nil {
				return err
			}

			if payload == nil {
				return Respond.NotFound()
			}

			if g.OnResponse != nil {
				v := g.OnResponse("Get", payload)
				if v != nil {
					return Respond.Ok().JSON(v)
				}
			}

			return Respond.Ok().JSON(payload)
		},
	}

	getListEndpoint := &BasicEndpoint[T]{
		Method:            "GET",
		Path:              "",
		Auth:              g.Auth,
		Encoding:          g.Encoding,
		CachePolicy:       g.CachePolicy,
		StreamingSettings: g.StreamingSettings,
		Name:              fmt.Sprintf("GET endpoint for the %s resource", g.Name),
		Handler: func(request *Request, params T) *Response {
			payload, err := g.Resource.List(request, params)

			if err != nil {
				return err
			}

			if payload == nil {
				return Respond.BadRequest()
			}

			if g.OnResponse != nil {
				resp := make([]any, 0, len(payload))

				for idx := range payload {
					v := g.OnResponse("List", &payload[idx])
					if v != nil {
						resp = append(resp, v)
					} else {
						resp = append(resp, payload[idx])
					}

				}

				return Respond.Ok().JSON(resp)
			}

			return Respond.Ok().JSON(payload)
		},
	}

	postEndpoint := &Endpoint[NoParams, B]{
		Method:            "POST",
		Path:              "",
		Auth:              g.Auth,
		Encoding:          g.Encoding,
		StreamingSettings: g.StreamingSettings,
		Name:              fmt.Sprintf("POST endpoint for the %s resource", g.Name),
		Handler: func(request *Request, params NoParams, body *B) *Response {
			if g.OnRequest != nil {
				b := g.OnRequest("Create", body)
				if b != nil {
					body = b
				}
			}

			payload, err := g.Resource.Create(request, body)

			if err != nil {
				return err
			}

			if g.OnResponse != nil {
				v := g.OnResponse("Create", payload)
				if v != nil {
					return Respond.Created().JSON(v)
				}
			}

			return Respond.Created().JSON(payload)
		},
	}

	putEndpoint := &Endpoint[T, B]{
		Method:            "PUT",
		Path:              ":id",
		Auth:              g.Auth,
		Encoding:          g.Encoding,
		StreamingSettings: g.StreamingSettings,
		Name:              fmt.Sprintf("PUT endpoint for the %s resource", g.Name),
		Handler: func(request *Request, params T, body *B) *Response {
			if g.OnRequest != nil {
				b := g.OnRequest("Update", body)
				if b != nil {
					body = b
				}
			}

			payload, err := g.Resource.Update(request, params, body)

			if err != nil {
				return err
			}

			if g.OnResponse != nil {
				v := g.OnResponse("Update", payload)
				if v != nil {
					return Respond.Created().JSON(v)
				}
			}

			return Respond.Ok().JSON(payload)
		},
	}

	deleteEndpoint := &BasicEndpoint[T]{
		Method:            "DELETE",
		Path:              ":id",
		Auth:              g.Auth,
		Encoding:          g.Encoding,
		StreamingSettings: g.StreamingSettings,
		Name:              fmt.Sprintf("PUT endpoint for the %s resource", g.Name),
		Handler: func(request *Request, params T) *Response {
			err := g.Resource.Delete(request, params)

			if err != nil {
				return err
			}

			return Respond.Ok()
		},
	}

	getEndpoint.Register(g)
	getListEndpoint.Register(g)
	postEndpoint.Register(g)
	putEndpoint.Register(g)
	deleteEndpoint.Register(g)

	endpoints := []EndpointInterface{
		getEndpoint,
		getListEndpoint,
		postEndpoint,
		putEndpoint,
		deleteEndpoint,
	}

	return endpoints
}
