package httpbutler_test

import (
	"strings"
	"testing"
	"time"

	"slices"

	f "github.com/ncpa0cpl/http-butler"
	"github.com/stretchr/testify/assert"
)

var bookStore []BookResource = []BookResource{}

type BookResource struct {
	ID    string
	Title string
	Pages uint
}

type BooksQueryParams struct {
	ID     *f.StringUrlParam
	Filter *f.StringQParam
}

func (b BookResource) Get(req *f.Request, params BooksQueryParams) (*BookResource, *f.Response) {
	id := params.ID.Get()

	for _, book := range bookStore {
		if book.ID == id {
			return &book, nil
		}
	}

	return nil, nil
}

func (b BookResource) List(req *f.Request, params BooksQueryParams) ([]BookResource, *f.Response) {
	if params.Filter.Has() {
		filtered := make([]BookResource, 0, len(bookStore))
		for _, book := range bookStore {
			if strings.Contains(book.Title, params.Filter.Get()) {
				filtered = append(filtered, book)
			}
		}
		return filtered, nil
	}

	return bookStore, nil
}

func (b BookResource) Create(req *f.Request, body *BookResource) (*BookResource, *f.Response) {
	bookStore = append(bookStore, *body)
	return body, nil
}

func (b BookResource) Update(req *f.Request, params BooksQueryParams, body *BookResource) (*BookResource, *f.Response) {
	id := params.ID.Get()

	for idx := range bookStore {
		book := &bookStore[idx]
		if book.ID == id {
			book.Title = body.Title
			book.Pages = body.Pages
			return book, nil
		}
	}

	return nil, nil
}

func (b BookResource) Delete(req *f.Request, params BooksQueryParams) *f.Response {
	id := params.ID.Get()

	for idx, book := range bookStore {
		if book.ID == id {
			bookStore = slices.Delete(bookStore, idx, idx+1)
			return nil
		}
	}

	return nil
}

func TestRestEndpointAdd(t *testing.T) {
	assert := assert.New(t)

	server := &TestServer{}

	end := &f.RestEndpoints[BooksQueryParams, BookResource]{
		Path:     "/books",
		Encoding: "gzip",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Resource: BookResource{},
	}

	end.Register(server)

	assert.Equal(2, len(server.getRoutes))
	assert.Equal(1, len(server.postRoutes))
	assert.Equal(1, len(server.putRoutes))
	assert.Equal(1, len(server.deleteRoutes))

	// GET
	assert.Equal("/books/:id", server.getRoutes[0].path)
	assert.Equal("/books", server.getRoutes[1].path)

	// POST
	assert.Equal("/books", server.postRoutes[0].path)

	// PUT
	assert.Equal("/books/:id", server.putRoutes[0].path)

	// DELETE
	assert.Equal("/books/:id", server.deleteRoutes[0].path)
}
