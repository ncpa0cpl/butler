package butler

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	echo "github.com/labstack/echo/v5"
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
	// A handler function for adding headers to the response.
	//
	// Header values set by this handler take priority over what the Handler
	// defines.
	//
	// This function is also used to generate responses to http HEAD requests.
	//
	// Example:
	//   import (b "github.com/ncpa0cpl/butler")
	//
	//   b.FsEndpoint{
	//     Head: func(req *b.Request, filepath string, s int) *b.Headers {
	// 	     return b.NewHeaders().
	// 	     	Set("Expires", "2026-03-16T12:00:00").
	// 	     	Set("X-Powered-By", "http-butler")
	//     },
	//   }
	Head func(request *Request, fullFilepath string, status int) *Headers
	// Optional handler function
	Handler func(
		request *Request,
		fullFilepath string,
		file *os.File,
		fstat os.FileInfo,
	) *Response
	isCustomHandler bool
	// Optional function for generating an ETag
	//
	// Value returned by this function will be set in the response Etag header.
	//
	// If the returned ETag matches the requests `If-None-Match` header
	// the handler call will be skipped and a 304 response will be sent
	//
	// If this function is nil, Butler will instead read the response body and
	// generate a hash to be used as an ETag
	GetEtag func(request *Request, fullFilepath string) string

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

func (e *FsEndpoint) EtagGenerator() func(request *Request) string {
	getEtag := e.GetEtag
	if getEtag == nil {
		return nil
	}

	return func(request *Request) string {
		filepath := request.EchoContext().Param("*")
		fullFilepath := path.Join(e.Dir, filepath)
		return getEtag(request, fullFilepath)
	}
}

func (e *FsEndpoint) Register(parent EndpointParent) {
	if e.parent != nil {
		panic("endpoint can only be registered once")
	}

	e.parent = parent

	if e.Handler == nil {
		e.isCustomHandler = false
		e.Handler = func(
			request *Request,
			fpath string,
			file *os.File,
			fstat os.FileInfo,
		) *Response {
			fmime := Mime.DetectFile(fpath, file)

			var response *Response

			if e.DisableStreaming || fmime == "text/javascript" || fmime == "text/html" ||
				fmime == "text/css" || fmime == "application/json" ||
				fstat.Size() < Units.MB {
				response = Respond.Ok().FileHandle(file, fmime)
			} else {
				response = Respond.Ok().StreamFileHandle(file, fmime)
			}

			return response
		}
	} else {
		e.isCustomHandler = true
	}

	if e.Name == "" {
		e.Name = "Static Files"
	}

	if e.Description == "" {
		e.Description = fmt.Sprintf("Serves static files from the local directory: '%s'", e.Dir)
	}

	registerEndpoint(e, parent)
}

func (e *FsEndpoint) BindParams(*Request) *ParamParsingError {
	return nil
}

func (e *FsEndpoint) ExecuteHandler(ctx *echo.Context, request *Request) (retVal *Response) {
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

	request.Data["_filehandle"] = file
	request.Data["_filestat"] = stat

	resp := e.Handler(request, fullFilepath, file, stat)

	if e.DisableStreaming {
		resp.SetAllowStreaming(false)
	}

	return resp
}

func (e *FsEndpoint) calculateContentLength(fpath string, file *os.File, req *Request) (string, string) {
	var err error
	requestedRange, err := parseRangeHeader(req.Headers)
	if err != nil {
		return "", ""
	}

	contentType := Mime.DetectFile(fpath, file)

	file.Seek(0, 0)
	data, err := io.ReadAll(file)
	file.Seek(0, 0)

	if requestedRange == nil {

		if err != nil {
			return "", ""
		}

		enc := e.GetEncoding()

		if enc == "auto" || enc == "" {
			enc = resolveAutoEncoding(
				contentType,
				req.Headers.Get("Accept-Encoding"),
				data,
			)
		}

		encodedLen := len(data)

		acceptedEncodings := req.Headers.Get("Accept-Encoding")

		var encodedData *bytes.Buffer
		var retEncoding string
		switch enc {
		case "brotli":
			if len(data) >= BROTLI_MIN_SIZE && strings.Contains(acceptedEncodings, "br") {
				encodedData, err = Brotli(data)
				encodedLen = encodedData.Len()
				retEncoding = "brotli"
			}
		case "deflate":
			if len(data) >= DEFLATE_MIN_SIZE && strings.Contains(acceptedEncodings, "deflate") {
				encodedData, err = Deflate(data)
				encodedLen = encodedData.Len()
				retEncoding = "deflate"
			}
		case "gzip":
			if len(data) >= GZIP_MIN_SIZE && strings.Contains(acceptedEncodings, "gzip") {
				encodedData, err = GZip(data)
				encodedLen = encodedData.Len()
				retEncoding = "gzip"
			}
		}

		if err != nil {
			return "", ""
		}

		return strconv.Itoa(encodedLen), retEncoding
	} else {
		dataSize := len(data)
		if !requestedRange.HasEnd {
			requestedRange.End = max(0, dataSize-1)
		}
		endIdx := min(dataSize-1, requestedRange.End)
		requestedLen := endIdx - requestedRange.Start + 1
		contentLength := strconv.FormatInt(int64(requestedLen), 10)
		return contentLength, ""
	}
}

func (e *FsEndpoint) GetHeadHandler() HeadHandler {
	if e.Head == nil {
		if e.isCustomHandler {
			return nil
		}

		return func(ctx *echo.Context, request *Request, status int) *Headers {
			if status != 200 {
				return nil
			}

			filepath := ctx.Param("*")
			fullFilepath := path.Join(e.Dir, filepath)

			var file *os.File
			var fstat fs.FileInfo

			if f, ok := request.Data["_filehandle"]; ok {
				file = f.(*os.File)
			} else {
				f, err := os.Open(fullFilepath)
				if err != nil {
					return nil
				}

				file = f
			}

			if s, ok := request.Data["_filestat"]; ok {
				fstat = s.(fs.FileInfo)
			} else {
				s, err := file.Stat()
				if err != nil {
					return nil
				}

				fstat = s
			}

			if fstat.IsDir() {
				return nil
			}

			headers := NewHeaders().Set("Last-Modified", fstat.ModTime().Format(http.TimeFormat))

			if request.Method == "HEAD" {
				len, enc := e.calculateContentLength(
					filepath, file, request,
				)
				// for get requests length will be set by the response sender
				headers.Set("Content-Length", len)
				if enc != "" {
					headers.Set("Content-Encoding", enc)
				}
			}

			return headers
		}
	}

	return func(ctx *echo.Context, request *Request, status int) *Headers {
		filepath := ctx.Param("*")
		fullFilepath := path.Join(e.Dir, filepath)

		return e.Head(request, fullFilepath, status)
	}
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

func (g *FsEndpoint) GetResponseContentType() string {
	return "application/octet-stream"
}

func (g *FsEndpoint) GetDefaultCachePolicy() *HttpCachePolicy {
	return g.CachePolicy
}

func (g *FsEndpoint) GetDefaultEncoding() string {
	if g.Encoding != "" {
		return g.Encoding
	}
	return "auto"
}
