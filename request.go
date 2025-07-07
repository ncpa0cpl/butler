package butler

import (
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type Request struct {
	Method  string
	Headers http.Header
	Path    string
	Data    map[string]any
	Logger  RequestLogger

	monitor          monitorRecorder
	monitorRecord    RecordBuilder
	ctx              echo.Context
	accessedSessions []*sessions.Session
	parent           EndpointParent
}

func NewRequest(
	ctx echo.Context,
	monitor monitorRecorder,
	parent EndpointParent,
) *Request {
	path := ctx.Request().URL.Path
	method := ctx.Request().Method

	req := &Request{
		ctx:           ctx,
		monitor:       monitor,
		monitorRecord: monitor.CreateRecord(method, ctx.Path(), path),
		Path:          path,
		Method:        method,
		Data:          map[string]any{},
		Headers:       ctx.Request().Header,
		parent:        parent,
	}

	req.Logger = newRequestLogger(req, ctx.Logger())

	return req
}

// Get the session, requires a session store to be configured.
// Session will be automatically saved before sending the response
//
// Default session name: "session"
func (r *Request) Session(sessionName ...string) (*sessions.Session, error) {
	name := firstOr(sessionName, "session")
	s, err := session.Get(name, r.ctx)
	if err != nil {
		return nil, err
	}

	r.accessedSessions = append(r.accessedSessions, s)

	return s, err
}

func (r *Request) HttpRequest() *http.Request {
	return r.ctx.Request()
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

// Returns the object to which the params were bound to by the endpoint handler
// (for example the instance of the T type from a `BasicEndpoint[T]`)
func (r *Request) GetParamsInterface() any {
	return r.Data[ENDPOINT_REQ_PARAMS_KEY]
}

func (r *Request) setParamsInterface(params any) {
	r.Data[ENDPOINT_REQ_PARAMS_KEY] = params
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

func (r *Request) saveSessions() {
	for _, s := range r.accessedSessions {
		err := s.Save(r.ctx.Request(), r.ctx.Response())
		if err != nil {
			r.Logger.Error("failed to save the session: ", err)
		}
	}
}
