# Response

## Response status code

The http status code is defined in the `Response.Status` property. There is also a set of helper functions that will
generate responses with different codes.

```go
// create a response struct
response := &butler.Response{
	Status: 200,
}

// 200 using a helper function
response := &butler.Respond.Ok()

// 400 using a helper function
response := &butler.Respond.BadRequest()

// 500 using a helper function
response := &butler.Respond.InternalError()

// 403 using a helper function
response := &butler.Respond.Forbidden()
```

## Response content

Response can define the data that will be sent to the client as a byte array or as a reader for streaming.

The body byte array can be easily accessed on the Response.Body property.

The reader for streaming can only be set through a response method.

### Regular []byte responses

```go
// raw Response
data := "Hello World"
response := &butler.Response{
	Status: 200,
	Body: []byte(data)
}

// Response method helper for JSON
data := struct {Text string}{"Hello World"}
response := &butler.Respond.Ok().JSON(data)

// Response method helper for image file
response := &butler.Respond.Ok().File("path/to/my/file.jpg", "image/jpg")
```

### Streamed responses

```go
// stream a byte array
var data []byte
response := &butler.Respond.Ok().StreamBytes("video/mp4", data)

// stream a file
response := &butler.Respond.Ok().StreamFile("video/mp4", "path/to/my/file.mp4")

// stream from a ButlerReader
file, _ := os.Open("my/file")
reader := butler.NewFileReader(file)
response := &butler.Respond.Ok().Stream("video/mp4", reader)
```

#### ButlerReader

Butler provide two readers out of the box: BytesReader and FileReader. But any struct implementing the ButlerReader can
be used with the `Stream()` function.

ButlerReader interface:
```go
type ButlerReader interface {
	// Read up to `upto` bytes and puts it into the `p` slice pointer.
	//
	// Returns `true` if everything has been read, `false` if there's still more to read.
	Read(upto int, p *[]byte) (done bool, err error)
	// Moves the reader cursor forward by `upto`.
	//
	// Returns `true` if there is no more bytes to read.
	Skip(upto int) (done bool)
	// Total number of bytes
	Len() int
}
````

## Automatic streaming

If the endpoint is not disallowed and the request contains a `Range` header, the Response.Body of any
successful request will be streamed automatically.

If you want to disable automatic streaming, call `SetAllowStreaming(false)` on the response.

## Response Encoding

Each endpoint and response can define what Content Encoding it will use when sending the responses.

There are 5 values that can be set as encoding:

- `auto` encoding will be chosen automatically based on the response body size, content type and request header
- `none` response body will never be encoded
- `gzip` response body will always be encoded using GZip compression if possible
- `brotli` response body will always be encoded using Brotli compression if possible
- `deflate` response body will always be encoded using Defalte compression if possible
