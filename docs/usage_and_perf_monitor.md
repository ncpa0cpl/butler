# Usage and Performance monitoring

Usage and perofrmance of the http butler server can be done easily by providing a monitor interface to the butler.Server.

```go
package main

import butler "github.com/ncpa0cpl/http-butler"

type MyMonitor struct {}

func (MyMonitor) Record(entry *butler.UsageRecord) {
	// here you can save or send to external service the usage record log

	entry.UrlPath // path of the requested url that's related to this record
	entry.Start // timestamp of when the request was received
	entry.End // timestamp of when the response was completed
	entry.Steps // info on each step that was taken to resolve this request (steps can be for example: auth handlers processing, middleware processing, endpoint handler, auto etag generation or encoding the response data)
}

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	app.Monitor(MyMonitor{})

	home := &butler.BasicEndpoint[butler.NoParams]{
		Method: "GET",
		Path: "",
		Handler: func(request *butler.Request, params butler.NoParams) *butler.Response {
			return butler.Respond.Ok().Text("Welcome!")
		},
	}

	app.Add(home)
	app.Listen()
}
```
