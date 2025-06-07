package httpbutler

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/carlmjohnson/requests"
	"github.com/labstack/echo/v4"
)

type ProxyRequestOptions struct {
	Method              string
	Headers             map[string][]string
	Body                *[]byte
	DoNotForwardHeaders bool
}

func createProxyHandler(response *Response, url string, opts *ProxyRequestOptions) func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		req := requests.URL(url)

		if opts != nil && opts.Method != "" {
			req.Method(opts.Method)
		} else {
			req.Method(ctx.Request().Method)
		}

		req.Header("X-Forwarded-Host", ctx.Request().Host)

		if opts != nil && !opts.DoNotForwardHeaders {
			for h, v := range ctx.Request().Header {
				req.Header(h, v...)
			}
		}

		if opts != nil && opts.Headers != nil {
			for h, v := range opts.Headers {
				req.Header(h, v...)
			}
		}

		req.Header("X-Forwarded-Proto", ctx.Scheme())
		forwarededFor := ctx.Request().Header.Get("X-Forwarded-For")
		if forwarededFor != "" {
			req.Header("X-Forwarded-For", fmt.Sprintf("%s, %s", forwarededFor, getLocalIP()))
		} else {
			clientIP := ctx.RealIP()
			req.Header("X-Forwarded-For", fmt.Sprintf("%s, %s", clientIP, getLocalIP()))
		}

		if opts != nil && opts.Body != nil {
			req.BodyBytes(*opts.Body)
		} else {
			req.Body(func() (io.ReadCloser, error) {
				return ctx.Request().Body, nil
			})
		}

		for idx := range response.cookies {
			cookie := &response.cookies[idx]
			ctx.SetCookie(cookie)
		}

		respHeaders := ctx.Response().Header()
		respWriter := ctx.Response().Writer

		response.Headers.CopyInto(respHeaders)

		req.AddValidator(func(res *http.Response) error {
			for k, v := range res.Header {
				if !response.Headers.Has(k) {
					respHeaders[k] = v
				}
			}

			return nil
		})
		req.ToWriter(respWriter)
		req.AddValidator(func(res *http.Response) error {
			respWriter.WriteHeader(res.StatusCode)
			return nil
		})

		return req.Fetch(context.Background())
	}
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
