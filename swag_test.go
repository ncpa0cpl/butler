package butler_test

import (
	f "github.com/ncpa0cpl/butler"
)

type SwagParams struct {
	Limit *f.NumberQParam
}

// func TestSwag(t *testing.T) {
// 	server := f.CreateServer()
// 	server.Port = 8080

// 	authors := &f.BasicEndpoint[SwagParams]{
// 		Method: "GET",
// 		Path:   "/authors",
// 		CachePolicy: &f.HttpCachePolicy{
// 			MaxAge: time.Hour,
// 		},
// 		Handler: func(request *f.Request, params SwagParams) *f.Response {
// 			return f.Respond.Ok()
// 		},
// 		Name:         "Authors Endpoint",
// 		Description:  "Sends a list of all authors",
// 		ResponseType: []Book{},
// 	}

// 	books := &f.BasicEndpoint[f.NoParams]{
// 		Method: "GET",
// 		Path:   "/publishers",
// 		CachePolicy: &f.HttpCachePolicy{
// 			MaxAge: time.Hour,
// 		},
// 		Handler: func(request *f.Request, params f.NoParams) *f.Response {
// 			return f.Respond.Ok()
// 		},
// 		Name:        "Books Publishers Endpoint",
// 		Description: "Sends a list of all book publishers",
// 	}

// 	restEndp := &f.RestEndpoints[BooksQueryParams, BookResource]{
// 		Path:     "/books",
// 		Encoding: "auto",
// 		Resource: BookResource{},
// 	}

// 	server.Add(authors)
// 	server.Add(books)
// 	server.Add(restEndp)

// 	f.AddApiDocumentationRoute("/api_docs", server)

// 	server.Listen()
// 	defer server.Close()
// }
