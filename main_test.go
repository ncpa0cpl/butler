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

func TestProxyResponse(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	endp := &f.BasicEndpoint[f.NoParams]{
		Method: "GET",
		Path:   "/proxytoswapi",
		Handler: func(request *f.Request, params f.NoParams) *f.Response {
			resp := f.Respond.Proxy("https://swapi.info/api/films")
			resp.Headers.Set("X-Custom-Header", "butler")
			return resp
		},
	}

	server.Add(endp)

	go server.Listen()
	defer server.Close()

	resp, err := http.Get("http://localhost:8080/proxytoswapi")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	var responseContent []Film
	err = json.Unmarshal(body, &responseContent)
	assert.NoError(err)

	// check if the butler server defined headers are present
	assert.Equal("butler", resp.Header.Get("X-Custom-Header"))

	// check if the swapi server defined headers are present
	assert.Equal("/api/films/all.json", resp.Header.Get("x-matched-path"))

	expectedResponse := []Film{
		{
			Title:        "A New Hope",
			EpisodeID:    4,
			OpeningCrawl: "It is a period of civil war.\r\nRebel spaceships, striking\r\nfrom a hidden base, have won\r\ntheir first victory against\r\nthe evil Galactic Empire.\r\n\r\nDuring the battle, Rebel\r\nspies managed to steal secret\r\nplans to the Empire's\r\nultimate weapon, the DEATH\r\nSTAR, an armored space\r\nstation with enough power\r\nto destroy an entire planet.\r\n\r\nPursued by the Empire's\r\nsinister agents, Princess\r\nLeia races home aboard her\r\nstarship, custodian of the\r\nstolen plans that can save her\r\npeople and restore\r\nfreedom to the galaxy....",
			Director:     "George Lucas",
			Producer:     "Gary Kurtz, Rick McCallum",
			ReleaseDate:  "1977-05-25",
			Characters: []string{
				"https://swapi.info/api/people/1",
				"https://swapi.info/api/people/2",
				"https://swapi.info/api/people/3",
				"https://swapi.info/api/people/4",
				"https://swapi.info/api/people/5",
				"https://swapi.info/api/people/6",
				"https://swapi.info/api/people/7",
				"https://swapi.info/api/people/8",
				"https://swapi.info/api/people/9",
				"https://swapi.info/api/people/10",
				"https://swapi.info/api/people/12",
				"https://swapi.info/api/people/13",
				"https://swapi.info/api/people/14",
				"https://swapi.info/api/people/15",
				"https://swapi.info/api/people/16",
				"https://swapi.info/api/people/18",
				"https://swapi.info/api/people/19",
				"https://swapi.info/api/people/81",
			},
			Planets: []string{
				"https://swapi.info/api/planets/1",
				"https://swapi.info/api/planets/2",
				"https://swapi.info/api/planets/3",
			},
			Starships: []string{
				"https://swapi.info/api/starships/2",
				"https://swapi.info/api/starships/3",
				"https://swapi.info/api/starships/5",
				"https://swapi.info/api/starships/9",
				"https://swapi.info/api/starships/10",
				"https://swapi.info/api/starships/11",
				"https://swapi.info/api/starships/12",
				"https://swapi.info/api/starships/13",
			},
			Vehicles: []string{
				"https://swapi.info/api/vehicles/4",
				"https://swapi.info/api/vehicles/6",
				"https://swapi.info/api/vehicles/7",
				"https://swapi.info/api/vehicles/8",
			},
			Species: []string{
				"https://swapi.info/api/species/1",
				"https://swapi.info/api/species/2",
				"https://swapi.info/api/species/3",
				"https://swapi.info/api/species/4",
				"https://swapi.info/api/species/5",
			},
			Created: time.Date(
				2014,
				time.December,
				10,
				14,
				23,
				31,
				880000000,
				time.UTC,
			),
			Edited: time.Date(
				2014,
				time.December,
				20,
				19,
				49,
				45,
				256000000,
				time.UTC,
			),
			URL: "https://swapi.info/api/films/1",
		},
		{
			Title:        "The Empire Strikes Back",
			EpisodeID:    5,
			OpeningCrawl: "It is a dark time for the\r\nRebellion. Although the Death\r\nStar has been destroyed,\r\nImperial troops have driven the\r\nRebel forces from their hidden\r\nbase and pursued them across\r\nthe galaxy.\r\n\r\nEvading the dreaded Imperial\r\nStarfleet, a group of freedom\r\nfighters led by Luke Skywalker\r\nhas established a new secret\r\nbase on the remote ice world\r\nof Hoth.\r\n\r\nThe evil lord Darth Vader,\r\nobsessed with finding young\r\nSkywalker, has dispatched\r\nthousands of remote probes into\r\nthe far reaches of space....",
			Director:     "Irvin Kershner",
			Producer:     "Gary Kurtz, Rick McCallum",
			ReleaseDate:  "1980-05-17",
			Characters: []string{
				"https://swapi.info/api/people/1",
				"https://swapi.info/api/people/2",
				"https://swapi.info/api/people/3",
				"https://swapi.info/api/people/4",
				"https://swapi.info/api/people/5",
				"https://swapi.info/api/people/10",
				"https://swapi.info/api/people/13",
				"https://swapi.info/api/people/14",
				"https://swapi.info/api/people/18",
				"https://swapi.info/api/people/20",
				"https://swapi.info/api/people/21",
				"https://swapi.info/api/people/22",
				"https://swapi.info/api/people/23",
				"https://swapi.info/api/people/24",
				"https://swapi.info/api/people/25",
				"https://swapi.info/api/people/26",
			},
			Planets: []string{
				"https://swapi.info/api/planets/4",
				"https://swapi.info/api/planets/5",
				"https://swapi.info/api/planets/6",
				"https://swapi.info/api/planets/27",
			},
			Starships: []string{
				"https://swapi.info/api/starships/3",
				"https://swapi.info/api/starships/10",
				"https://swapi.info/api/starships/11",
				"https://swapi.info/api/starships/12",
				"https://swapi.info/api/starships/15",
				"https://swapi.info/api/starships/17",
				"https://swapi.info/api/starships/21",
				"https://swapi.info/api/starships/22",
				"https://swapi.info/api/starships/23",
			},
			Vehicles: []string{
				"https://swapi.info/api/vehicles/8",
				"https://swapi.info/api/vehicles/14",
				"https://swapi.info/api/vehicles/16",
				"https://swapi.info/api/vehicles/18",
				"https://swapi.info/api/vehicles/19",
				"https://swapi.info/api/vehicles/20",
			},
			Species: []string{
				"https://swapi.info/api/species/1",
				"https://swapi.info/api/species/2",
				"https://swapi.info/api/species/3",
				"https://swapi.info/api/species/6",
				"https://swapi.info/api/species/7",
			},
			Created: time.Date(
				2014,
				time.December,
				12,
				11,
				26,
				24,
				656000000,
				time.UTC,
			),
			Edited: time.Date(
				2014,
				time.December,
				15,
				13,
				7,
				53,
				386000000,
				time.UTC,
			),
			URL: "https://swapi.info/api/films/2",
		},
		{
			Title:        "Return of the Jedi",
			EpisodeID:    6,
			OpeningCrawl: "Luke Skywalker has returned to\r\nhis home planet of Tatooine in\r\nan attempt to rescue his\r\nfriend Han Solo from the\r\nclutches of the vile gangster\r\nJabba the Hutt.\r\n\r\nLittle does Luke know that the\r\nGALACTIC EMPIRE has secretly\r\nbegun construction on a new\r\narmored space station even\r\nmore powerful than the first\r\ndreaded Death Star.\r\n\r\nWhen completed, this ultimate\r\nweapon will spell certain doom\r\nfor the small band of rebels\r\nstruggling to restore freedom\r\nto the galaxy...",
			Director:     "Richard Marquand",
			Producer:     "Howard G. Kazanjian, George Lucas, Rick McCallum",
			ReleaseDate:  "1983-05-25",
			Characters: []string{
				"https://swapi.info/api/people/1",
				"https://swapi.info/api/people/2",
				"https://swapi.info/api/people/3",
				"https://swapi.info/api/people/4",
				"https://swapi.info/api/people/5",
				"https://swapi.info/api/people/10",
				"https://swapi.info/api/people/13",
				"https://swapi.info/api/people/14",
				"https://swapi.info/api/people/16",
				"https://swapi.info/api/people/18",
				"https://swapi.info/api/people/20",
				"https://swapi.info/api/people/21",
				"https://swapi.info/api/people/22",
				"https://swapi.info/api/people/25",
				"https://swapi.info/api/people/27",
				"https://swapi.info/api/people/28",
				"https://swapi.info/api/people/29",
				"https://swapi.info/api/people/30",
				"https://swapi.info/api/people/31",
				"https://swapi.info/api/people/45",
			},
			Planets: []string{
				"https://swapi.info/api/planets/1",
				"https://swapi.info/api/planets/5",
				"https://swapi.info/api/planets/7",
				"https://swapi.info/api/planets/8",
				"https://swapi.info/api/planets/9",
			},
			Starships: []string{
				"https://swapi.info/api/starships/2",
				"https://swapi.info/api/starships/3",
				"https://swapi.info/api/starships/10",
				"https://swapi.info/api/starships/11",
				"https://swapi.info/api/starships/12",
				"https://swapi.info/api/starships/15",
				"https://swapi.info/api/starships/17",
				"https://swapi.info/api/starships/22",
				"https://swapi.info/api/starships/23",
				"https://swapi.info/api/starships/27",
				"https://swapi.info/api/starships/28",
				"https://swapi.info/api/starships/29",
			},
			Vehicles: []string{
				"https://swapi.info/api/vehicles/8",
				"https://swapi.info/api/vehicles/16",
				"https://swapi.info/api/vehicles/18",
				"https://swapi.info/api/vehicles/19",
				"https://swapi.info/api/vehicles/24",
				"https://swapi.info/api/vehicles/25",
				"https://swapi.info/api/vehicles/26",
				"https://swapi.info/api/vehicles/30",
			},
			Species: []string{
				"https://swapi.info/api/species/1",
				"https://swapi.info/api/species/2",
				"https://swapi.info/api/species/3",
				"https://swapi.info/api/species/5",
				"https://swapi.info/api/species/6",
				"https://swapi.info/api/species/8",
				"https://swapi.info/api/species/9",
				"https://swapi.info/api/species/10",
				"https://swapi.info/api/species/15",
			},
			Created: time.Date(
				2014,
				time.December,
				18,
				10,
				39,
				33,
				255000000,
				time.UTC,
			),
			Edited: time.Date(
				2014,
				time.December,
				20,
				9,
				48,
				37,
				462000000,
				time.UTC,
			),
			URL: "https://swapi.info/api/films/3",
		},
		{
			Title:        "The Phantom Menace",
			EpisodeID:    1,
			OpeningCrawl: "Turmoil has engulfed the\r\nGalactic Republic. The taxation\r\nof trade routes to outlying star\r\nsystems is in dispute.\r\n\r\nHoping to resolve the matter\r\nwith a blockade of deadly\r\nbattleships, the greedy Trade\r\nFederation has stopped all\r\nshipping to the small planet\r\nof Naboo.\r\n\r\nWhile the Congress of the\r\nRepublic endlessly debates\r\nthis alarming chain of events,\r\nthe Supreme Chancellor has\r\nsecretly dispatched two Jedi\r\nKnights, the guardians of\r\npeace and justice in the\r\ngalaxy, to settle the conflict....",
			Director:     "George Lucas",
			Producer:     "Rick McCallum",
			ReleaseDate:  "1999-05-19",
			Characters: []string{
				"https://swapi.info/api/people/2",
				"https://swapi.info/api/people/3",
				"https://swapi.info/api/people/10",
				"https://swapi.info/api/people/11",
				"https://swapi.info/api/people/16",
				"https://swapi.info/api/people/20",
				"https://swapi.info/api/people/21",
				"https://swapi.info/api/people/32",
				"https://swapi.info/api/people/33",
				"https://swapi.info/api/people/34",
				"https://swapi.info/api/people/35",
				"https://swapi.info/api/people/36",
				"https://swapi.info/api/people/37",
				"https://swapi.info/api/people/38",
				"https://swapi.info/api/people/39",
				"https://swapi.info/api/people/40",
				"https://swapi.info/api/people/41",
				"https://swapi.info/api/people/42",
				"https://swapi.info/api/people/43",
				"https://swapi.info/api/people/44",
				"https://swapi.info/api/people/46",
				"https://swapi.info/api/people/47",
				"https://swapi.info/api/people/48",
				"https://swapi.info/api/people/49",
				"https://swapi.info/api/people/50",
				"https://swapi.info/api/people/51",
				"https://swapi.info/api/people/52",
				"https://swapi.info/api/people/53",
				"https://swapi.info/api/people/54",
				"https://swapi.info/api/people/55",
				"https://swapi.info/api/people/56",
				"https://swapi.info/api/people/57",
				"https://swapi.info/api/people/58",
				"https://swapi.info/api/people/59",
			},
			Planets: []string{
				"https://swapi.info/api/planets/1",
				"https://swapi.info/api/planets/8",
				"https://swapi.info/api/planets/9",
			},
			Starships: []string{
				"https://swapi.info/api/starships/31",
				"https://swapi.info/api/starships/32",
				"https://swapi.info/api/starships/39",
				"https://swapi.info/api/starships/40",
				"https://swapi.info/api/starships/41",
			},
			Vehicles: []string{
				"https://swapi.info/api/vehicles/33",
				"https://swapi.info/api/vehicles/34",
				"https://swapi.info/api/vehicles/35",
				"https://swapi.info/api/vehicles/36",
				"https://swapi.info/api/vehicles/37",
				"https://swapi.info/api/vehicles/38",
				"https://swapi.info/api/vehicles/42",
			},
			Species: []string{
				"https://swapi.info/api/species/1",
				"https://swapi.info/api/species/2",
				"https://swapi.info/api/species/6",
				"https://swapi.info/api/species/11",
				"https://swapi.info/api/species/12",
				"https://swapi.info/api/species/13",
				"https://swapi.info/api/species/14",
				"https://swapi.info/api/species/15",
				"https://swapi.info/api/species/16",
				"https://swapi.info/api/species/17",
				"https://swapi.info/api/species/18",
				"https://swapi.info/api/species/19",
				"https://swapi.info/api/species/20",
				"https://swapi.info/api/species/21",
				"https://swapi.info/api/species/22",
				"https://swapi.info/api/species/23",
				"https://swapi.info/api/species/24",
				"https://swapi.info/api/species/25",
				"https://swapi.info/api/species/26",
				"https://swapi.info/api/species/27",
			},
			Created: time.Date(
				2014,
				time.December,
				19,
				16,
				52,
				55,
				740000000,
				time.UTC,
			),
			Edited: time.Date(
				2014,
				time.December,
				20,
				10,
				54,
				7,
				216000000,
				time.UTC,
			),
			URL: "https://swapi.info/api/films/4",
		},
		{
			Title:        "Attack of the Clones",
			EpisodeID:    2,
			OpeningCrawl: "There is unrest in the Galactic\r\nSenate. Several thousand solar\r\nsystems have declared their\r\nintentions to leave the Republic.\r\n\r\nThis separatist movement,\r\nunder the leadership of the\r\nmysterious Count Dooku, has\r\nmade it difficult for the limited\r\nnumber of Jedi Knights to maintain \r\npeace and order in the galaxy.\r\n\r\nSenator Amidala, the former\r\nQueen of Naboo, is returning\r\nto the Galactic Senate to vote\r\non the critical issue of creating\r\nan ARMY OF THE REPUBLIC\r\nto assist the overwhelmed\r\nJedi....",
			Director:     "George Lucas",
			Producer:     "Rick McCallum",
			ReleaseDate:  "2002-05-16",
			Characters: []string{
				"https://swapi.info/api/people/2",
				"https://swapi.info/api/people/3",
				"https://swapi.info/api/people/6",
				"https://swapi.info/api/people/7",
				"https://swapi.info/api/people/10",
				"https://swapi.info/api/people/11",
				"https://swapi.info/api/people/20",
				"https://swapi.info/api/people/21",
				"https://swapi.info/api/people/22",
				"https://swapi.info/api/people/33",
				"https://swapi.info/api/people/35",
				"https://swapi.info/api/people/36",
				"https://swapi.info/api/people/40",
				"https://swapi.info/api/people/43",
				"https://swapi.info/api/people/46",
				"https://swapi.info/api/people/51",
				"https://swapi.info/api/people/52",
				"https://swapi.info/api/people/53",
				"https://swapi.info/api/people/58",
				"https://swapi.info/api/people/59",
				"https://swapi.info/api/people/60",
				"https://swapi.info/api/people/61",
				"https://swapi.info/api/people/62",
				"https://swapi.info/api/people/63",
				"https://swapi.info/api/people/64",
				"https://swapi.info/api/people/65",
				"https://swapi.info/api/people/66",
				"https://swapi.info/api/people/67",
				"https://swapi.info/api/people/68",
				"https://swapi.info/api/people/69",
				"https://swapi.info/api/people/70",
				"https://swapi.info/api/people/71",
				"https://swapi.info/api/people/72",
				"https://swapi.info/api/people/73",
				"https://swapi.info/api/people/74",
				"https://swapi.info/api/people/75",
				"https://swapi.info/api/people/76",
				"https://swapi.info/api/people/77",
				"https://swapi.info/api/people/78",
				"https://swapi.info/api/people/82",
			},
			Planets: []string{
				"https://swapi.info/api/planets/1",
				"https://swapi.info/api/planets/8",
				"https://swapi.info/api/planets/9",
				"https://swapi.info/api/planets/10",
				"https://swapi.info/api/planets/11",
			},
			Starships: []string{
				"https://swapi.info/api/starships/21",
				"https://swapi.info/api/starships/32",
				"https://swapi.info/api/starships/39",
				"https://swapi.info/api/starships/43",
				"https://swapi.info/api/starships/47",
				"https://swapi.info/api/starships/48",
				"https://swapi.info/api/starships/49",
				"https://swapi.info/api/starships/52",
				"https://swapi.info/api/starships/58",
			},
			Vehicles: []string{
				"https://swapi.info/api/vehicles/4",
				"https://swapi.info/api/vehicles/44",
				"https://swapi.info/api/vehicles/45",
				"https://swapi.info/api/vehicles/46",
				"https://swapi.info/api/vehicles/50",
				"https://swapi.info/api/vehicles/51",
				"https://swapi.info/api/vehicles/53",
				"https://swapi.info/api/vehicles/54",
				"https://swapi.info/api/vehicles/55",
				"https://swapi.info/api/vehicles/56",
				"https://swapi.info/api/vehicles/57",
			},
			Species: []string{
				"https://swapi.info/api/species/1",
				"https://swapi.info/api/species/2",
				"https://swapi.info/api/species/6",
				"https://swapi.info/api/species/12",
				"https://swapi.info/api/species/13",
				"https://swapi.info/api/species/15",
				"https://swapi.info/api/species/28",
				"https://swapi.info/api/species/29",
				"https://swapi.info/api/species/30",
				"https://swapi.info/api/species/31",
				"https://swapi.info/api/species/32",
				"https://swapi.info/api/species/33",
				"https://swapi.info/api/species/34",
				"https://swapi.info/api/species/35",
			},
			Created: time.Date(
				2014,
				time.December,
				20,
				10,
				57,
				57,
				886000000,
				time.UTC,
			),
			Edited: time.Date(
				2014,
				time.December,
				20,
				20,
				18,
				48,
				516000000,
				time.UTC,
			),
			URL: "https://swapi.info/api/films/5",
		},
		{
			Title:        "Revenge of the Sith",
			EpisodeID:    3,
			OpeningCrawl: "War! The Republic is crumbling\r\nunder attacks by the ruthless\r\nSith Lord, Count Dooku.\r\nThere are heroes on both sides.\r\nEvil is everywhere.\r\n\r\nIn a stunning move, the\r\nfiendish droid leader, General\r\nGrievous, has swept into the\r\nRepublic capital and kidnapped\r\nChancellor Palpatine, leader of\r\nthe Galactic Senate.\r\n\r\nAs the Separatist Droid Army\r\nattempts to flee the besieged\r\ncapital with their valuable\r\nhostage, two Jedi Knights lead a\r\ndesperate mission to rescue the\r\ncaptive Chancellor....",
			Director:     "George Lucas",
			Producer:     "Rick McCallum",
			ReleaseDate:  "2005-05-19",
			Characters: []string{
				"https://swapi.info/api/people/1",
				"https://swapi.info/api/people/2",
				"https://swapi.info/api/people/3",
				"https://swapi.info/api/people/4",
				"https://swapi.info/api/people/5",
				"https://swapi.info/api/people/6",
				"https://swapi.info/api/people/7",
				"https://swapi.info/api/people/10",
				"https://swapi.info/api/people/11",
				"https://swapi.info/api/people/12",
				"https://swapi.info/api/people/13",
				"https://swapi.info/api/people/20",
				"https://swapi.info/api/people/21",
				"https://swapi.info/api/people/33",
				"https://swapi.info/api/people/35",
				"https://swapi.info/api/people/46",
				"https://swapi.info/api/people/51",
				"https://swapi.info/api/people/52",
				"https://swapi.info/api/people/53",
				"https://swapi.info/api/people/54",
				"https://swapi.info/api/people/55",
				"https://swapi.info/api/people/56",
				"https://swapi.info/api/people/58",
				"https://swapi.info/api/people/63",
				"https://swapi.info/api/people/64",
				"https://swapi.info/api/people/67",
				"https://swapi.info/api/people/68",
				"https://swapi.info/api/people/75",
				"https://swapi.info/api/people/78",
				"https://swapi.info/api/people/79",
				"https://swapi.info/api/people/80",
				"https://swapi.info/api/people/81",
				"https://swapi.info/api/people/82",
				"https://swapi.info/api/people/83",
			},
			Planets: []string{
				"https://swapi.info/api/planets/1",
				"https://swapi.info/api/planets/2",
				"https://swapi.info/api/planets/5",
				"https://swapi.info/api/planets/8",
				"https://swapi.info/api/planets/9",
				"https://swapi.info/api/planets/12",
				"https://swapi.info/api/planets/13",
				"https://swapi.info/api/planets/14",
				"https://swapi.info/api/planets/15",
				"https://swapi.info/api/planets/16",
				"https://swapi.info/api/planets/17",
				"https://swapi.info/api/planets/18",
				"https://swapi.info/api/planets/19",
			},
			Starships: []string{
				"https://swapi.info/api/starships/2",
				"https://swapi.info/api/starships/32",
				"https://swapi.info/api/starships/48",
				"https://swapi.info/api/starships/59",
				"https://swapi.info/api/starships/61",
				"https://swapi.info/api/starships/63",
				"https://swapi.info/api/starships/64",
				"https://swapi.info/api/starships/65",
				"https://swapi.info/api/starships/66",
				"https://swapi.info/api/starships/68",
				"https://swapi.info/api/starships/74",
				"https://swapi.info/api/starships/75",
			},
			Vehicles: []string{
				"https://swapi.info/api/vehicles/33",
				"https://swapi.info/api/vehicles/50",
				"https://swapi.info/api/vehicles/53",
				"https://swapi.info/api/vehicles/56",
				"https://swapi.info/api/vehicles/60",
				"https://swapi.info/api/vehicles/62",
				"https://swapi.info/api/vehicles/67",
				"https://swapi.info/api/vehicles/69",
				"https://swapi.info/api/vehicles/70",
				"https://swapi.info/api/vehicles/71",
				"https://swapi.info/api/vehicles/72",
				"https://swapi.info/api/vehicles/73",
				"https://swapi.info/api/vehicles/76",
			},
			Species: []string{
				"https://swapi.info/api/species/1",
				"https://swapi.info/api/species/2",
				"https://swapi.info/api/species/3",
				"https://swapi.info/api/species/6",
				"https://swapi.info/api/species/15",
				"https://swapi.info/api/species/19",
				"https://swapi.info/api/species/20",
				"https://swapi.info/api/species/23",
				"https://swapi.info/api/species/24",
				"https://swapi.info/api/species/25",
				"https://swapi.info/api/species/26",
				"https://swapi.info/api/species/27",
				"https://swapi.info/api/species/28",
				"https://swapi.info/api/species/29",
				"https://swapi.info/api/species/30",
				"https://swapi.info/api/species/33",
				"https://swapi.info/api/species/34",
				"https://swapi.info/api/species/35",
				"https://swapi.info/api/species/36",
				"https://swapi.info/api/species/37",
			},
			Created: time.Date(
				2014,
				time.December,
				20,
				18,
				49,
				38,
				403000000,
				time.UTC,
			),
			Edited: time.Date(
				2014,
				time.December,
				20,
				20,
				47,
				52,
				73000000,
				time.UTC,
			),
			URL: "https://swapi.info/api/films/6",
		},
	}

	assert.Equal(expectedResponse, responseContent)
}

type Film struct {
	Title        string    `json:"title"`
	EpisodeID    int       `json:"episode_id"`
	OpeningCrawl string    `json:"opening_crawl"`
	Director     string    `json:"director"`
	Producer     string    `json:"producer"`
	ReleaseDate  string    `json:"release_date"`
	Characters   []string  `json:"characters"`
	Planets      []string  `json:"planets"`
	Starships    []string  `json:"starships"`
	Vehicles     []string  `json:"vehicles"`
	Species      []string  `json:"species"`
	Created      time.Time `json:"created"`
	Edited       time.Time `json:"edited"`
	URL          string    `json:"url"`
}
