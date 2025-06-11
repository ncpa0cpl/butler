package butler

import (
	echo "github.com/labstack/echo/v4"
)

type AnyEndpoint interface {
	GetPath() string
	GetMethod() string
	GetAuth() AuthHandler
	GetEncoding() string
	ExecuteHandler(ctx echo.Context, request *Request) *Response
	GetCachePolicy() *HttpCachePolicy
	GetStreamingSettings() *StreamingSettings
	GetMiddlewares() []Middleware
}

func registerEndpoint[E AnyEndpoint](e E, parent EndpointParent) {
	server := parent.GetServer()
	monitor := createMonitorRecorder(server)

	echoServer := parent.GetEcho()
	basepath := parent.GetPath()
	middlewares := append(parent.GetMiddlewares(), e.GetMiddlewares()...)
	authHandlers := parent.GetAuthHandlers()
	defaultEncoding := e.GetEncoding()
	cachePolicy := e.GetCachePolicy()
	streamSettings := e.GetStreamingSettings()
	fullpath := pathJoin(basepath, e.GetPath())
	method := e.GetMethod()

	reqMiddlewares := getReqMiddlewares(middlewares)
	respMiddlewares := getRespMiddlewares(middlewares)

	endpAuth := e.GetAuth()
	if endpAuth != nil {
		authHandlers = append(authHandlers, endpAuth)
	}

	if defaultEncoding == "" {
		defaultEncoding = "auto"
	}

	handler := func(ctx echo.Context) error {
		request := NewRequest(ctx, monitor)

		if len(authHandlers) > 0 {
			request.monitorStart(MonitorStep.Auth, "")

			for _, authHandler := range authHandlers {
				auth := authHandler(request)
				if !auth.IsSuccessful() {
					return auth.SendResponse(request)
				}
			}

			request.monitorEnd(MonitorStep.Auth, "")
		}

		var response *Response
		for _, md := range reqMiddlewares {
			request.monitorStart(MonitorStep.ReqMiddleware, md.Name)

			err := md.OnRequest(
				request,
				func(sendInstead *Response) {
					response = sendInstead
				},
			)

			request.monitorEnd(MonitorStep.ReqMiddleware, md.Name)

			if err != nil {
				request.Logger.Errorf("middleware %s request handler returned an error", md.Name)
				return err
			}

			if response != nil {
				break
			}
		}

		if response == nil {
			request.monitorStart(MonitorStep.Handler, "")
			response = e.ExecuteHandler(ctx, request)
			request.monitorEnd(MonitorStep.Handler, "")
		}

		for _, md := range respMiddlewares {
			request.monitorStart(MonitorStep.ResMiddleware, md.Name)

			err := md.OnResponse(
				request,
				response,
				func(sendInstead *Response) {
					response = sendInstead
				},
			)

			request.monitorEnd(MonitorStep.ResMiddleware, md.Name)

			if err != nil {
				request.Logger.Errorf("middleware %s response handler returned an error", md.Name)
				return err
			}
		}

		if response == nil {
			request.Logger.Errorf("endpoint handler did not return a response [path=%s]", fullpath)
			return ctx.NoContent(500)
		}

		if response.customHandler != nil {
			return response.send(request)
		}

		if response.StreamingSettings == nil {
			response.StreamingSettings = streamSettings
		}

		if response.Status < 300 && request.Method == "GET" {
			cp := resolveCachePolicy(cachePolicy, response)
			if cp != nil {

				response.Headers.Set("Cache-Control", cp.ToString())

				if !cp.DisableETagGeneration {
					request.monitorStart(MonitorStep.EtagHandler, "")
					AddEtag(response)
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
							return response.send(request)
						}
					}
				}

			}
		}

		if response.Encoding == "" {
			response.Encoding = defaultEncoding
		}

		return response.send(request)
	}

	switch method {
	case "GET":
		echoServer.GET(fullpath, handler)
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
