package butler

import (
	"net/http"
)

type FileLoader interface {
	Path() string
	Load(filepath string) error
	ReadAll() ([]byte, error)
	Reader() (ButlerReader, error)
	Size() int64
	ContentType() string
	ModTime() string
	IsDir() bool
	ETag() (string, error)
	AllowStreaming() bool
	AllowEncoding() bool
	Close() error
}

type DefaultFileLoader struct {
	fs FilesystemLayer

	path string
	file File
	stat FileStat

	was_read bool
	contents []byte

	ctype string
}

func (l DefaultFileLoader) Path() string {
	return l.path
}

func (l DefaultFileLoader) Size() int64 {
	return l.stat.Size()
}

func (l DefaultFileLoader) IsDir() bool {
	return l.stat.IsDir()
}

func (l DefaultFileLoader) AllowStreaming() bool {
	ctype := l.ContentType()
	return ctype != "text/javascript" && ctype != "text/html" &&
		ctype != "text/css" && ctype != "application/json" &&
		l.Size() > Units.MB
}

func (l DefaultFileLoader) AllowEncoding() bool {
	return true
}

func (l DefaultFileLoader) ModTime() string {
	return l.stat.ModTime().Format(http.TimeFormat)
}

func (l DefaultFileLoader) ContentType() string {
	if l.ctype != "" {
		return l.ctype
	}

	l.file.Seek(0, 0)
	ctype := Mime.DetectFile(l.path, l.file)
	l.file.Seek(0, 0)
	l.ctype = ctype
	return ctype
}

func (l *DefaultFileLoader) Load(filepath string) error {
	file, err := l.fs.Handle(filepath)
	if err != nil {
		return err
	}

	stat, err := l.fs.StatFromHandle(file)
	if err != nil {
		return err
	}

	l.path = filepath
	l.file = file
	l.stat = stat

	return nil
}

func (l *DefaultFileLoader) ReadAll() ([]byte, error) {
	if l.was_read {
		return l.contents, nil
	}

	l.file.Seek(0, 0)
	contents, err := l.fs.ReadFromHandle(l.file)
	l.file.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	l.contents = contents
	l.was_read = true

	return contents, nil
}

func (l DefaultFileLoader) Reader() (ButlerReader, error) {
	if l.was_read {
		return NewBytesReader(l.contents), nil
	}

	return l.fs.ReaderFromHandle(l.file)
}

func (l DefaultFileLoader) ETag() (string, error) {
	return l.fs.ETagFromHandle(l.file)
}

func (l *DefaultFileLoader) Close() error {
	return l.file.Close()
}
