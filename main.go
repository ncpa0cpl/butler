package butler

import (
	"fmt"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type EchoServer interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	Any(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) []*echo.Route
}

type EndpointParent interface {
	GetServer() *Server
	GetEcho() EchoServer
	GetMiddlewares() []Middleware
	GetPath() string
	GetAuthHandlers() []AuthHandler
}

type EndpointInterface interface {
	Register(server EndpointParent) []EndpointInterface
}

type Server struct {
	Cors         *CorsSettings
	Port         int
	echo         *echo.Echo
	endpoints    []EndpointInterface
	middlewares  []Middleware
	usageMonitor UsageMonitor
}

func CreateServer() *Server {
	e := echo.New()

	e.Logger = NewButlerLogger("", os.Stdout)

	return &Server{
		Port:      80,
		Cors:      &CorsSettings{},
		echo:      e,
		endpoints: []EndpointInterface{},
	}
}

func (server *Server) GetEcho() EchoServer {
	return server.echo
}

func (server *Server) SetLogger(logger echo.Logger) {
	server.echo.Logger = logger
}

func (server *Server) Logger() echo.Logger {
	return server.echo.Logger
}

func (server *Server) SetSessionStore(store sessions.Store) {
	md := session.Middleware(store)
	server.echo.Use(md)
}

func (server *Server) GetMiddlewares() []Middleware {
	return server.middlewares
}

func (server *Server) GetPath() string {
	return ""
}

func (server *Server) GetAuthHandlers() []AuthHandler {
	return []AuthHandler{}
}

func (server *Server) GetServer() *Server {
	return server
}

func (server *Server) Add(endpoint EndpointInterface) {
	routes := endpoint.Register(server)
	server.endpoints = append(server.endpoints, routes...)
}

func (server *Server) Use(middleware Middleware) {
	server.middlewares = append(server.middlewares, middleware)
}

// add a usage monitor to the app
//
// monitor will only receive records for endpoints that were added after the monitor was registered
func (server *Server) Monitor(usageMonitor UsageMonitor) {
	server.usageMonitor = usageMonitor
}

func (server *Server) Listen() error {
	server.echo.Use(middleware.CORSWithConfig(server.Cors.config))

	err := server.echo.Start(fmt.Sprintf(":%v", server.Port))
	if err != nil {
		server.echo.Logger.Error(err)
	}
	return err
}

func (server *Server) Close() {
	server.echo.Close()
}
