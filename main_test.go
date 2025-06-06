package httpbutler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	f "github.com/ncpa0cpl/http-butler"
	"github.com/stretchr/testify/assert"
)

func TestGetEndpointWithQueryParams(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	books := &f.BasicEndpoint[QParams]{
		Method: "GET",
		Path:   "/books",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params QParams) *f.Response {
			search := params.Search.Get()
			limit := params.Limit.Get()
			del := params.IncludeDel.Get()

			return f.Respond.Ok().JSON([]Book{
				{
					Title: search,
				},
				{
					Title: fmt.Sprintf("%v", limit),
				},
				{
					Title: fmt.Sprintf("%v", del),
				},
			})
		},
	}

	server.Add(books)

	go server.Listen()
	defer server.Close()

	resp, err := http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	cp := resp.Header.Get("Content-Type")
	assert.Equal("application/json; charset=utf-8", cp)

	cacheControl := resp.Header.Get("Cache-Control")
	assert.Equal("public, max-age=3600", cacheControl)

	etag := resp.Header.Get("ETag")
	assert.NotZero(etag)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("[{\"Title\":\"\"},{\"Title\":\"0\"},{\"Title\":\"false\"}]", string(body))

	resp, err = http.Get("http://localhost:8080/books?search=fooqux&limit=2&includedel=true")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	cp = resp.Header.Get("Content-Type")
	assert.Equal("application/json; charset=utf-8", cp)

	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("[{\"Title\":\"fooqux\"},{\"Title\":\"2\"},{\"Title\":\"true\"}]", string(body))

	resp, err = http.Get("http://localhost:8080/books?search=fooqux&limit=a&includedel=true")
	assert.NoError(err)
	assert.Equal(400, resp.StatusCode)
}

func TestGetEndpointWithUrlParams(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	type UrlParams struct {
		ID   *f.StringUrlParam
		Page *f.NumberUrlParam
	}

	books := &f.BasicEndpoint[UrlParams]{
		Method: "GET",
		Path:   "/books/:id/:page",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params UrlParams) *f.Response {
			id := params.ID.Get()
			page := params.Page.Get()

			return f.Respond.Ok().JSON([]Book{
				{
					Title: id,
				},
				{
					Title: fmt.Sprintf("%v", page),
				},
			})
		},
	}

	server.Add(books)

	go server.Listen()
	defer server.Close()

	resp, err := http.Get("http://localhost:8080/books/B1Y332O/5")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	cp := resp.Header.Get("Content-Type")
	assert.Equal("application/json; charset=utf-8", cp)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("[{\"Title\":\"B1Y332O\"},{\"Title\":\"5\"}]", string(body))

	resp, err = http.Get("http://localhost:8080/books/B1Y332O/A")
	assert.NoError(err)
	assert.Equal(400, resp.StatusCode)
}

func TestGroupedEndpoints(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	apiGroup := &f.Group{
		Path: "/api",
	}

	type LoopbackPayload struct {
		Value string
	}

	loopback := &f.Endpoint[f.NoParams, LoopbackPayload]{
		Method: "POST",
		Path:   "/loopback",
		Handler: func(request *f.Request, params f.NoParams, body *LoopbackPayload) *f.Response {
			return f.Respond.Ok().JSON(body)
		},
	}

	apiGroup.Add(loopback)
	server.Add(apiGroup)

	go server.Listen()
	defer server.Close()

	postBody, _ := json.Marshal(&LoopbackPayload{Value: "return this back"})
	resp, err := http.Post("http://localhost:8080/api/loopback", "application/json", bytes.NewBuffer(postBody))
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("{\"Value\":\"return this back\"}", string(body))
}

func TestNestedGroups(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	g1 := &f.Group{
		Path: "/group1",
	}

	g2 := &f.Group{
		Path: "/group2",
	}

	g3 := &f.Group{
		Path: "/group3",
	}

	type LoopbackPayload struct {
		Value string
	}

	loopback := &f.Endpoint[f.NoParams, LoopbackPayload]{
		Method: "POST",
		Path:   "/loopback",
		Handler: func(request *f.Request, params f.NoParams, body *LoopbackPayload) *f.Response {
			return f.Respond.Ok().JSON(body)
		},
	}

	g1.Add(loopback)
	g2.Add(g1)
	g3.Add(g2)
	server.Add(g3)

	go server.Listen()
	defer server.Close()

	postBody, _ := json.Marshal(&LoopbackPayload{Value: "return this back"})
	resp, err := http.Post("http://localhost:8080/group3/group2/group1/loopback", "application/json", bytes.NewBuffer(postBody))
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("{\"Value\":\"return this back\"}", string(body))
}

func TestEtagCaching(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	books := &f.BasicEndpoint[f.NoParams]{
		Method: "GET",
		Path:   "/books",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params f.NoParams) *f.Response {
			return f.Respond.Ok().JSON([]Book{
				{
					Title: "Murder in Orient Express",
				},
				{
					Title: "It",
				},
				{
					Title: "Harry Potter",
				},
			})
		},
	}

	server.Add(books)

	go server.Listen()
	defer server.Close()

	resp, err := http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	etag := resp.Header.Get("ETag")
	assert.NotZero(etag)

	client := &http.Client{}
	request, err := http.NewRequest("GET", "http://localhost:8080/books", nil)
	assert.NoError(err)
	request.Header.Set("If-None-Match", etag)
	resp, err = client.Do(request)
	assert.NoError(err)

	assert.Equal(304, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal(0, len(body))
}

func TestAutoEncoding(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	books := &f.BasicEndpoint[f.NoParams]{
		Method: "GET",
		Path:   "/books",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params f.NoParams) *f.Response {
			payload := make([]Book, 0, 100)
			for range 100 {
				payload = append(payload, Book{
					Title: "Some Book",
				})
			}

			return f.Respond.Ok().JSON(payload)
		},
	}

	server.Add(books)

	go server.Listen()
	defer server.Close()

	client := &http.Client{}
	request, err := http.NewRequest("GET", "http://localhost:8080/books", nil)
	request.Header.Add("Accept-Encoding", "br")
	assert.NoError(err)
	resp, err := client.Do(request)
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	assert.Equal("br", resp.Header.Get("Content-Encoding"))

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal(40, len(body))

	reader := brotli.NewReader(bytes.NewReader(body))
	respData, err := io.ReadAll(reader)
	assert.NoError(err)

	assert.Equal("[{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"},{\"Title\":\"Some Book\"}]", string(respData))
}

func TestStreamEndpoint(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	stream := &f.BasicEndpoint[f.NoParams]{
		Method: "GET",
		Path:   "/stream",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		StreamingSettings: &f.StreamingSettings{
			ChunkSize: 64,
		},
		Handler: func(request *f.Request, params f.NoParams) *f.Response {
			return f.Respond.Ok().StreamBytes("text/plain", TEST_FILE_DATA)
		},
	}

	server.Add(stream)

	go server.Listen()
	defer server.Close()

	client := &http.Client{}

	// first 32 bytes (below chunk size)
	request, err := http.NewRequest("GET", "http://localhost:8080/stream", nil)
	assert.NoError(err)
	request.Header.Set("Range", "bytes=0-31")
	resp, err := client.Do(request)
	assert.NoError(err)

	assert.Equal(206, resp.StatusCode)
	assert.Equal("bytes 0-31/185", resp.Header.Get("content-range"))
	assert.Equal("32", resp.Header.Get("content-length"))

	body, err := io.ReadAll(resp.Body)
	assert.Equal(
		[]byte{
			76, 111, 114, 101, 109, 32, 105, 112, 115, 117, 109, 32, 100, 111, 108, 111,
			114, 32, 115, 105, 116, 32, 97, 109, 101, 116, 44, 32, 99, 111, 110, 115,
		},
		body,
	)

	// slice of bytes from the middle (above chunk size)
	request, err = http.NewRequest("GET", "http://localhost:8080/stream", nil)
	assert.NoError(err)
	request.Header.Set("Range", "bytes=32-182")
	resp, err = client.Do(request)
	assert.NoError(err)

	assert.Equal(206, resp.StatusCode)
	assert.Equal("bytes 32-182/185", resp.Header.Get("content-range"))
	assert.Equal("151", resp.Header.Get("content-length"))

	body, err = io.ReadAll(resp.Body)
	assert.Equal(
		TEST_FILE_DATA[32:183],
		body,
	)

	// last 15 bytes (below chunk size)
	request, err = http.NewRequest("GET", "http://localhost:8080/stream", nil)
	assert.NoError(err)
	request.Header.Set("Range", "bytes=170-184")
	resp, err = client.Do(request)
	assert.NoError(err)

	assert.Equal(206, resp.StatusCode)
	assert.Equal("bytes 170-184/185", resp.Header.Get("content-range"))
	assert.Equal("15", resp.Header.Get("content-length"))

	body, err = io.ReadAll(resp.Body)
	assert.Equal(
		TEST_FILE_DATA[170:],
		body,
	)

	// last 89 bytes (above chunk size)
	request, err = http.NewRequest("GET", "http://localhost:8080/stream", nil)
	assert.NoError(err)
	request.Header.Set("Range", "bytes=96-")
	resp, err = client.Do(request)
	assert.NoError(err)

	assert.Equal(206, resp.StatusCode)
	assert.Equal("bytes 96-184/185", resp.Header.Get("content-range"))
	assert.Equal("89", resp.Header.Get("content-length"))

	body, err = io.ReadAll(resp.Body)
	assert.Equal(
		TEST_FILE_DATA[96:],
		body,
	)

	// whole thing (no Range header)
	request, err = http.NewRequest("GET", "http://localhost:8080/stream", nil)
	assert.NoError(err)
	resp, err = client.Do(request)
	assert.NoError(err)

	assert.Equal(206, resp.StatusCode)
	assert.Equal("bytes 0-184/185", resp.Header.Get("content-range"))
	assert.Equal("185", resp.Header.Get("content-length"))

	body, err = io.ReadAll(resp.Body)
	assert.Equal(
		TEST_FILE_DATA,
		body,
	)
}

func TestRestEndpointHandling(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	restEndp := &f.RestEndpoints[BooksQueryParams, BookResource]{
		Path:     "/books",
		Encoding: "auto",
		Resource: BookResource{},
	}

	server.Add(restEndp)

	go server.Listen()
	defer server.Close()

	client := &http.Client{}

	resp, err := http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("[]", string(body))

	postBody, _ := json.Marshal(&BookResource{"1", "Harry Potter", 100})
	resp, err = http.Post("http://localhost:8080/books", "application/json", bytes.NewBuffer(postBody))
	assert.NoError(err)
	assert.Equal(201, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("{\"ID\":\"1\",\"Title\":\"Harry Potter\",\"Pages\":100}", string(body))

	resp, err = http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("[{\"ID\":\"1\",\"Title\":\"Harry Potter\",\"Pages\":100}]", string(body))

	postBody, _ = json.Marshal(&BookResource{"2", "It", 543})
	resp, err = http.Post("http://localhost:8080/books", "application/json", bytes.NewBuffer(postBody))
	assert.NoError(err)
	assert.Equal(201, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("{\"ID\":\"2\",\"Title\":\"It\",\"Pages\":543}", string(body))

	resp, err = http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("[{\"ID\":\"1\",\"Title\":\"Harry Potter\",\"Pages\":100},{\"ID\":\"2\",\"Title\":\"It\",\"Pages\":543}]", string(body))

	resp, err = http.Get("http://localhost:8080/books?filter=Harry")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("[{\"ID\":\"1\",\"Title\":\"Harry Potter\",\"Pages\":100}]", string(body))

	resp, err = http.Get("http://localhost:8080/books?filter=foobar")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("[]", string(body))

	resp, err = http.Get("http://localhost:8080/books/1")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("{\"ID\":\"1\",\"Title\":\"Harry Potter\",\"Pages\":100}", string(body))

	resp, err = http.Get("http://localhost:8080/books/2")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("{\"ID\":\"2\",\"Title\":\"It\",\"Pages\":543}", string(body))

	resp, err = http.Get("http://localhost:8080/books/3")
	assert.NoError(err)
	assert.Equal(404, resp.StatusCode)

	postBody, _ = json.Marshal(&BookResource{"1", "Harry Potter", 1001})
	req, err := http.NewRequest("PUT", "http://localhost:8080/books/1", bytes.NewBuffer(postBody))
	assert.NoError(err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("{\"ID\":\"1\",\"Title\":\"Harry Potter\",\"Pages\":1001}", string(body))

	resp, err = http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("[{\"ID\":\"1\",\"Title\":\"Harry Potter\",\"Pages\":1001},{\"ID\":\"2\",\"Title\":\"It\",\"Pages\":543}]", string(body))

	resp, err = http.Get("http://localhost:8080/books/1")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("{\"ID\":\"1\",\"Title\":\"Harry Potter\",\"Pages\":1001}", string(body))

	req, err = http.NewRequest("DELETE", "http://localhost:8080/books/1", bytes.NewBuffer([]byte{}))
	assert.NoError(err)
	resp, err = client.Do(req)
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	resp, err = http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal("[{\"ID\":\"2\",\"Title\":\"It\",\"Pages\":543}]", string(body))

	resp, err = http.Get("http://localhost:8080/books/1")
	assert.NoError(err)
	assert.Equal(404, resp.StatusCode)
}
