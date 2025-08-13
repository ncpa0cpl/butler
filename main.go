package butler

import (
	"cmp"
	"fmt"
	"os"
	"slices"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/ncpa0cpl/butler/echo_middleware/cors"
	"github.com/ncpa0cpl/butler/swag"
	"golang.org/x/crypto/acme/autocert"
)

type LogHandler interface {
	OnLog(loglevel log.Lvl, request *Request, message []any) []any
	OnLogf(loglevel log.Lvl, request *Request, message string, fargs []any) (string, []any)
}

type EndpointParent interface {
	GetServer() *Server
	GetEcho() *echo.Echo
	GetMiddlewares() []Middleware
	GetPath() string
	GetAuthHandlers() []AuthHandler
	GetReqLogHandler() LogHandler
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
	GetResponseContentType() string
	GetDefaultCachePolicy() *HttpCachePolicy
	GetDefaultEncoding() string
}

type Server struct {
	Cors                 *CorsSettings
	Port                 int
	echo                 *echo.Echo
	endpoints            []EndpointInterface
	middlewares          []Middleware
	usageMonitor         UsageMonitor
	requestLoggerHandler LogHandler
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

func (server *Server) GetReqLogHandler() LogHandler {
	return server.requestLoggerHandler
}

func (server *Server) Add(endpoint EndpointInterface) {
	endpoint.Register(server)
	server.endpoints = append(server.endpoints, endpoint)
}

func (server *Server) Use(middleware Middleware) {
	server.middlewares = append(server.middlewares, middleware)
}

// Automatically redirect all HTTP requests to a HTTPS equivalent
func (server *Server) ForceHTTPS() {
	server.echo.Pre(middleware.HTTPSRedirect())
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

func (server *Server) ListenTLS(certFile any, keyFile any) error {
	server.echo.Use(cors.CORSWithConfig(server.Cors.config))

	err := server.echo.StartTLS(fmt.Sprintf(":%v", server.Port), certFile, keyFile)
	if err != nil {
		server.echo.Logger.Error(err)
	}
	return err
}

func (server *Server) ListenAutoTLS(dirCache string, allowedHosts ...string) error {
	server.echo.Use(cors.CORSWithConfig(server.Cors.config))

	server.echo.AutoTLSManager.Cache = autocert.DirCache(dirCache)
	server.echo.AutoTLSManager.HostPolicy = autocert.HostWhitelist(allowedHosts...)

	err := server.echo.StartAutoTLS(fmt.Sprintf(":%v", server.Port))
	if err != nil {
		server.echo.Logger.Error(err)
	}
	return err
}

func (server *Server) Close() {
	server.echo.Close()
}

// add a handler function that can intercept logs made within request handlers, modify them or act on them
//
// log handlers are given access to the *butler.Request but should not modify the http.Request or http.Response
func (server *Server) OnRequestLog(handler LogHandler) {
	server.requestLoggerHandler = handler
}

func methodToSortPrio(m string) int {
	if m == "GET" {
		return 0
	}
	if m == "POST" {
		return 1
	}
	if m == "DELETE" {
		return 2
	}
	if m == "PUT" {
		return 3
	}
	if m == "PATCH" {
		return 4
	}
	return -1
}

func sortEndpoints(a, b swag.EndpointData) int {
	if (a.IsGroup && b.IsGroup) || (!a.IsGroup && !b.IsGroup) {
		if a.Method != b.Method {
			return methodToSortPrio(a.Method) - methodToSortPrio(b.Method)
		}

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

func mapEndpoints(endpoints []EndpointInterface, level int) []swag.EndpointData {
	endpData := make([]swag.EndpointData, 0, len(endpoints))

	for _, endpoint := range endpoints {
		sub := endpoint.GetSubRoutes()
		_, isWs := endpoint.(*WebSocketEndpoint)
		d := swag.EndpointData{
			Name:            endpoint.GetName(),
			Description:     endpoint.GetDescription(),
			Path:            endpoint.GetPath(),
			Method:          endpoint.GetMethod(),
			ParamsT:         swag.NewParamsTypeStructure(endpoint.GetParamsT()),
			BodyT:           swag.NewTypeStructure(endpoint.GetBodyT()),
			ResponseT:       swag.NewTypeStructure(endpoint.GetResponseT()),
			RespContentType: endpoint.GetResponseContentType(),
			Encoding:        endpoint.GetDefaultEncoding(),
			IsWs:            isWs,
			IsGroup:         len(sub) > 0,
			Children:        mapEndpoints(sub, level+1),
		}
		d.PopulateUid(level)
		cachePolicy := endpoint.GetDefaultCachePolicy()
		if cachePolicy != nil {
			d.CacheOptions = cachePolicy.toSwagOptions()
		}
		endpData = append(endpData, d)
	}

	slices.SortFunc(endpData, sortEndpoints)

	return endpData
}

func AddApiDocumentationRoute(path string, server *Server, m ...Middleware) error {
	html, err := swag.CreateApiDocumentation(mapEndpoints(server.endpoints, 1))
	if err != nil {
		return err
	}

	endp := BasicEndpoint[NoParams]{
		Method: "GET",
		Path:   path,
		Handler: func(request *Request, params NoParams) *Response {
			return Respond.Ok().Bytes(html, "text/html")
		},
	}

	for _, middleware := range m {
		endp.Use(middleware)
	}

	server.Add(&endp)

	return nil
}
