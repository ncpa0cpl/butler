package butler

import (
	"io"
	"io/fs"
	"os"
	"time"
)

type FileStat interface {
	IsDir() bool
	ModTime() time.Time
	Mode() fs.FileMode
	Name() string
	Size() int64
}

type File interface {
	Close() error
	Name() string
	Read(b []byte) (n int, err error)
	ReadAt(b []byte, off int64) (n int, err error)
	Seek(offset int64, whence int) (ret int64, err error)
}

type FilesystemLayer interface {
	Stat(filepath string) (FileStat, error)
	StatFromHandle(file File) (FileStat, error)
	Reader(filepath string) (ButlerReader, error)
	Read(filepath string) ([]byte, error)
	Handle(filepath string) (File, error)
	ReadFromHandle(file File) ([]byte, error)
	ReaderFromHandle(file File) (ButlerReader, error)
	ETag(filepath string) (string, error)
	ETagFromHandle(file File) (string, error)
}

type DefaultFs struct{}

func (*DefaultFs) Stat(filepath string) (FileStat, error) {
	return os.Stat(filepath)
}

func (*DefaultFs) StatFromHandle(file File) (FileStat, error) {
	return file.(*os.File).Stat()
}

func (*DefaultFs) Handle(filepath string) (File, error) {
	return os.Open(filepath)
}

func (fs *DefaultFs) Reader(filepath string) (ButlerReader, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	return NewFileReader(fs, f)
}

func (fs *DefaultFs) ReaderFromHandle(file File) (ButlerReader, error) {
	return NewFileReader(fs, file)
}

func (*DefaultFs) Read(filepath string) ([]byte, error) {
	return os.ReadFile(filepath)
}

func (*DefaultFs) ReadFromHandle(file File) ([]byte, error) {
	return io.ReadAll(file)
}

func (*DefaultFs) ETag(filepath string) (string, error) {
	return "", nil
}

func (*DefaultFs) ETagFromHandle(file File) (string, error) {
	return "", nil
}
