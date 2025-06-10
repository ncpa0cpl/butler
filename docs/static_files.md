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

import butler "github.com/ncpa0cpl/butler"

func main() {
	app := butler.CreateServer()
	app.Port = 8080

	staticFiles := &butler.FsEndpoint{
		Path: "/static",
		Dir: "/local/directory/path",
		Handler: func(request *butler.Request, filepath string, file *os.File, fstat os.FileInfo) *butler.Response {
			modTime := fstat.ModTime()

			data, _ := io.ReadAll(file)
			file.Close()

			response := Respond.Ok().Blob(data)
			response.Headers.Set("Last-Modified", modTime.Format(http.TimeFormat))

			return response
		}
	}

	app.Add(staticFiles)
	app.Listen()
}
```

*Note:*
Handler function should close the file handle. (response helpers for files do file closing automatically)
