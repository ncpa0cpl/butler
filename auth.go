package httpbutler

import "github.com/labstack/echo/v4"

type auth struct {
	success  bool
	result   string
	response *Response
}

var Auth auth

// AuthHandler runs before any middleware or endpoint handlers and can be used to determine if a given request
// is authenticated or has the rights to call given endpoint or group
type AuthHandler func(request *Request) *auth

// Client is not authenticated with the server (server does not know who the client is)
func (auth) Unauthorized() *auth {
	return &auth{
		success: false,
		result:  "unauthorized",
		response: &Response{
			Status: 401,
		},
	}
}

// Client does not have the rights to access (server knows the client identity but refuses access)
func (auth) Forbidden() *auth {
	return &auth{
		success: false,
		result:  "forbidden",
		response: &Response{
			Status: 403,
		},
	}
}

func (auth) Ok() *auth {
	return &auth{
		success: true,
	}
}

func (a *auth) IsSuccessful() bool {
	return a.success
}

func (a *auth) SendResponse(ctx echo.Context) error {
	return a.response.send(ctx)
}
