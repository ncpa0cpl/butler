package httpbutler

import (
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

type Request struct {
	ctx     echo.Context
	Method  string
	Headers http.Header
	Data    map[string]any
}

func NewRequest(ctx echo.Context) *Request {
	req := &Request{
		ctx:     ctx,
		Method:  ctx.Request().Method,
		Data:    map[string]any{},
		Headers: ctx.Request().Header,
	}

	return req
}

func (r *Request) Logger() echo.Logger {
	return r.ctx.Logger()
}

func (r *Request) GetCookie(name string) (*http.Cookie, error) {
	return r.ctx.Cookie(name)
}

func (r *Request) FormValue(name string) string {
	return r.ctx.FormValue(name)
}

func (r *Request) FormParams() (url.Values, error) {
	return r.ctx.FormParams()
}

func (r *Request) FormFile(name string) (*multipart.FileHeader, error) {
	return r.ctx.FormFile(name)
}

func (r *Request) MultipartForm() (*multipart.Form, error) {
	return r.ctx.MultipartForm()
}

func (r *Request) EchoContext() echo.Context {
	return r.ctx
}
