package butler

import (
	"fmt"
	"net/http"
	"runtime/debug"

	echo "github.com/labstack/echo/v5"
)

const ENDPOINT_REQ_PARAMS_KEY = "__endpoint_req_params_interface"

type HeadHandler = func(ctx *echo.Context, request *Request, respStatus int) *Headers

type AnyEndpoint interface {
	GetPath() string
	GetMethod() string
	GetAuth() AuthHandler
	GetEncoding() string
	ExecuteHandler(ctx *echo.Context, request *Request) *Response
	GetCachePolicy() *HttpCachePolicy
	GetStreamingSettings() *StreamingSettings
	GetMiddlewares() []Middleware
	BindParams(request *Request) *ParamParsingError
	EtagGenerator() func(request *Request) string
	GetHeadHandler() HeadHandler
}

func registerEndpoint[E AnyEndpoint](e E, parent EndpointParent) {
	server := parent.GetServer()
	monitor := createMonitorRecorder(server)

	echoServer := parent.GetEcho()
	middlewares := append(parent.GetMiddlewares(), e.GetMiddlewares()...)
	authHandlers := parent.GetAuthHandlers()
	defaultEncoding := e.GetEncoding()
	cachePolicy := e.GetCachePolicy()
	streamSettings := e.GetStreamingSettings()
	fullpath := e.GetPath()
	method := e.GetMethod()
	getEtag := e.EtagGenerator()
	getHeaders := e.GetHeadHandler()

	reqMiddlewares := getReqMiddlewares(middlewares)
	respMiddlewares := getRespMiddlewares(middlewares)

	endpAuth := e.GetAuth()
	if endpAuth != nil {
		authHandlers = append(authHandlers, endpAuth)
	}

	if defaultEncoding == "" {
		defaultEncoding = "auto"
	}

	requestMiddlewares := func(request *Request, response *Response) (*Response, bool) {
		respond := func(sendInstead *Response) {
			response = sendInstead
		}

		for _, md := range reqMiddlewares {
			request.monitorStart(MonitorStep.ReqMiddleware, md.Name)
			err := md.OnRequest(request, respond)
			request.monitorEnd(MonitorStep.ReqMiddleware, md.Name)

			if err != nil {
				request.Logger.Errorf("middleware %s request handler returned an error", md.Name)
				return Respond.InternalError(), true
			}

			if response != nil {
				break
			}
		}
		return response, false
	}

	responseMiddlewares := func(request *Request, response *Response) *Response {
		respond := func(sendInstead *Response) {
			response = sendInstead
		}

		for _, md := range respMiddlewares {
			request.monitorStart(MonitorStep.ResMiddleware, md.Name)
			err := md.OnResponse(request, response, respond)
			request.monitorEnd(MonitorStep.ResMiddleware, md.Name)

			if err != nil {
				request.Logger.Errorf("middleware %s response handler returned an error", md.Name)
				return Respond.InternalError()
			}
		}

		return response
	}

	createResponse := func(ctx *echo.Context, request *Request) (result *Response) {
		var response *Response
		var etag string

		if len(authHandlers) > 0 {
			request.monitorStart(MonitorStep.Auth, "")

			for _, authHandler := range authHandlers {
				auth := authHandler(request)
				if !auth.IsSuccessful() {
					return auth.GetResponse()
				}
			}

			request.monitorEnd(MonitorStep.Auth, "")
		}

		request.monitorStart(MonitorStep.BindinParams, "")
		perr := e.BindParams(request)
		if perr != nil {
			request.Logger.Error(perr.ToString())
			return perr.Response()
		}
		request.monitorEnd(MonitorStep.BindinParams, "")

		response, failed := requestMiddlewares(request, response)
		if failed {
			return response
		}

		if response == nil {
			if getEtag != nil {
				etag = getEtag(request)
			}
			if request.Method == "GET" && etag != "" && !cachePolicy.DisableAutoResponseSkipping {
				ifNoneMatch := request.Headers.Get("If-None-Match")

				if etag == ifNoneMatch {
					response = Respond.NotModified()

					if cachePolicy.Vary != nil && len(cachePolicy.Vary) > 0 {
						response.Headers.Set("Vary", cachePolicy.VaryHeader())
					}

					response.Headers.Set("Cache-Control", cachePolicy.CacheControlHeader())
					response.Headers.Set("ETag", etag)

					return response
				}
			}

			request.monitorStart(MonitorStep.Handler, "")
			response = e.ExecuteHandler(ctx, request)
			if response != nil && getHeaders != nil {
				headers := getHeaders(ctx, request, response.Status)
				if headers != nil {
					headers.CopyInto(&response.Headers)
				}
			}
			request.monitorEnd(MonitorStep.Handler, "")
		}

		if response == nil {
			request.Logger.Errorf("endpoint handler did not return a response [path=%s]", fullpath)
			response = Respond.InternalError()
			return response
		}

		if response.customHandler != nil {
			return response
		}

		if response.StreamingSettings == nil {
			response.StreamingSettings = streamSettings
		}

		cp := resolveCachePolicy(cachePolicy, response)
		if cp != nil {
			if cp.Vary != nil && len(cp.Vary) > 0 {
				response.Headers.Set("Vary", cp.VaryHeader())
			}

			if response.Status >= 200 && response.Status < 300 && request.Method == "GET" {
				response.Headers.Set("Cache-Control", cp.CacheControlHeader())

				if etag != "" {
					response.Headers.Set("ETag", etag)
				} else if !cp.DisableETagGeneration {
					request.monitorStart(MonitorStep.EtagHandler, "")
					GenerateAndAddETag(response)
					request.monitorEnd(MonitorStep.EtagHandler, "")
				}

				if !cp.DisableAutoResponseSkipping {
					etag := response.Headers.Get("ETag")
					if etag != "" {
						ifNoneMatch := request.Headers.Get("If-None-Match")
						if etag == ifNoneMatch {
							response.Status = 304
							response.Body = nil
							response.streamReader = nil
							response.streamWriter = nil
							response.customHandler = nil
							return response
						}
					}
				}
			}
		}

		if response.Encoding == "" {
			response.Encoding = defaultEncoding
		}

		return response
	}

	handler := func(ctx *echo.Context) (result error) {
		request := NewRequest(ctx, monitor, parent)
		defer request.completeMonitor()

		defer func() {
			if r := recover(); r != nil {
				if r == http.ErrAbortHandler {
					panic(r)
				}
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				request.Logger.Fatal(
					fmt.Sprintf("[PANIC RECOVERY] %v \n%s", err, debug.Stack()),
				)

				request.saveSessions()

				result = ctx.NoContent(500)
			}
		}()

		resp := createResponse(ctx, request)
		resp = responseMiddlewares(request, resp)
		return resp.send(request)
	}

	headHandler := func(ctx *echo.Context) (result error) {
		request := NewRequest(ctx, monitor, parent)
		defer request.completeMonitor()

		defer func() {
			if r := recover(); r != nil {
				if r == http.ErrAbortHandler {
					panic(r)
				}
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				request.Logger.Fatal(
					fmt.Sprintf("[PANIC RECOVERY] %v \n%s", err, debug.Stack()),
				)

				request.saveSessions()

				result = ctx.NoContent(500)
			}
		}()

		var response *Response

		if len(authHandlers) > 0 {
			request.monitorStart(MonitorStep.Auth, "")

			for _, authHandler := range authHandlers {
				auth := authHandler(request)
				if !auth.IsSuccessful() {
					return auth.GetResponse().send(request)
				}
			}

			request.monitorEnd(MonitorStep.Auth, "")
		}

		request.monitorStart(MonitorStep.BindinParams, "")
		perr := e.BindParams(request)
		if perr != nil {
			request.Logger.Error(perr.ToString())
			return perr.Response().send(request)
		}
		request.monitorEnd(MonitorStep.BindinParams, "")

		response, failed := requestMiddlewares(request, response)
		if failed {
			return response.send(request)
		}

		request.monitorStart(MonitorStep.Handler, "")
		response = Respond.Ok()
		var headers *Headers
		if getHeaders != nil {
			headers = getHeaders(ctx, request, response.Status)
		}
		if headers != nil {
			response.SetHeaders(headers)
		}
		request.monitorEnd(MonitorStep.Handler, "")

		response = responseMiddlewares(request, response)
		return response.send(request)
	}

	switch method {
	case "GET":
		echoServer.GET(fullpath, handler)
		echoServer.HEAD(fullpath, headHandler)
		return
	case "POST":
		echoServer.POST(fullpath, handler)
		return
	case "PUT":
		echoServer.PUT(fullpath, handler)
		return
	case "PATCH":
		echoServer.PATCH(fullpath, handler)
		return
	case "DELETE":
		echoServer.DELETE(fullpath, handler)
		return
	case "OPTIONS":
		echoServer.OPTIONS(fullpath, handler)
		return
	case "HEAD":
		echoServer.HEAD(fullpath, handler)
		return
	case "ANY":
		echoServer.Any(fullpath, handler)
		return
	}

	panic("invalid method: " + e.GetMethod())
}

func resolveCachePolicy(endpointPolicy *HttpCachePolicy, response *Response) *HttpCachePolicy {
	if response.Headers.Get("Cache-Control") == "" {
		if response.CachePolicy != nil {
			return response.CachePolicy
		} else if endpointPolicy != nil {
			return endpointPolicy
		}
	}
	return nil
}
