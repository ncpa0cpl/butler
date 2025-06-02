package httpbutler_test

import (
	"fmt"
	"testing"
	"time"

	echo "github.com/labstack/echo/v4"
	f "github.com/ncpa0cpl/http-butler"
	"github.com/stretchr/testify/assert"
)

type TestRegisteredRoute struct {
	path    string
	handler echo.HandlerFunc
}

type TestServer struct {
	getRoutes     []TestRegisteredRoute
	postRoutes    []TestRegisteredRoute
	patchRoutes   []TestRegisteredRoute
	putRoutes     []TestRegisteredRoute
	deleteRoutes  []TestRegisteredRoute
	headRoutes    []TestRegisteredRoute
	optionsRoutes []TestRegisteredRoute
}

func (s *TestServer) GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	s.getRoutes = append(s.getRoutes, TestRegisteredRoute{
		path, h,
	})
	return nil
}
func (s *TestServer) POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	s.postRoutes = append(s.getRoutes, TestRegisteredRoute{
		path, h,
	})
	return nil
}
func (s *TestServer) PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	s.putRoutes = append(s.getRoutes, TestRegisteredRoute{
		path, h,
	})
	return nil
}
func (s *TestServer) PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	s.patchRoutes = append(s.getRoutes, TestRegisteredRoute{
		path, h,
	})
	return nil
}
func (s *TestServer) DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	s.deleteRoutes = append(s.getRoutes, TestRegisteredRoute{
		path, h,
	})
	return nil
}
func (s *TestServer) OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	s.optionsRoutes = append(s.getRoutes, TestRegisteredRoute{
		path, h,
	})
	return nil
}
func (s *TestServer) HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	s.headRoutes = append(s.getRoutes, TestRegisteredRoute{
		path, h,
	})
	return nil
}
func (s *TestServer) GetPath() string {
	return ""
}
func (s *TestServer) GetMiddlewares() []f.Middleware {
	return []f.Middleware{}
}
func (s *TestServer) GetEcho() f.EchoServer {
	return s
}
func (s *TestServer) GetAuthHandlers() []f.AuthHandler {
	return []f.AuthHandler{}
}

type QParams struct {
	Search     *f.StringQParam
	Limit      *f.NumberQParam
	IncludeDel *f.BoolQParam
}

type Book struct {
	Title string
}

func TestEndpointAdd(t *testing.T) {
	assert := assert.New(t)

	server := &TestServer{}

	end := f.BasicEndpoint[QParams]{
		Method:   "GET",
		Path:     "/books",
		Encoding: "gzip",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params QParams) *f.Response {
			search := params.Search.Get()
			limit := params.Limit.Get()
			del := params.IncludeDel.Get()

			return f.Respond.Ok().JSON([]Book{
				{
					Title: search,
				},
				{
					Title: fmt.Sprintf("%v", limit),
				},
				{
					Title: fmt.Sprintf("%v", del),
				},
			})
		},
	}

	end.Register(server)

	assert.Equal(1, len(server.getRoutes))
	assert.Equal("/books", server.getRoutes[0].path)
}
