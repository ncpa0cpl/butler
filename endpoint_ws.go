package butler

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	echo "github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{}

type WebSocketEndpoint struct {
	Path         string
	Auth         AuthHandler
	MaxMsgSize   int64
	PongTimeout  time.Duration
	PingInterval time.Duration
	WriteTimeout time.Duration
	OnOpen       func(ws *Websocket) error
	// Optional. A function that runs just before the connection is upgraded to a WebSocket,
	// can return a response that will be sent instead of upgrading the connection. To allow upgrade
	// return nil.
	BeforeUpgrade func(request *Request) *Response

	Description string
	Name        string

	parent EndpointParent
}

func (e *WebSocketEndpoint) GetName() string {
	return e.Name
}

func (e *WebSocketEndpoint) GetDescription() string {
	return e.Description
}

func (g *WebSocketEndpoint) GetSubRoutes() []EndpointInterface {
	return []EndpointInterface{}
}

func (e *WebSocketEndpoint) GetPath() string {
	return pathJoin(e.parent.GetPath(), e.Path)
}

func (e *WebSocketEndpoint) GetMethod() string {
	return "GET"
}

func (e *WebSocketEndpoint) GetAuth() AuthHandler {
	return e.Auth
}

func (e *WebSocketEndpoint) GetEncoding() string {
	return "none"
}

func (e *WebSocketEndpoint) GetCachePolicy() *HttpCachePolicy {
	return nil
}

func (e *WebSocketEndpoint) GetStreamingSettings() *StreamingSettings {
	return nil
}

func (e *WebSocketEndpoint) GetMiddlewares() []Middleware {
	return []Middleware{}
}

func (e *WebSocketEndpoint) ExecuteHandler(ctx echo.Context, request *Request) error {
	ws, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		return err
	}

	if e.MaxMsgSize != 0 {
		ws.SetReadLimit(e.MaxMsgSize)
	}

	return e.OnOpen(newWebsocket(request, ws, e.PingInterval, e.PongTimeout, e.WriteTimeout))
}

func (e *WebSocketEndpoint) Register(parent EndpointParent) {
	if e.OnOpen == nil {
		panic("endpoint has no handler")
	}
	if e.parent != nil {
		panic("endpoint can only be registered once")
	}

	e.parent = parent

	monitor := voidRecorder{}

	echoServer := parent.GetEcho()
	middlewares := append(parent.GetMiddlewares(), e.GetMiddlewares()...)
	authHandlers := parent.GetAuthHandlers()

	reqMiddlewares := getReqMiddlewares(middlewares)
	respMiddlewares := getRespMiddlewares(middlewares)

	endpAuth := e.GetAuth()
	if endpAuth != nil {
		authHandlers = append(authHandlers, endpAuth)
	}

	handler := func(ctx echo.Context) error {
		request := NewRequest(ctx, monitor)

		defer func() {
			if r := recover(); r != nil {
				if r == http.ErrAbortHandler {
					panic(r)
				}
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				var stack []byte
				var length int

				stack = make([]byte, Units.MB)
				length = runtime.Stack(stack, true)
				stack = stack[:length]

				msg := fmt.Sprintf("[PANIC RECOVERY] %v %s", err, stack[:length])
				request.Logger.Fatal(msg)

				if err != nil {
					ctx.Error(err)
				} else {
					ctx.NoContent(500)
				}
			}
		}()

		if len(authHandlers) > 0 {
			for _, authHandler := range authHandlers {
				auth := authHandler(request)
				if !auth.IsSuccessful() {
					return auth.SendResponse(request)
				}
			}
		}

		var response *Response
		for _, md := range reqMiddlewares {
			err := md.OnRequest(
				request,
				func(sendInstead *Response) {
					response = sendInstead
				},
			)

			if err != nil {
				request.Logger.Errorf("middleware %s request handler returned an error", md.Name)
				response = Respond.InternalError()
				return response.send(request)
			}

			if response != nil {
				break
			}
		}

		if response == nil && e.BeforeUpgrade != nil {
			r := e.BeforeUpgrade(request)
			if r != nil {
				response = r
			}
		}

		if response == nil {
			return e.ExecuteHandler(ctx, request)
		}

		for _, md := range respMiddlewares {
			err := md.OnResponse(
				request,
				response,
				func(sendInstead *Response) {
					response = sendInstead
				},
			)

			if err != nil {
				request.Logger.Errorf("middleware %s response handler returned an error", md.Name)
				response = Respond.InternalError()
				return response.send(request)
			}
		}

		if response.customHandler != nil {
			return response.send(request)
		}

		return response.send(request)
	}

	echoServer.GET(e.Path, handler)
}

//

func (g *WebSocketEndpoint) GetParamsT() any {
	return nil
}

func (g *WebSocketEndpoint) GetBodyT() any {
	return nil
}

func (g *WebSocketEndpoint) GetResponseT() any {
	return nil
}

func (g *WebSocketEndpoint) GetResponseContentType() string {

	return ""
}

func (g *WebSocketEndpoint) GetDefaultCachePolicy() *HttpCachePolicy {
	return nil
}

func (g *WebSocketEndpoint) GetDefaultEncoding() string {
	return "none"
}
