# Auth Handlers

Each endpoint and group can have a Auth handler that will run before the request is handled. Auth handlers always run before any middleware.

Auth Handler is where you should put logic for identifying the client and check if they have the right to access
the endpoints.

```go
package main

import butler "github.com/ncpa0cpl/http-butler"

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	home := &butler.BasicEndpoint[butler.NoParams]{
		Method: "GET",
		Path: "/user-panel",
		Auth: func(request *butler.Request) *butler.Ath {
			if !isUserAuthorized(request) {
				return butler.Auth.Unauthorized()
			}

			if !userCanAccess(request) {
				return butler.Auth.Forbidden()
			}

			return butler.Auth.Ok()
		},
		Handler: func(request *butler.Request, params butler.NoParams) *butler.Response {
			return butler.Respond.Ok().Text("<Sensitive user data>")
		},
	}

	app.Add(home)
	app.Listen()
}
```
