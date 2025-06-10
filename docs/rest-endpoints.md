# Rest Endpoints

Rest style endpoints can be easily implemented using the `butler.RestEndpoint` struct.

```go
package main

import butler "github.com/ncpa0cpl/butler"

// define a struct that implements the butler.RestResource interface
type Resource struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

type ResourceParams struct {
	ID *f.StringUrlParam // will extract the id from the url for the GET, PUT and DELETE requests
}

// will handle requests to `GET /api/resource/:id`
func (b Resource) Get(req *butler.Request, params ResourceParams) (*Resource, *butler.Response) {
	resourceID := params.ID.Get()
	// return a resource based on the ID
	return &Resource{resourceID, ""}, nil
}

// will handle requests to `GET /api/resource`
func (b Resource) List(req *butler.Request, params ResourceParams) ([]Resource, *butler.Response) {
	return []Resource{}, nil
}

// will handle requests to `POST /api/resource`
func (b Resource) Create(req *f.Request, body *Resource) (*Resource, *f.Response) {
	// save the given body and return a version of it to be sent in response
	return body, nil
}

// will handle requests to `PUT /api/resource/:id`
func (b Resource) Update(req *f.Request, params ResourceParams, body *Resource) (*Resource, *f.Response) {
	resourceID := params.ID.Get()
	// update the resource by using the given ID and the update body, then return the
	// body that dshould be sent in response
	return body, nil
}

// will handle requests to `DELETE /api/resource/:id`
func (b Resource) Delete(req *f.Request, params ResourceParams) *f.Response {
	resourceID := params.ID.Get()
	// delete the resource based on the ID
	return nil
}

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	resourceEndpoints := &butler.RestEndpoint[ResourceParams, Resource]{
		Path: "/api/resource",
		Resource: Resource{}
	}

	app.Add(resourceEndpoints)
	app.Listen()
}
```
