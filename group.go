package butler

import echo "github.com/labstack/echo/v4"

type Group struct {
	Path string
	Auth AuthHandler

	routes      []EndpointInterface
	middlewares []Middleware
	parent      EndpointParent

	Name        string
	Description string
}

func (g *Group) GetEcho() *echo.Echo {
	return g.parent.GetEcho()
}

func (g *Group) GetMiddlewares() []Middleware {
	return append(g.parent.GetMiddlewares(), g.middlewares...)
}

func (g *Group) GetPath() string {
	return pathJoin(g.parent.GetPath(), g.Path)
}

func (g *Group) GetMethod() string {
	return ""
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

func (g *Group) GetName() string {
	return g.Name
}

func (g *Group) GetDescription() string {
	return g.Description
}

func (g *Group) GetSubRoutes() []EndpointInterface {
	return g.routes
}

func (g *Group) Add(endpoint EndpointInterface) {
	g.routes = append(g.routes, endpoint)
}

func (g *Group) Use(middleware Middleware) {
	g.middlewares = append(g.middlewares, middleware)
}

func (g *Group) Register(server EndpointParent) {
	if g.parent != nil {
		panic("group cannot be registered twice")
	}

	g.parent = server
	for _, endp := range g.routes {
		endp.Register(g)
	}

}

//

func (g *Group) GetParamsT() any   { return nil }
func (g *Group) GetBodyT() any     { return nil }
func (g *Group) GetResponseT() any { return nil }
