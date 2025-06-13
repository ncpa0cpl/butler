package butler

import (
	"cmp"
	"fmt"
	"os"
	"slices"

	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echo "github.com/labstack/echo/v4"
	"github.com/ncpa0cpl/butler/echo_middleware/cors"
	"github.com/ncpa0cpl/butler/swag"
)

type EndpointParent interface {
	GetServer() *Server
	GetEcho() *echo.Echo
	GetMiddlewares() []Middleware
	GetPath() string
	GetAuthHandlers() []AuthHandler
}

type EndpointInterface interface {
	Register(server EndpointParent)

	GetName() string
	GetDescription() string
	GetSubRoutes() []EndpointInterface
	GetPath() string
	GetMethod() string
	GetParamsT() any
	GetBodyT() any
	GetResponseT() any
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

func (server *Server) GetEcho() *echo.Echo {
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
	endpoint.Register(server)
	server.endpoints = append(server.endpoints, endpoint)
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
	server.echo.Use(cors.CORSWithConfig(server.Cors.config))

	err := server.echo.Start(fmt.Sprintf(":%v", server.Port))
	if err != nil {
		server.echo.Logger.Error(err)
	}
	return err
}

func (server *Server) Close() {
	server.echo.Close()
}

func sortEndpoints(a, b swag.EndpointData) int {
	if (a.IsGroup && b.IsGroup) || (!a.IsGroup && !b.IsGroup) {
		aName := a.Name
		bName := b.Name

		if aName == "" {
			aName = a.Path
		}
		if bName == "" {
			bName = b.Path
		}

		return cmp.Compare(aName, bName)
	}

	if a.IsGroup {
		return -1
	}
	return 1
}

func mapEndpoints(engpoints []EndpointInterface) []swag.EndpointData {
	endpData := make([]swag.EndpointData, 0, len(engpoints))

	for _, endpoint := range engpoints {
		sub := endpoint.GetSubRoutes()
		uid, _ := uuid.NewV4()
		endpData = append(endpData, swag.EndpointData{
			Uid:         uid.String(),
			Name:        endpoint.GetName(),
			Description: endpoint.GetDescription(),
			Path:        endpoint.GetPath(),
			Method:      endpoint.GetMethod(),
			ParamsT:     swag.NewParamsTypeStructure(endpoint.GetParamsT()),
			BodyT:       swag.NewTypeStructure(endpoint.GetBodyT()),
			ResponseT:   swag.NewTypeStructure(endpoint.GetResponseT()),
			IsGroup:     len(sub) > 0,
			Children:    mapEndpoints(sub),
		})
	}

	slices.SortFunc(endpData, sortEndpoints)

	return endpData
}

func AddApiDocumentationRoute(path string, server *Server, m ...echo.MiddlewareFunc) {
	swag.CreateApiDocumentation(path, mapEndpoints(server.endpoints), server.echo, m...)
}
