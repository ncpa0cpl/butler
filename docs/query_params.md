# Query Parameters

Http Butler uses fully typed Query Parameters, it will automatically parse the query and url params from the request
and provide them to your handler or fail the request with a 400 if the parsing was unsuccessful.

```go
package main

import butler "github.com/ncpa0cpl/butler"

type SearchParams struct {
	Query *butler.StringQParam
	Limit *butler.NumberQParam
}

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	searchEndpoint := &butler.BasicEndpoint[SearchParams]{
		Method: "GET",
		Path: "/search",
		Handler: func(request *butler.Request, params SearchParams) *butler.Response {
			// check if the query param was present in the URL
			if !params.Query.Has() {
				return butler.Respond.BadRequest().Text("`query` search param was not provided")
			}

			query := params.Query.Get()
			limit := params.Limit.Get(int64(10)) // use limit of 10 if it is not set

			result := yourSearchingFunction(query, limit)

			return butler.Respond.Ok().JSON(result)
		}
	}

	app.Add(searchEndpoint)
	app.Listen()
}
```

## Path parameters

Path parameters can be accessed similarly to the query params.


```go
package main

import butler "github.com/ncpa0cpl/butler"

type UserParams struct {
	ID *butler.NumberUrlParam
}

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	userEndpoint := &butler.BasicEndpoint[UserParams]{
		Method: "GET",
		Path: "/user/:id",
		Handler: func(request *butler.Request, params UserParams) *butler.Response {
			userID := params.ID.Get()

			user := yourFindUserFunction(userID)

			return butler.Respond.Ok().JSON(user)
		},
	}

	app.Add(userEndpoint)
	app.Listen()
}
```

## Parameter names

Names of the parameters by default are the same as the struct field names, but with the first letter in lowercase (ex. "Limit" -> "limit").

Param names can be overridden with a "name" tag like so:

```go
type UserParams struct {
	QueryLimit *butler.StringQParam `name:"limit"`
}
```

## Default Parameters

1. `butler.StringQParam` - string
2. `butler.StringListQParam` - []string
3. `butler.NumberQParam` - int64
4. `butler.NumberListQParam` - []int64
5. `butler.FloatQParam` - float64
6. `butler.FloatListQParam` - []float64
7. `butler.BoolQParam` - bool
8. `butler.StringUrlParam` - string
9. `butler.NumberUrlParam` - int64
10. `butler.FloatUrlParam` - float64
11. `butler.BoolUrlParam` - bool

## Custom Parameters

It's possible to implement custom parameters. It needs to implement the following interface:

```go
type SearchQParam interface {
	Init(req *Request, name string) *ParamParsingError
}
```

### Example

```go
type MyCustomParam struct {
	Value string
}

func (p *MyCustomParam) Init(ctx *Request, name string) *ParamParsingError {
	v := ctx.EchoContext().QueryParam(name)
	p.Value = v
	return nil
}
```
