# Middleware

A middleware can provide two function, one that runs before a request is handled, and one the runs after.

```go
package main

import butler "github.com/ncpa0cpl/butler"

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	loggingMiddleware := butler.Middleware{
		Name: "logging_mdw",
		OnRequest: func(
			request *Request,
			next func(request *Request),
			respond func(response *Response),
		) error {
			myLogger.log("request was received")
			return nil
		},
		OnResponse: func(
			request *Request,
			response *Response,
			next func(response *Response),
		) error {
			myLogger.log("response is being sent")
			return nil
		}
	}

	app.Use(loggingMiddleware)
}
```

## OnRequest

OnRequest function will run for every received request that did not fail auth. It is given
onr callback: `respond()`. It can be used to immediately send a response to the client,
skipping the subsequent middlewares and handlers.

## OnResponse

OnResponse function will run after the request has been created by the Endpoint handler. It is given one function
`next()`. The next function can be used to replace the response object with a different response, that will be then
passed over to the subsequent middlewares. OnResponse can also mutate the given response object.
