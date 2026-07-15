package butler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"syscall"

	echo "github.com/labstack/echo/v5"
)

type StreamingSettings struct {
	// Determines how often the data is flushed. For example if set to 1MB, connection will be flushed after writing to
	// it 1MB of data (or less if there's nothing more to send)
	//
	// Default is 256KB
	ChunkSize uint
	// Default is 5
	KeepAliveTimeout uint
	// Default is 1000
	KeepAliveMax uint
}

var DEFAULT_STREAMING_SETTINGS StreamingSettings = StreamingSettings{
	ChunkSize:        uint(Units.MB) / 4, // 256KB
	KeepAliveTimeout: 5,
	KeepAliveMax:     1000,
}

func (s *StreamingSettings) genKeepAliveHeader() string {
	return fmt.Sprintf("timeout=%d, max=%d", s.KeepAliveTimeout, s.KeepAliveMax)
}

func (resp *Response) stream(ctx *echo.Context, request *Request, body []byte) error {
	if len(body) == 0 {
		panic("cannot stream an empty body")
	}

	reader := NewBytesReader(body)
	return streamReader(ctx, request, resp, reader)
}

func (resp *Response) streamFromReader(ctx *echo.Context, request *Request) error {
	if resp.streamReader == nil {
		panic("cannot stream without a reader")
	}
	return streamReader(ctx, request, resp, resp.streamReader)
}

func streamReader(ctx *echo.Context, request *Request, resp *Response, reader ButlerReader) error {
	defer reader.Close()
	respH := ctx.Response().Header()

	contentType := resp.Headers.Get("Content-Type")

	requestedRange, err := parseRangeHeader(request.Headers)
	if err != nil {
		resp.Status = 400
		ctx.NoContent(400)
		return err
	}

	settings := resp.StreamingSettings
	if settings == nil {
		settings = &DEFAULT_STREAMING_SETTINGS
	}

	respH.Set("Accept-Ranges", "bytes")
	respH.Set("Connection", "keep-alive")
	respH.Set("Keep-Alive", settings.genKeepAliveHeader())
	dataSize := reader.Len()

	if requestedRange == nil {
		requestedRange = &Range{
			Start:    0,
			End:      max(0, dataSize-1),
			HasStart: true,
			HasEnd:   true,
		}
	} else {
		// if the request contained a Range header we must set the code to 206 (Partial Content)
		resp.Status = 206
	}

	if !requestedRange.HasEnd {
		requestedRange.End = max(0, dataSize-1)
	}

	writer := ctx.Response()

	if requestedRange.Start == requestedRange.End {
		respH.Set("Content-Length", "0")
		writer.WriteHeader(resp.Status)
		return nil
	}

	endIdx := min(dataSize-1, requestedRange.End)
	requestedLen := endIdx - requestedRange.Start + 1

	contentLength := strconv.FormatInt(int64(requestedLen), 10)
	contentRange := ("bytes " +
		strconv.FormatInt(int64(requestedRange.Start), 10) +
		"-" + strconv.FormatInt(int64(endIdx), 10) +
		"/" + strconv.FormatInt(int64(dataSize), 10))
	respH.Set("Content-Length", contentLength)
	respH.Set("Content-Range", contentRange)
	respH.Set("Content-Type", contentType)

	done := reader.Skip(requestedRange.Start)
	if done {
		resp.Status = 400
		ctx.NoContent(400)
		return err
	}

	maxChunkSize := int(settings.ChunkSize)

	if maxChunkSize <= 0 {
		panic("incorrect chunk size")
	}

	flusher, ok := writer.(http.Flusher)
	if !ok {
		panic("unable to get the http.Flusher")
	}

	writer.WriteHeader(resp.Status)
	if requestedLen > maxChunkSize {
		sent := 0
		for {
			// check if the request channel is still opened
			// and stop sending if it's not
			channelDone := ctx.Request().Context().Done()
			select {
			case <-channelDone:
				return nil
			default:
				// no-op
			}

			nextChunk := min(maxChunkSize, requestedLen-sent)

			var buff []byte
			done, err := reader.Read(nextChunk, &buff)
			if err != nil {
				request.Logger.Error("encountered an unexpected error when reading from the butlerReader ", err)
				return err
			}

			_, err = writer.Write(buff)
			if err != nil {
				if !errors.Is(err, syscall.EPIPE) && errors.Is(err, syscall.ECONNRESET) {
					request.Logger.Error("encountered an unexpected error when writing to the http.ResponseWriter ", err)
					return err
				}
				return nil
			}

			flusher.Flush()
			sent += len(buff)

			if done || sent >= requestedLen {
				return nil
			}
		}
	} else {
		var buff []byte
		reader.Read(requestedLen, &buff)

		_, err = writer.Write(buff)
		if err != nil {
			if !errors.Is(err, syscall.EPIPE) && errors.Is(err, syscall.ECONNRESET) {
				request.Logger.Error("encountered an unexpected error when writing to the http.ResponseWriter ", err)
				return err
			}
			return nil
		}

		flusher.Flush()
	}

	return nil
}

type HttpWriter interface {
	// writes the data to the http response and flushes
	//
	// if the connection was closed by the client, will not do anything and return true
	Write(buff []byte) (connClosed bool)
	// same as Write but accepts string as argument
	WriteString(str string) (connClosed bool)
}

type flushWriter struct {
	reqContext context.Context
	writer     http.ResponseWriter
	flusher    http.Flusher
	mx         sync.Mutex
}

func (fw *flushWriter) Write(buff []byte) bool {
	fw.mx.Lock()
	defer fw.mx.Unlock()

	channelDone := fw.reqContext.Done()
	select {
	case <-channelDone:
		return true
	default:
		// no-op
	}

	_, err := fw.writer.Write(buff)
	if err != nil {
		panic("encountered an unexpected error when writing to the http.ResponseWriter")
	}

	fw.flusher.Flush()
	return false
}

func (fw *flushWriter) WriteString(str string) bool {
	return fw.Write([]byte(str))
}

func (resp *Response) streamFromWriter(ctx *echo.Context, handler func(HttpWriter) error) error {
	respH := ctx.Response().Header()

	contentType := resp.Headers.Get("Content-Type")

	settings := resp.StreamingSettings
	if settings == nil {
		settings = &DEFAULT_STREAMING_SETTINGS
	}

	respH.Set("Connection", "keep-alive")
	respH.Set("Keep-Alive", settings.genKeepAliveHeader())
	respH.Set("Content-Type", contentType)

	writer := ctx.Response()
	flusher, ok := writer.(http.Flusher)
	if !ok {
		panic("unable to get the http.Flusher")
	}

	writer.WriteHeader(resp.Status)

	httpw := &flushWriter{ctx.Request().Context(), writer, flusher, sync.Mutex{}}
	return handler(httpw)
}
