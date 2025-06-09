package httpbutler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

type responseLog struct {
	ltype   string
	message string
	cause   error
}

type Response struct {
	Status  int
	Headers Headers
	Body    []byte
	// One of: `auto`, `none`, `gzip`, `brotli`, `deflate`
	//
	// Default: `auto`
	Encoding          string
	CachePolicy       *HttpCachePolicy
	AllowStreaming    bool
	StreamingSettings *StreamingSettings
	customHandler     func(ctx echo.Context) error
	cookies           []http.Cookie
	etag              string
	logs              []responseLog
	streamReader      ButlerReader
	streamWriter      func(HttpWriter) error
}

// marks this response to be encoded with a given encoding (one of: `auto`, `none`, `gzip`, `brotli`, `deflate`)
//
// encoding happens as the last step before sending a response,
// middlewares run before the content gets encoded.
//
// encoding will not occur if the client does not accept the given encoding
// or if the Content-Encoding header is set by the endpoint handler or middleware
//
// this encoding setting takes priority over the one defined in the Endpoint
//
// set to empty string to disable encoding
func (resp *Response) SetEncoding(encoding string) *Response {
	resp.Encoding = encoding
	return resp
}

// Set if the responsebody can be streamed in chunks.
//
// A response will be automatically streamed if the request contains a Range header.
// This setting applies to all types of response, except the explicit Respond.Stream()
// which always sends the response as a stream.
func (resp *Response) SetAllowStreaming(canStream bool) *Response {
	resp.AllowStreaming = canStream
	return resp
}

// Sets the etag to include in the response, when set to non empty string, the default
// etag generation will be skipped.
func (resp *Response) Etag(etag string) *Response {
	resp.etag = etag
	return resp
}

// this policy setting takes priority over the one defined in the Endpoint
func (resp *Response) SetCachePolicy(policy *HttpCachePolicy) *Response {
	resp.CachePolicy = policy
	return resp
}

// replaces all the headers of this response
func (resp *Response) SetHeaders(headers Headers) *Response {
	resp.Headers = headers
	return resp
}

// Adds a cookie to be sent along the response
func (resp *Response) SetCookie(cookie *http.Cookie) *Response {
	resp.cookies = append(resp.cookies, *cookie)
	return resp
}

func (resp *Response) DeleteCookie(name string) *Response {
	cookie := http.Cookie{
		Name:  name,
		Value: "v",
		// Setting a time in the distant past, like the unix epoch, removes the cookie,
		// since it has long expired.
		Expires: time.Unix(0, 0),
	}
	resp.cookies = append(resp.cookies, cookie)
	return resp
}

// serializes the given argument using JSON and assigns it to the response body, changes the response content-type
func (resp *Response) JSON(data any) *Response {
	bytes, err := json.Marshal(data)

	if err != nil {
		resp.Status = 500
		resp.logs = append(resp.logs, responseLog{"error", "encountered an error when serializing to JSON", err})
	} else {
		resp.Body = bytes
		resp.Headers.Set("Content-Type", "application/json; charset=utf-8")
	}

	return resp
}

// assigns given argument to the response body, changes the response content-type
func (resp *Response) Text(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/plain")
	return resp
}

// assigns given argument to the response body, changes the response content-type
func (resp *Response) Html(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/html")
	return resp
}

// assigns given argument to the response body, changes the response content-type
func (resp *Response) Css(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/css")
	return resp
}

// assigns given argument to the response body, changes the response content-type
func (resp *Response) Script(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/javascript")
	return resp
}

// assigns given argument to the response body, changes the response content-type
func (resp *Response) XML(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/xml")
	return resp
}

// send the given byte slice with a `application/octet-stream` content type
func (resp *Response) OctetStream(data []byte) *Response {
	resp.Body = data
	resp.Headers.Set("Content-Type", "application/octet-stream")
	return resp
}

// send the given byte slice, automatically detect the data `Content-Type`
func (resp *Response) Blob(data []byte) *Response {
	resp.Body = data
	contentType := http.DetectContentType(data)
	resp.Headers.Set("Content-Type", contentType)
	return resp
}

// send the given byte slice with the specified `contentType`, if `contentType` argument
// is not specified in the arguments it will not be set
func (resp *Response) Bytes(data []byte, contentType ...string) *Response {
	resp.Body = data
	if len(contentType) > 0 {
		resp.Headers.Set("Content-Type", contentType[len(contentType)-1])
	}
	return resp
}

// send the given file with the specified `contentType`, if `contentType` argument
// is not specified it will be detected automatically
func (resp *Response) File(filepath string, contentType ...string) *Response {
	data, err := os.ReadFile(filepath)

	if err != nil {
		resp.Status = 500
		resp.logs = append(resp.logs, responseLog{"error", "unable to read the given file", err})
	} else {
		resp.Body = data

		if len(contentType) > 0 {
			resp.Headers.Set("Content-Type", contentType[len(contentType)-1])
		} else {
			resp.Headers.Set("Content-Type", http.DetectContentType(data))
		}
	}

	return resp
}

// send the given file with the specified `contentType`, if `contentType` argument
// is not specified it will be detected automatically
//
// Call to this function will close the given `filehandle`
func (resp *Response) FileHandle(filehandle *os.File, contentType ...string) *Response {
	filehandle.Seek(0, 0)
	data, err := io.ReadAll(filehandle)
	filehandle.Close()

	if err != nil {
		resp.Status = 500
		resp.logs = append(resp.logs, responseLog{"error", "unable to read the given file", err})
	} else {
		resp.Body = data

		if len(contentType) > 0 {
			resp.Headers.Set("Content-Type", contentType[len(contentType)-1])
		} else {
			resp.Headers.Set("Content-Type", http.DetectContentType(data))
		}
	}

	return resp
}

// sends the data in the given reader in chunks, respects the requests Range header
//
// note: when streaming auto etag generation will be disabled
func (resp *Response) Stream(reader ButlerReader, contentType string) *Response {
	resp.Body = nil

	resp.streamReader = reader
	resp.Headers.Set("Content-Type", contentType)

	return resp
}

// sends the given byte array in chunks, respects the requests Range header
//
// note: when streaming auto etag generation will be disabled
func (resp *Response) StreamBytes(data []byte, contentType string) *Response {
	resp.Body = nil

	resp.streamReader = NewBytesReader(data)
	resp.Headers.Set("Content-Type", contentType)

	return resp
}

// sends the data in the given file in chunks, respects the requests Range header
//
// note: when streaming auto etag generation will be disabled
func (resp *Response) StreamFile(filepath string, contentType string) *Response {
	resp.Body = nil

	file, err := os.Open(filepath)
	if err != nil {
		resp.Status = 500
		resp.logs = append(resp.logs, responseLog{"error", "failed to open file " + filepath, err})
		return resp
	}

	return resp.StreamFileHandle(file, contentType)
}

// sends the data in the given file in chunks, respects the requests Range header
//
// Call to this function will close the given `filehandle`
//
// note: when streaming auto etag generation will be disabled
func (resp *Response) StreamFileHandle(filehandle *os.File, contentType string) *Response {
	resp.Body = nil

	var err error
	resp.streamReader, err = NewFileReader(filehandle)
	if err != nil {
		resp.Status = 500
		resp.logs = append(resp.logs, responseLog{"error", "failed to create file reader " + filehandle.Name(), err})
		return resp
	}

	resp.Headers.Set("Content-Type", contentType)

	return resp
}

/*
Send a response in chunks through a writer

Each write to the writer will flush the written data, allowing the client to receive and process parts of
the response while the server is still generating the rest of the response.

It is safe to write to the given writer in parallel from different go routines.

If a write is called after the client closes the connection, write will return false.

@example

	Respond.Ok().StreamWriter(func (w HttpWriter) error {
	    var wg sync.WaitGroup

		wg.Add(1)
		go getPartOfTheResponseDataThen(func(data []byte) {
		    w.Write(data)
		    wg.Done()
	    })

		wg.Add(1)
		go getAnotherPartOfTheResponseDataThen(func(data []byte) {
		    w.Write(data)
			wg.Done()
		})

		wg.Wait()
		return nil
	})
*/
func (resp *Response) StreamWriter(handler func(HttpWriter) error) *Response {
	resp.Body = nil
	resp.streamWriter = handler
	return resp
}

func (resp *Response) send(request *Request) error {
	ctx := request.EchoContext()

	if resp.customHandler != nil {
		return resp.customHandler(ctx)
	}

	if resp.AllowStreaming {
		resp.Headers.Set("Accept-Ranges", "bytes")
	}

	for idx := range resp.cookies {
		cookie := &resp.cookies[idx]
		ctx.SetCookie(cookie)
	}

	encodeErr := resp.encodeBody(request)

	if encodeErr != nil {
		ctx.Logger().Error(encodeErr)
	}

	resp.Headers.CopyInto(ctx.Response().Header())

	if resp.streamWriter != nil {
		return resp.streamFromWriter(ctx, resp.streamWriter)
	}

	if resp.streamReader != nil {
		return resp.streamFromReader(ctx, request)
	}

	if resp.Body != nil && len(resp.Body) > 0 {
		if resp.shouldAutoStream(request) {
			return resp.stream(ctx, request)
		} else {
			return ctx.Blob(resp.Status, resp.Headers.Get("Content-Type"), resp.Body)
		}
	}

	return ctx.NoContent(resp.Status)
}

func (resp *Response) encodeBody(request *Request) error {
	enc := resp.Encoding

	if enc == "auto" {
		enc = resolveAutoEncoding(request, resp)
	}

	if enc == "none" {
		return nil
	}

	switch enc {
	case "brotli":
		return EncodeRequestBrotli(request, resp)
	case "deflate":
		return EncodeRequestDeflate(request, resp)
	case "gzip":
		return EncodeRequestGzip(request, resp)
	}

	return nil
}

func (resp *Response) shouldAutoStream(request *Request) bool {
	return resp.AllowStreaming &&
		resp.Status < 300 &&
		(request.Headers.Get("Range") != "" || len(resp.Body) >= int(10*Units.MB))
}

type resp struct{}

var Respond resp

// Creates a Response with a custom handler, when a custom handler is used the Response body, status, headers, cookies
// etc. that are set in the butler.Response will not be added to the echo.Context. You have to add those directly to the
// echo.Context yourself.
func (resp) Handler(customHandler func(ctx echo.Context) error) *Response {
	return &Response{
		customHandler: customHandler,
	}
}

// Redirects the request to a different server.
//
// By default the method, body and headers are reused from the current request. Those can be changed by passing
// a ProxyRequestOptions as a second argument.
//
// You can add headers and cookies to the Proxy Response. Status, body, encoding and all other options
// will not be applied as those are decided by the called server.
func (resp) Proxy(url string, options ...ProxyRequestOptions) *Response {
	resp := &Response{}

	var opts *ProxyRequestOptions
	if len(options) > 0 {
		opts = &options[0]
	}

	resp.customHandler = createProxyHandler(resp, url, opts)

	return resp
}

// HTTP Code: 200
func (resp) Ok() *Response {
	return &Response{
		Status: 200,
	}
}

// HTTP Code: 201
func (resp) Created() *Response {
	return &Response{
		Status: 201,
	}
}

// HTTP Code: 202
func (resp) Accepted() *Response {
	return &Response{
		Status: 202,
	}
}

// HTTP Code: 203
func (resp) NonAuthoritativeInformation() *Response {
	return &Response{
		Status: 203,
	}
}

// HTTP Code: 204
func (resp) NoContent() *Response {
	return &Response{
		Status: 204,
	}
}

// HTTP Code: 205
func (resp) ResetContent() *Response {
	return &Response{
		Status: 205,
	}
}

// HTTP Code: 206
func (resp) PartialContent() *Response {
	return &Response{
		Status: 206,
	}
}

// HTTP Code: 300
func (resp) MultipleChoices() *Response {
	return &Response{
		Status: 300,
	}
}

// HTTP Code: 301
func (resp) MovedPermanently() *Response {
	return &Response{
		Status: 301,
	}
}

// HTTP Code: 302
func (resp) Found() *Response {
	return &Response{
		Status: 302,
	}
}

// HTTP Code: 303
func (resp) SeeOther() *Response {
	return &Response{
		Status: 303,
	}
}

// HTTP Code: 304
func (resp) NotModified() *Response {
	return &Response{
		Status: 304,
	}
}

// HTTP Code: 307
func (resp) TemporaryRedirect() *Response {
	return &Response{
		Status: 307,
	}
}

// HTTP Code: 308
func (resp) PermanentRedirect() *Response {
	return &Response{
		Status: 308,
	}
}

// HTTP Code: 400
func (resp) BadRequest() *Response {
	return &Response{
		Status: 400,
	}
}

// HTTP Code: 401
func (resp) Unauthorized() *Response {
	return &Response{
		Status: 401,
	}
}

// HTTP Code: 402
func (resp) PaymentRequired() *Response {
	return &Response{
		Status: 402,
	}
}

// HTTP Code: 403
func (resp) Forbidden() *Response {
	return &Response{
		Status: 403,
	}
}

// HTTP Code: 404
func (resp) NotFound() *Response {
	return &Response{
		Status: 404,
	}
}

// HTTP Code: 405
func (resp) MethodNotAllowed() *Response {
	return &Response{
		Status: 405,
	}
}

// HTTP Code: 406
func (resp) NotAcceptable() *Response {
	return &Response{
		Status: 406,
	}
}

// HTTP Code: 407
func (resp) ProxyAuthRequired() *Response {
	return &Response{
		Status: 407,
	}
}

// HTTP Code: 408
func (resp) RequestTimeout() *Response {
	return &Response{
		Status: 408,
	}
}

// HTTP Code: 409
func (resp) Conflict() *Response {
	return &Response{
		Status: 409,
	}
}

// HTTP Code: 410
func (resp) Gone() *Response {
	return &Response{
		Status: 410,
	}
}

// HTTP Code: 411
func (resp) LengthRequired() *Response {
	return &Response{
		Status: 411,
	}
}

// HTTP Code: 412
func (resp) PreconditionFailed() *Response {
	return &Response{
		Status: 412,
	}
}

// HTTP Code: 413
func (resp) ContentTooLarge() *Response {
	return &Response{
		Status: 413,
	}
}

// HTTP Code: 414
func (resp) UriTooLong() *Response {
	return &Response{
		Status: 414,
	}
}

// HTTP Code: 415
func (resp) UnsupportedMediaType() *Response {
	return &Response{
		Status: 415,
	}
}

// HTTP Code: 416
func (resp) RangeNotSatisfiable() *Response {
	return &Response{
		Status: 416,
	}
}

// HTTP Code: 417
func (resp) ExpectationFailed() *Response {
	return &Response{
		Status: 417,
	}
}

// HTTP Code: 421
func (resp) MisdirectedRequest() *Response {
	return &Response{
		Status: 421,
	}
}

// HTTP Code: 426
func (resp) UpgradeRequired() *Response {
	return &Response{
		Status: 426,
	}
}

// HTTP Code: 428
func (resp) PrecodnitionRequired() *Response {
	return &Response{
		Status: 428,
	}
}

// HTTP Code: 429
func (resp) TooManyRequest() *Response {
	return &Response{
		Status: 429,
	}
}

// HTTP Code: 431
func (resp) HeadersTooLarge() *Response {
	return &Response{
		Status: 431,
	}
}

// HTTP Code: 451
func (resp) UnavailableForLegalReasons() *Response {
	return &Response{
		Status: 451,
	}
}

// HTTP Code: 500
func (resp) InternalError() *Response {
	return &Response{
		Status: 500,
	}
}

// HTTP Code: 501
func (resp) NotImplemented() *Response {
	return &Response{
		Status: 501,
	}
}

// HTTP Code: 502
func (resp) BadGateway() *Response {
	return &Response{
		Status: 502,
	}
}

// HTTP Code: 503
func (resp) ServiceUnavailable() *Response {
	return &Response{
		Status: 503,
	}
}

// HTTP Code: 504
func (resp) GatewayTimeout() *Response {
	return &Response{
		Status: 504,
	}
}

// HTTP Code: 505
func (resp) HttpVersionNotSupported() *Response {
	return &Response{
		Status: 505,
	}
}

// HTTP Code: 506
func (resp) VariantAlsoNegotiates() *Response {
	return &Response{
		Status: 506,
	}
}

// HTTP Code: 510
func (resp) NotExtended() *Response {
	return &Response{
		Status: 510,
	}
}

// HTTP Code: 511
func (resp) NetworkAuthRequired() *Response {
	return &Response{
		Status: 511,
	}
}
