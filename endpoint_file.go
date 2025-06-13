package butler

import (
	"fmt"
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
	parent      EndpointParent
}

func (e *FsEndpoint) GetName() string {
	return e.Name
}

func (e *FsEndpoint) GetDescription() string {
	return e.Description
}

func (e *FsEndpoint) GetSubRoutes() []EndpointInterface {
	return []EndpointInterface{}

}

func (e *FsEndpoint) GetPath() string {
	return pathJoin(e.parent.GetPath(), strings.TrimRight(e.Path, "/")+"/*")
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

func (e *FsEndpoint) Register(parent EndpointParent) {
	if e.parent != nil {
		panic("endpoint can only be registered once")
	}

	e.parent = parent

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

	if e.Name == "" {
		e.Name = "Static Files"
	}

	if e.Description == "" {
		e.Description = fmt.Sprintf("Serves static files from the local directory: '%s'", e.Dir)
	}

	registerEndpoint(e, parent)
}

func (e *FsEndpoint) ExecuteHandler(ctx echo.Context, request *Request) (retVal *Response) {
	filepath := ctx.Param("*")
	fullFilepath := path.Join(e.Dir, filepath)

	if !fileExists(fullFilepath) {
		return Respond.NotFound()
	}

	file, err := os.Open(fullFilepath)
	if err != nil {
		request.Logger.Error("failed to open file: ", fullFilepath)
		return Respond.InternalError()
	}

	stat, err := file.Stat()
	if err != nil {
		request.Logger.Error("failed to get file stat: ", fullFilepath)
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

//

func (g *FsEndpoint) GetParamsT() any {
	return nil
}

func (g *FsEndpoint) GetBodyT() any {
	return nil
}

func (g *FsEndpoint) GetResponseT() any {
	return nil
}
