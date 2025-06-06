package httpbutler

import (
	"fmt"
	"net/http"
	"strconv"

	echo "github.com/labstack/echo/v4"
)

type StreamingSettings struct {
	ChunkSize        uint
	KeepAliveTimeout uint
	KeepAliveMax     uint
}

var DEFAULT_STREAMING_SETTINGS StreamingSettings = StreamingSettings{
	ChunkSize:        1024,
	KeepAliveTimeout: 5,
	KeepAliveMax:     1000,
}

func (s *StreamingSettings) genKeepAliveHeader() string {
	return fmt.Sprintf("timeout=%d, max=%d", s.KeepAliveTimeout, s.KeepAliveMax)
}

func (resp *Response) stream(ctx echo.Context, request *Request) error {
	if len(resp.Body) == 0 {
		panic("cannot stream an empty body")
	}

	reader := NewBytesReader(resp.Body)
	return streamReader(ctx, request, resp, reader)
}

func (resp *Response) streamFromReader(ctx echo.Context, request *Request) error {
	if resp.streamReader == nil {
		panic("cannot stream without a reader")
	}
	return streamReader(ctx, request, resp, resp.streamReader)
}

func streamReader(ctx echo.Context, request *Request, resp *Response, reader ButlerReader) error {
	respH := ctx.Response().Header()

	contentType := resp.Headers.Get("Content-Type")

	requestedRange, err := parseRangeHeader(request.Headers)
	if err != nil {
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
			End:      dataSize - 1,
			HasStart: true,
			HasEnd:   true,
		}
	}

	if !requestedRange.HasEnd {
		requestedRange.End = dataSize - 1
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
		ctx.NoContent(400)
		return err
	}

	writer := ctx.Response().Writer

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
				return err
			}

			writer.Write(buff)
			flusher.Flush()
			sent += len(buff)

			if done || sent >= requestedLen {
				return nil
			}
		}
	} else {
		var buff []byte
		reader.Read(requestedLen, &buff)

		writer.Write(buff)
		flusher.Flush()
	}

	return nil
}
