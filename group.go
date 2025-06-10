package butler

type Group struct {
	Path string
	Auth AuthHandler

	routes      []EndpointInterface
	middlewares []Middleware
	parent      EndpointParent

	Name        string
	Description string
}

func (g *Group) GetEcho() EchoServer {
	return g.parent.GetEcho()
}

func (g *Group) GetMiddlewares() []Middleware {
	return append(g.parent.GetMiddlewares(), g.middlewares...)
}

func (g *Group) GetPath() string {
	return pathJoin(g.parent.GetPath(), g.Path)
}

func (g *Group) GetAuthHandlers() []AuthHandler {
	if g.Auth == nil {
		return g.parent.GetAuthHandlers()
	}
	return append(g.parent.GetAuthHandlers(), g.Auth)
}

func (g *Group) GetServer() *Server {
	return g.parent.GetServer()
}

func (g *Group) Add(endpoint EndpointInterface) {
	g.routes = append(g.routes, endpoint)
}

func (g *Group) Use(middleware Middleware) {
	g.middlewares = append(g.middlewares, middleware)
}

func (g *Group) Register(server EndpointParent) []EndpointInterface {
	if g.parent != nil {
		panic("group cannot be registered twice")
	}

	endpoints := make([]EndpointInterface, 0, len(g.routes))

	g.parent = server
	for _, endp := range g.routes {
		subendpoints := endp.Register(g)
		endpoints = append(endpoints, subendpoints...)
	}

	return endpoints
}
