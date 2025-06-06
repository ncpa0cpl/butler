# Request body

To access the request body you can use the Endpoint struct generic type.

If the Endpoint fails to bind the request body to the specified body type, it will automatically
respond with a 400 status code.


```go
package main

import butler "github.com/ncpa0cpl/http-butler"

type RequestPayload struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	endpoint := &butler.Endpoint[butler.NoParams, RequestPayload]{
		Method: "POST",
		Path: "/entry",
		Handler: func(request *butler.Request, params butler.NoParams, body *RequestPayload) *butler.Response {

			fmt.Println("received a request with a body label:", body.Label)

			return butler.Respond.Ok()
		},
	}

	app.Add(endpoint)
	app.Listen()
}
```
