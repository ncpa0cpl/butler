package httpbutler

import (
	"net/http"
	"os"
	"path"
	"strings"

	echo "github.com/labstack/echo/v4"
)

type FsEndpoint struct {
	Path string
	Dir  string
	Auth AuthHandler
	// Specifies the Content Encoding that should be used for the endpoint responses
	Encoding string
	// CachePolicy is used to determine the value of the Cache-Control header and the server behavior
	// when receiving a request with a If-None-Match header.
	CachePolicy       *HttpCachePolicy
	StreamingSettings *StreamingSettings
	DisableStreaming  bool
	// Optional handler function
	Handler func(request *Request, file []byte, fstat os.FileInfo) *Response

	Description string
	Name        string

	middlewares []Middleware
}

func (e *FsEndpoint) GetPath() string {
	return strings.TrimRight(e.Path, "/") + "/*"
}

func (e *FsEndpoint) GetMethod() string {
	return "GET"
}

func (e *FsEndpoint) GetAuth() AuthHandler {
	return e.Auth
}

func (e *FsEndpoint) GetEncoding() string {
	return e.Encoding
}

func (e *FsEndpoint) GetCachePolicy() *HttpCachePolicy {
	return e.CachePolicy
}

func (e *FsEndpoint) GetStreamingSettings() *StreamingSettings {
	return e.StreamingSettings
}

func (e *FsEndpoint) GetMiddlewares() []Middleware {
	return e.middlewares
}

func (e *FsEndpoint) Use(middleware Middleware) {
	e.middlewares = append(e.middlewares, middleware)
}

func (e *FsEndpoint) Register(parent EndpointParent) []EndpointInterface {
	if e.Handler == nil {
		e.Handler = func(request *Request, file []byte, fstat os.FileInfo) *Response {
			modTime := fstat.ModTime()

			response := Respond.Ok().Blob(file)
			response.Headers.Set("Last-Modified", modTime.Format(http.TimeFormat))

			return response
		}
	}

	registerEndpoint(e, parent)
	return []EndpointInterface{e}
}

func (e *FsEndpoint) ExecuteHandler(ctx echo.Context, request *Request) (retVal *Response) {
	defer func() {
		if r := recover(); r != nil {
			retVal = Respond.InternalError()
		}
	}()

	filepath := ctx.Param("*")
	fullFilepath := path.Join(e.Dir, filepath)

	if !fileExists(fullFilepath) {
		return Respond.NotFound()
	}

	file, err := os.Open(fullFilepath)
	defer file.Close()
	if err != nil {
		ctx.Logger().Error("failed to open file: ", fullFilepath)
		return Respond.InternalError()
	}

	stat, err := file.Stat()
	if err != nil {
		ctx.Logger().Error("failed to get file stat: ", fullFilepath)
		return Respond.InternalError()
	}

	if stat.IsDir() {
		return Respond.NotFound()
	}

	fcontent := make([]byte, 0, stat.Size())
	_, err = file.Read(fcontent)
	if err != nil {
		ctx.Logger().Error("failed to read file: ", fullFilepath)
		return Respond.InternalError()
	}

	resp := e.Handler(request, fcontent, stat)

	if e.DisableStreaming {
		resp.SetAllowStreaming(false)
	}

	return resp
}
