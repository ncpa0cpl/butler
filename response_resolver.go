package butler

import (
	"net/http"
)

type ResponseBodyResolver interface {
	Init(req *Request, resp *Response)
	ETag() string
	SetupStreaming(resp *Response)
	Resolve(resp *Response) []byte
}

type RespByteSliceResolver struct {
	contentType string
	etag        string
	content     []byte

	monitor          RecordBuilder
	autoGenerateEtag bool
}

func NewRespByteSliceResolver(data []byte, contentType string) ResponseBodyResolver {
	return &RespByteSliceResolver{
		contentType: contentType,
		content:     data,
	}
}

func (r *RespByteSliceResolver) Init(req *Request, resp *Response) {
	r.autoGenerateEtag = !resp.CachePolicy.DisableETagGeneration
	r.etag = resp.etag
	r.monitor = req.monitorRecord
}

func (r *RespByteSliceResolver) ETag() string {
	if r.etag != "" {
		return r.etag
	}

	if r.autoGenerateEtag {
		r.monitor.StepStart(MonitorStep.EtagHandler, "")
		etag, err := generateEtagHash(r.content)
		r.monitor.StepEnd(MonitorStep.EtagHandler, "")
		if err != nil {
			return ""
		}

		r.etag = etag
		return etag
	}

	return ""
}

func (r *RespByteSliceResolver) SetupStreaming(resp *Response) {}

func (r *RespByteSliceResolver) Resolve(resp *Response) []byte {
	if len(r.contentType) > 0 {
		resp.Headers.Set("Content-Type", r.contentType)
	}
	return r.content
}

type RespFileResolver struct {
	fs          FilesystemLayer
	filepath    string
	contentType string
	stream      bool

	monitor          RecordBuilder
	autoGenerateEtag bool
	etag             string
	content          []byte
	error            error
}

func NewRespFileResolver(filepath string, contentType string, stream bool) ResponseBodyResolver {
	return &RespFileResolver{
		filepath:    filepath,
		contentType: contentType,
		stream:      stream,
		content:     nil,
	}
}

func (r *RespFileResolver) load() {
	if r.content != nil || r.error != nil || r.stream {
		return
	}

	content, err := r.fs.Read(r.filepath)
	r.content = content
	r.error = err
}

func (r *RespFileResolver) Init(req *Request, resp *Response) {
	r.fs = req.server.filesystem
	r.autoGenerateEtag = !resp.CachePolicy.DisableETagGeneration
	r.etag = resp.etag
	r.monitor = req.monitorRecord
}

func (r *RespFileResolver) ETag() string {
	if r.error != nil {
		return ""
	}

	if r.etag != "" {
		return r.etag
	}

	fsetag, err := r.fs.ETag(r.filepath)
	if err != nil {
		r.error = err
		return ""
	}

	if fsetag != "" {
		r.etag = fsetag
		return fsetag
	}

	if r.stream {
		return ""
	}

	r.load()
	if r.error != nil {
		return ""
	}

	if r.autoGenerateEtag {
		r.monitor.StepStart(MonitorStep.EtagHandler, "")
		etag, err := generateEtagHash(r.content)
		r.monitor.StepEnd(MonitorStep.EtagHandler, "")
		if err != nil {
			r.error = err
			return ""
		}

		r.etag = etag
		return etag
	}

	return ""
}

func (r *RespFileResolver) SetupStreaming(resp *Response) {
	if r.stream {
		var err error
		resp.streamReader, err = r.fs.Reader(r.filepath)
		if err != nil {
			resp.Status = 500
			resp.logs = append(resp.logs,
				responseLog{LogLevel.Error, "failed to create file reader " + r.filepath, err},
			)
		} else if len(r.contentType) > 0 {
			resp.Headers.Set("Content-Type", r.contentType)
		}
	}
}

func (r *RespFileResolver) Resolve(resp *Response) []byte {
	r.load()

	if r.error != nil {
		resp.Status = 500
		resp.logs = append(resp.logs,
			responseLog{LogLevel.Error, "unable to read the given file", r.error},
		)
		return []byte{}
	} else {
		if len(r.contentType) > 0 {
			resp.Headers.Set("Content-Type", r.contentType)
		} else {
			resp.Headers.Set("Content-Type", http.DetectContentType(r.content))
		}

		return r.content
	}
}

type RespFileHandlerResolver struct {
	fs          FilesystemLayer
	file        File
	contentType string
	stream      bool

	monitor          RecordBuilder
	autoGenerateEtag bool
	etag             string
	content          []byte
	error            error
}

func NewRespFileHandlerResolver(file File, contentType string, stream bool) ResponseBodyResolver {
	return &RespFileHandlerResolver{
		file:        file,
		contentType: contentType,
		stream:      stream,
		content:     nil,
	}
}

func (r *RespFileHandlerResolver) load() {
	if r.content != nil || r.error != nil || r.stream {
		return
	}

	content, err := r.fs.ReadFromHandle(r.file)
	r.content = content
	r.error = err
}

func (r *RespFileHandlerResolver) Init(req *Request, resp *Response) {
	r.fs = req.server.filesystem
	r.autoGenerateEtag = !resp.CachePolicy.DisableETagGeneration
	r.etag = resp.etag
	r.monitor = req.monitorRecord
}

func (r *RespFileHandlerResolver) ETag() string {
	if r.error != nil {
		return ""
	}

	if r.etag != "" {
		return r.etag
	}

	fsetag, err := r.fs.ETagFromHandle(r.file)
	if err != nil {
		r.error = err
		return ""
	}

	if fsetag != "" {
		r.etag = fsetag
		return fsetag
	}

	if r.stream {
		return ""
	}

	r.load()
	if r.error != nil {
		return ""
	}

	if r.autoGenerateEtag {
		r.monitor.StepStart(MonitorStep.EtagHandler, "")
		etag, err := generateEtagHash(r.content)
		r.monitor.StepEnd(MonitorStep.EtagHandler, "")
		if err != nil {
			r.error = err
			return ""
		}

		r.etag = etag
		return etag
	}

	return ""
}

func (r *RespFileHandlerResolver) SetupStreaming(resp *Response) {
	if r.stream {
		var err error
		resp.streamReader, err = r.fs.ReaderFromHandle(r.file)
		if err != nil {
			resp.Status = 500
			resp.logs = append(resp.logs,
				responseLog{LogLevel.Error, "failed to create file reader " + r.file.Name(), err},
			)
		} else if len(r.contentType) > 0 {
			resp.Headers.Set("Content-Type", r.contentType)
		}
	}
}

func (r *RespFileHandlerResolver) Resolve(resp *Response) []byte {
	defer r.file.Close()
	r.load()

	if r.error != nil {
		resp.Status = 500
		resp.logs = append(resp.logs, responseLog{LogLevel.Error, "unable to read the given file", r.error})
		return []byte{}
	} else {
		if len(r.contentType) > 0 {
			resp.Headers.Set("Content-Type", r.contentType)
		} else {
			resp.Headers.Set("Content-Type", http.DetectContentType(r.content))
		}

		return r.content
	}
}

type RespFileLoaderResolver struct {
	file   FileLoader
	stream bool

	monitor          RecordBuilder
	autoGenerateEtag bool
	etag             string
	content          []byte
	error            error
}

func NewRespFileLoaderResolver(file FileLoader, stream bool) ResponseBodyResolver {
	return &RespFileLoaderResolver{
		file:   file,
		stream: stream,
	}
}

func (r *RespFileLoaderResolver) load() {
	if r.content != nil || r.error != nil || r.stream {
		return
	}

	content, err := r.file.ReadAll()
	r.content = content
	r.error = err
}

func (r *RespFileLoaderResolver) Init(req *Request, resp *Response) {
	r.autoGenerateEtag = !resp.CachePolicy.DisableETagGeneration
	r.etag = resp.etag
	r.monitor = req.monitorRecord
}

func (r *RespFileLoaderResolver) ETag() string {
	if r.error != nil {
		return ""
	}

	if r.etag != "" {
		return r.etag
	}

	fsetag, err := r.file.ETag()
	if err != nil {
		r.error = err
		return ""
	}

	if fsetag != "" {
		r.etag = fsetag
		return fsetag
	}

	if r.stream {
		return ""
	}

	r.load()
	if r.error != nil {
		return ""
	}

	if r.autoGenerateEtag {
		r.monitor.StepStart(MonitorStep.EtagHandler, "")
		etag, err := generateEtagHash(r.content)
		r.monitor.StepEnd(MonitorStep.EtagHandler, "")
		if err != nil {
			r.error = err
			return ""
		}

		r.etag = etag
		return etag
	}

	return ""
}

func (r *RespFileLoaderResolver) SetupStreaming(resp *Response) {
	if r.stream {
		var err error
		resp.streamReader, err = r.file.Reader()
		if err != nil {
			resp.Status = 500
			resp.logs = append(resp.logs,
				responseLog{LogLevel.Error, "failed to create file reader " + r.file.Path(), err},
			)
		} else {
			contentType := r.file.ContentType()
			if len(contentType) > 0 {
				resp.Headers.Set("Content-Type", contentType)
			}
		}
	}
}

func (r *RespFileLoaderResolver) Resolve(resp *Response) []byte {
	defer r.file.Close()
	r.load()

	if r.error != nil {
		resp.Status = 500
		resp.logs = append(resp.logs, responseLog{LogLevel.Error, "unable to read the given file", r.error})
		return []byte{}
	} else {
		contentType := r.file.ContentType()
		if len(contentType) > 0 {
			resp.Headers.Set("Content-Type", contentType)
		} else {
			resp.Headers.Set("Content-Type", http.DetectContentType(r.content))
		}

		if r.file.AllowEncoding() == false {
			resp.Encoding = "none"
		}

		return r.content
	}
}

type RespEmptyResolver struct{}

func NewRespEmptyResolver() ResponseBodyResolver {
	return &RespEmptyResolver{}
}

func (r *RespEmptyResolver) Init(req *Request, resp *Response) {}

func (r *RespEmptyResolver) ETag() string {
	return ""
}

func (r *RespEmptyResolver) SetupStreaming(resp *Response) {}

func (r *RespEmptyResolver) Resolve(resp *Response) []byte {
	return []byte{}
}
