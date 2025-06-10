# Groups

```go
package main

import butler "github.com/ncpa0cpl/butler"

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	app.Add(&butler.BasicEndpoint[butler.NoParams]{
		Method: "GET",
		Path: "",
		Handler: func(request *butler.Request, params butler.NoParams) *butler.Response {
			return butler.Respond.Ok().Html("<h1>Welcome!</h1>")
		},
	})

	apiGroup := &butler.Group{
		Path: "/api",
	}

	apiGroup.Add(&butler.BasicEndpoint[butler.NoParams]{
		Method: "GET",
		Path: "/users",
		Handler: func(request *butler.Request, params butler.NoParams) *butler.Response {
			return butler.Respond.Ok().JSON([]string{"User1", "User2"})
		},
	})

	app.Add(apiGroup)
	app.Listen()
}
```

The above code would register two endpoint urls:
http://hostname/ - serves the HTML content
http://hostname/api/users - serves the list of users in JSON

## Group middlewares

Groups can have middlewares added to them just like the server root. A middleware added to group will be ran for
every endpoint within the Group.

```go
apiGroup := &butler.Group{
		Path: "/api",
}

apiGroup.Use(MyMiddleware)
```
