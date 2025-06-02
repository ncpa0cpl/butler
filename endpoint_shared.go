package httpbutler

import (
	"strings"

	echo "github.com/labstack/echo/v4"
)

type AnyEndpoint interface {
	GetPath() string
	GetMethod() string
	GetAuth() AuthHandler
	GetEncoding() string
	ExecuteHandler(ctx echo.Context, request *Request) *Response
	GetCachePolicy() *HttpCachePolicy
}

func registerEndpoint[E AnyEndpoint](e E, parent EndpointParent) {
	echoServer := parent.GetEcho()
	basepath := parent.GetPath()
	middlewares := parent.GetMiddlewares()
	authHandlers := parent.GetAuthHandlers()
	encoding := e.GetEncoding()
	cachePolicy := e.GetCachePolicy()
	fullpath := strings.TrimRight(basepath, "/") + "/" + strings.TrimLeft(e.GetPath(), "/")

	reqMiddlewares := getReqMiddlewares(middlewares)
	respMiddlewares := getRespMiddlewares(middlewares)

	endpAuth := e.GetAuth()
	if endpAuth != nil {
		authHandlers = append(authHandlers, endpAuth)
	}

	encoder := createEncoder(encoding)

	handler := func(ctx echo.Context) error {
		request := newRequest(ctx)

		for _, authHandler := range authHandlers {
			auth := authHandler(request)
			if !auth.IsSuccessful() {
				return auth.SendResponse(ctx)
			}
		}

		var response *Response
		for _, md := range reqMiddlewares {
			err := md.OnRequest(
				request,
				func(nextReq *Request) {
					request = nextReq
				},
				func(sendInstead *Response) {
					response = sendInstead
				},
			)

			if err != nil {
				ctx.Logger().Errorf("middleware %s request handler returned an error", md.Name)
				return err
			}

			if response != nil {
				break
			}
		}

		if response == nil {
			response = e.ExecuteHandler(ctx, request)
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
				ctx.Logger().Errorf("middleware %s response handler returned an error", md.Name)
				return err
			}
		}

		if response == nil {
			ctx.Logger().Errorf("endpoint handler did not return a response [path=%s]", fullpath)
			return ctx.NoContent(500)
		}

		if response.customHandler != nil {
			return response.send(ctx)
		}

		cp := resolveCachePolicy(cachePolicy, response)
		if cp != nil {
			response.Headers.Set("Cache-Control", cp.ToString())

			if !cp.DisableETagGeneration {
				AddEtag(response)
			}

			if !cp.DisableAutoResponse {
				etag := response.Headers.Get("ETag")
				if etag != "" {
					ifNoneMatch := request.Headers.Get("If-None-Match")
					if etag == ifNoneMatch {
						response.Headers.CopyInto(ctx.Response().Header())
						return ctx.NoContent(304)
					}
				}
			}
		}

		encoder(request, response, ctx)

		return response.send(ctx)
	}

	switch e.GetMethod() {
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
	}

	panic("invalid method: " + e.GetMethod())
}

func resolveCachePolicy(endpointPolicy *HttpCachePolicy, response *Response) *HttpCachePolicy {
	if response.Headers.Get("Cache-Control") == "" {
		if response.CachePolicy != nil {
			return response.CachePolicy
			// response.Headers.Set("Cache-Control", response.CachePolicy.ToString())
		} else if endpointPolicy != nil {
			return endpointPolicy
			// response.Headers.Set("Cache-Control", endpointPolicy.ToString())
		}
	}
	return nil
}

func createEncoder(encoding string) func(request *Request, response *Response, ctx echo.Context) {
	if encoding == "gzip" {
		return func(request *Request, response *Response, ctx echo.Context) {
			// if the response already has encoding specified don't do anything
			if response.Encoding != "" || response.Headers.Get("Content-Encoding") != "" {
				return
			}

			acceptedEncodings := request.Headers.Get("Accept-Encoding")
			if strings.Contains(acceptedEncodings, "gzip") {
				data, err := GZip(response.Body)
				if err == nil {
					response.Body = data.Bytes()
					response.Headers.Set("Content-Encoding", "gzip")
				} else {
					ctx.Logger().Error("encountered an error when encoding the response (GZip)")
				}
			}
		}
	}

	return func(request *Request, response *Response, ctx echo.Context) {}
}
