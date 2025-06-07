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
	Handler func(
		request *Request,
		fullFilepath string,
		file *os.File,
		fstat os.FileInfo,
	) *Response

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
		e.Handler = func(
			request *Request,
			fpath string,
			file *os.File,
			fstat os.FileInfo,
		) *Response {
			fmime := Mime.DetectFile(fpath, file)
			modTime := fstat.ModTime()

			var response *Response

			if e.DisableStreaming || fmime == "text/javascript" || fmime == "text/html" ||
				fmime == "text/css" || fmime == "application/json" ||
				fstat.Size() < Units.MB {
				response = Respond.Ok().FileHandle(file, fmime)
			} else {
				response = Respond.Ok().StreamFileHandle(file, fmime)
			}

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

	resp := e.Handler(request, fullFilepath, file, stat)

	if e.DisableStreaming {
		resp.SetAllowStreaming(false)
	}

	return resp
}
