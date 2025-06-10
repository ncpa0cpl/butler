# Getting Started

```go
package main

import butler "github.com/ncpa0cpl/butler"

func main() {
	app := butler.CreateServer()
	app.Port = 8080

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
