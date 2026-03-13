package butler

import (
	"cmp"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/v5/session"
	echo "github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/labstack/gommon/log"
	"github.com/ncpa0cpl/butler/echo_middleware/cors"
	"github.com/ncpa0cpl/butler/swag"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	log                  ILogger
	echo                 *echo.Echo
	endpoints            []EndpointInterface
	middlewares          []Middleware
	usageMonitor         UsageMonitor
	requestLoggerHandler LogHandler
	http2                bool
	http2Options         *http2.Server
	http3                bool
	http3Server          *http3.Server
	cancelFn             context.CancelFunc
}

func CreateServer() *Server {
	e := echo.New()

	log := NewButlerLogger("", os.Stdout)

	e.Logger = slog.New(log.SlogHandler())

	return &Server{
		Port:      80,
		Cors:      &CorsSettings{},
		log:       log,
		echo:      e,
		endpoints: []EndpointInterface{},
	}
}

func (server *Server) GetEcho() *echo.Echo {
	return server.echo
}

func (server *Server) SetLogger(logger ILogger) {
	server.log = logger
}

func (server *Server) Logger() ILogger {
	return server.log
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

func (server *Server) EnableHttp2(options *http2.Server) {
	server.http2 = true

	if options != nil {
		server.http2Options = options
	} else {
		server.http2Options = &http2.Server{}
	}
}

func (server *Server) EnableHttp3() {
	server.http3 = true
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
	var err error
	server.echo.Use(cors.CORSWithConfig(server.Cors.config))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	server.cancelFn = cancel
	sc := echo.StartConfig{
		Address: fmt.Sprintf(":%v", server.Port),
	}

	if server.http2 {
		h2Handler := h2c.NewHandler(server.echo, server.http2Options)
		if err := sc.Start(ctx, h2Handler); err != nil {
			server.log.Error(err)
		}
		return err
	}

	if err := sc.Start(ctx, server.echo); err != nil {
		server.log.Error(err)
	}

	return err
}

func (server *Server) ListenTLS(certFile, keyFile any) error {
	var err error

	server.echo.Use(cors.CORSWithConfig(server.Cors.config))

	tlsConf, err := newTlsConfig(certFile, keyFile)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	server.cancelFn = cancel

	sc := echo.StartConfig{
		Address:   fmt.Sprintf(":%v", server.Port),
		TLSConfig: tlsConf,
	}

	if server.http3 {
		server.http3Server = &http3.Server{
			Addr:      fmt.Sprintf(":%v", server.Port),
			TLSConfig: tlsConf.Clone(),
			Logger:    server.echo.Logger,
			Handler:   server.echo,
		}

		server.echo.Pre(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				err := server.http3Server.SetQUICHeaders(c.Response().Header())
				if err != nil {
					return err
				}
				return next(c)
			}
		})

		go func() {
			if err := server.http3Server.ListenAndServe(); err != nil {
				server.log.Error(err)
			}
		}()
	}

	if server.http2 {
		sc.TLSConfig.NextProtos = []string{"h2", "http/1.1"}
		sc.BeforeServeFunc = func(s *http.Server) error {
			return http2.ConfigureServer(s, server.http2Options)
		}
	}

	if err := sc.Start(ctx, server.echo); err != nil {
		server.log.Error(err)
	}
	return err
}

func (server *Server) Close() {
	server.cancelFn()
	if server.http3Server != nil {
		server.http3Server.Close()
	}
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

func filepathOrContent(fileOrContent interface{}) (content []byte, err error) {
	switch v := fileOrContent.(type) {
	case string:
		return os.ReadFile(v)
	case []byte:
		return v, nil
	default:
		return nil, echo.ErrInvalidCertOrKeyType
	}
}

func newTlsConfig(certFile, keyFile any) (*tls.Config, error) {
	var err error

	var cert []byte
	if cert, err = filepathOrContent(certFile); err != nil {
		return nil, err
	}

	var key []byte
	if key, err = filepathOrContent(keyFile); err != nil {
		return nil, err
	}

	tlsConf := &tls.Config{}
	tlsConf.Certificates = make([]tls.Certificate, 1)
	if tlsConf.Certificates[0], err = tls.X509KeyPair(cert, key); err != nil {
		return nil, err
	}

	return tlsConf, nil
}
