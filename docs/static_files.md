# Static File Server

Static files can be easily hosted by using the `butler.FsEndpoint` struct.

```go
package main

import butler "github.com/ncpa0cpl/butler"

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	staticFiles := &butler.FsEndpoint{
		Path: "/static",
		Dir: "/local/directory/path",
	}

	app.Add(staticFiles)
	app.Listen()
}
```

Just like all the other Endpoint types, `FsEndpoint` can have defined specific settings for Encoding, CachePolicy and Streaming.

### FsEndpoint.Handler

The handler of a FsEndpoint can change how a Response is created for a file, but is not necessary, butler will automatically create a handler if one is not defined.

```go
package main

import (
	butler "github.com/ncpa0cpl/butler"
)

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	staticFiles := &butler.FsEndpoint{
		Path: "/static",
		Dir: "/local/directory/path",
		Handler: func(request *butler.Request, loader FileLoader) *butler.Response {
			response := butler.Respond.Ok().StreamFileLoader(loader)
			return response
		}
	}

	app.Add(staticFiles)
	app.Listen()
}
```

*Note:*
If the loader is not passed over to a FileLoader() or StreamFileLoader() methods of the response, then the FileLoader should be closed manually.

### FsEndpoint.Head

Head method of the FsEndpoint is used for adding headers to the response. It is also called when a HEAD request is received.

By default butler will create a head handler if one is not provided, the default head handler populates the 'Last-Modified' for GET and HEAD request, and the 'Content-Length' and 'Content-Encoding' for HEAD request.

```go
package main

import (
	butler "github.com/ncpa0cpl/butler"
)

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	staticFiles := &butler.FsEndpoint{
		Path: "/static",
		Dir: "/local/directory/path",
		Head: func(ctx *echo.Context, request *Request, status int) *butler.Headers {
			file, _ := os.Open(filepath)
			stat, _ := file.Stat()
			
			headers := NewHeaders().
				Set("Last-Modified", stat.ModTime().Format(http.TimeFormat))
				
			return headers
		}
	}

	app.Add(staticFiles)
	app.Listen()
}
```


### FsEndpoint.FileLoader

The FileLoader of FsEndpoint struct can specify a custom Loader to use instead of the default one.

```go
package main

import (
	butler "github.com/ncpa0cpl/butler"
)

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	staticFiles := &butler.FsEndpoint{
		Path: "/static",
		Dir: "/local/directory/path",
		FileLoader: func() FileLoader {
			return CustomFileLoader{}
		}
	}

	app.Add(staticFiles)
	app.Listen()
}
```

The file loader interface:

```go
type FileLoader interface {
	Path() string
	Load(filepath string) error
	ReadAll() ([]byte, error)
	Reader() (ButlerReader, error)
	Size() int64
	ContentType() string
	ModTime() string
	IsDir() bool
	AllowStreaming() bool
	AllowEncoding() bool
	Close()
}
```
