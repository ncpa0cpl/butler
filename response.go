package httpbutler

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Status        int
	Headers       Headers
	Body          []byte
	Encoding      string
	CachePolicy   *HttpCachePolicy
	customHandler func(ctx echo.Context) error
	cookies       []http.Cookie
	etag          string
	logs          []string
}

// marks this response to be encoded with a given encoding (gzip)
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
func (resp *Response) Encode(encoding string) *Response {
	resp.Encoding = encoding
	return resp
}

// Sets the etag to include in the response, when set to non empty string, the default etag generation will be skipped.
func (resp *Response) Etag(etag string) *Response {
	resp.etag = etag
	return resp
}

// this policy setting takes priority over the one defined in the Endpoint
func (resp *Response) SetCachePolicy(policy *HttpCachePolicy) *Response {
	resp.CachePolicy = policy
	return resp
}

func (resp *Response) SetHeaders(headers Headers) *Response {
	resp.Headers = headers
	return resp
}

// Adds a cookie to be sent along the response
func (resp *Response) SetCookie(cookie *http.Cookie) *Response {
	resp.cookies = append(resp.cookies, *cookie)
	return resp
}

func (resp *Response) JSON(data any) *Response {
	bytes, err := json.Marshal(data)

	if err != nil {
		resp.Status = 500
		resp.logs = append(resp.logs, "encountered an error when serializing to JSON")
	} else {
		resp.Body = bytes
		resp.Headers.Set("Content-Type", "application/json; charset=utf-8")
	}

	return resp
}

func (resp *Response) Text(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/plain")
	return resp
}

func (resp *Response) Html(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/html")
	return resp
}

func (resp *Response) Css(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/css")
	return resp
}

func (resp *Response) Script(data string) *Response {
	byte := []byte(data)
	resp.Body = byte
	resp.Headers.Set("Content-Type", "text/javascript")
	return resp
}

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

// send the given byte slice, automatically detect the data `Conent-Type`
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
		resp.logs = append(resp.logs, "unable to read the given file")
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

func (resp *Response) send(ctx echo.Context) error {
	if resp.customHandler != nil {
		return resp.customHandler(ctx)
	}

	for idx := range resp.cookies {
		cookie := &resp.cookies[idx]
		ctx.SetCookie(cookie)
	}

	resp.encodeBody(ctx)
	resp.Headers.CopyInto(ctx.Response().Header())

	if len(resp.Body) > 0 {
		return ctx.Blob(resp.Status, resp.Headers.Get("Content-Type"), resp.Body)
	} else {
		return ctx.NoContent(resp.Status)
	}
}

func (resp *Response) encodeBody(ctx echo.Context) {
	if len(resp.Body) > 0 && resp.Encoding == "gzip" && resp.Headers.Get("Content-Encoding") == "" {
		acceptedEncodings := ctx.Request().Header.Get("Accept-Encoding")
		if strings.Contains(acceptedEncodings, "gzip") {
			data, err := GZip(resp.Body)
			if err == nil {
				resp.Body = data.Bytes()
				resp.Headers.Set("Content-Encoding", "gzip")
			} else {
				ctx.Logger().Error("encountered an error when encoding the response (GZip)")
			}
		}
	}
}

type resp struct{}

var Respond resp

// Creates a Response with a custom handler, when a custom handler is used the Response body, status, headers, cookies
// etc. will not be added to the response
func (resp) Handler(customHandler func(ctx echo.Context) error) *Response {
	return &Response{
		customHandler: customHandler,
	}
}

// func (resp) Stream(streamFn func()) *Response {
// 	return &Response{
// 		customHandler: func(ctx echo.Context) error {
// 			return nil
// 		},
// 	}
// }

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
func (resp) Lengthrequired() *Response {
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
func (resp) UnavilableForLegalReasons() *Response {
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
func (resp) NetowrkAuthRequired() *Response {
	return &Response{
		Status: 511,
	}
}
