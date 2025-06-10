package butler

type Ath struct {
	success  bool
	result   string
	response *Response
}

var Auth Ath

// AuthHandler runs before any middleware or endpoint handlers and can be used to determine if a given request
// is authenticated or has the rights to call given endpoint or group
type AuthHandler func(request *Request) *Ath

// Client is not authenticated with the server (server does not know who the client is)
func (Ath) Unauthorized() *Ath {
	return &Ath{
		success: false,
		result:  "unauthorized",
		response: &Response{
			Status: 401,
		},
	}
}

// Client does not have the rights to access (server knows the client identity but refuses access)
func (Ath) Forbidden() *Ath {
	return &Ath{
		success: false,
		result:  "forbidden",
		response: &Response{
			Status: 403,
		},
	}
}

func (Ath) Ok() *Ath {
	return &Ath{
		success: true,
	}
}

func (a *Ath) IsSuccessful() bool {
	return a.success
}

func (a *Ath) SendResponse(ctx *Request) error {
	return a.response.send(ctx)
}
