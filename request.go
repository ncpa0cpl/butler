package httpbutler

import (
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

type Request struct {
	Method  string
	Headers http.Header
	Path    string
	Data    map[string]any

	monitor       monitorRecorder
	monitorRecord RecordBuilder
	ctx           echo.Context
}

func NewRequest(ctx echo.Context, monitor monitorRecorder) *Request {
	path := ctx.Request().URL.Path

	req := &Request{
		ctx:           ctx,
		monitor:       monitor,
		monitorRecord: monitor.CreateRecord(path),
		Path:          path,
		Method:        ctx.Request().Method,
		Data:          map[string]any{},
		Headers:       ctx.Request().Header,
	}

	return req
}

func (r *Request) HttpRequest() *http.Request {
	return r.ctx.Request()
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

func (r *Request) monitorStart(step, name string) {
	r.monitorRecord.StepStart(step, name)
}

func (r *Request) monitorEnd(step, name string) {
	r.monitorRecord.StepEnd(step, name)
}

func (r *Request) completeMonitor() {
	r.monitor.FinalizeRecord(r.monitorRecord)
}
