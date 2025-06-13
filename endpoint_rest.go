package butler

import (
	"fmt"

	echo "github.com/labstack/echo/v4"
)

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
	routes      []EndpointInterface
}

func (g *RestEndpoints[T, B]) GetName() string {
	return g.Name
}

func (g *RestEndpoints[T, B]) GetDescription() string {
	return g.Description
}

func (g *RestEndpoints[T, B]) GetSubRoutes() []EndpointInterface {
	return g.routes
}

func (g *RestEndpoints[T, B]) GetEcho() *echo.Echo {
	return g.parent.GetEcho()
}

func (g *RestEndpoints[T, B]) GetMiddlewares() []Middleware {
	return append(g.parent.GetMiddlewares(), g.middlewares...)
}

func (g *RestEndpoints[T, B]) GetPath() string {
	return pathJoin(g.parent.GetPath(), g.Path)
}

func (g *RestEndpoints[T, B]) GetMethod() string {
	return ""
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

func (g *RestEndpoints[T, B]) Register(server EndpointParent) {
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
		Handler: func(request *Request, params T) *Response {
			err := g.Resource.Delete(request, params)

			if err != nil {
				return err
			}

			return Respond.Ok()
		},
	}

	if g.Name != "" {
		getEndpoint.Name = fmt.Sprintf("get one %s", g.Name)
		getListEndpoint.Name = fmt.Sprintf("list %s", g.Name)
		postEndpoint.Name = fmt.Sprintf("create a %s", g.Name)
		putEndpoint.Name = fmt.Sprintf("update a %s", g.Name)
		deleteEndpoint.Name = fmt.Sprintf("delete a %s", g.Name)
	}

	getEndpoint.Register(g)
	getListEndpoint.Register(g)
	postEndpoint.Register(g)
	putEndpoint.Register(g)
	deleteEndpoint.Register(g)

	var zeroResp B
	getEndpoint.responseT = zeroResp
	getListEndpoint.responseT = zeroResp
	postEndpoint.responseT = zeroResp
	putEndpoint.responseT = zeroResp

	endpoints := []EndpointInterface{
		getEndpoint,
		getListEndpoint,
		postEndpoint,
		putEndpoint,
		deleteEndpoint,
	}

	g.routes = endpoints
}

//

func (g *RestEndpoints[T, B]) GetParamsT() any   { return nil }
func (g *RestEndpoints[T, B]) GetBodyT() any     { return nil }
func (g *RestEndpoints[T, B]) GetResponseT() any { return nil }
