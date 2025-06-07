package httpbutler

import (
	"io"
	"os"
)

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
	Close()
}

type BytesReader struct {
	bytes  []byte
	cursor int
}

func NewBytesReader(bytes []byte) *BytesReader {
	return &BytesReader{
		bytes, 0,
	}
}

func (r *BytesReader) Read(upto int, p *[]byte) (done bool, err error) {
	end := min(r.cursor+upto, len(r.bytes))
	*p = r.bytes[r.cursor:end]
	r.cursor = end

	if r.cursor >= r.Len()-1 {
		return true, nil
	}

	return false, nil
}

func (r *BytesReader) Skip(upto int) (done bool) {
	r.cursor = r.cursor + upto
	if r.cursor >= r.Len() {
		r.cursor = r.Len() - 1
		return true
	}
	return false
}

func (r *BytesReader) Len() int {
	return len(r.bytes)
}

func (r *BytesReader) Close() {}

type FileReader struct {
	filesize int64
	file     *os.File
	cursor   int
}

func NewFileReader(file *os.File) (*FileReader, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	filesize := stat.Size()

	return &FileReader{
		filesize, file, 0,
	}, nil
}

func (r *FileReader) Read(upto int, p *[]byte) (done bool, err error) {
	end := min(r.cursor+upto, int(r.filesize))
	buff := make([]byte, upto)
	bytesRead, err := r.file.ReadAt(buff, int64(r.cursor))

	if err != nil && err != io.EOF {
		return false, err
	}

	*p = buff[:bytesRead]
	r.cursor = end

	if r.cursor >= r.Len()-1 {
		r.cursor = r.Len() - 1
		return true, nil
	}

	return false, nil
}

func (r *FileReader) Skip(upto int) (done bool) {
	r.cursor = r.cursor + upto
	if r.cursor >= r.Len() {
		r.cursor = r.Len() - 1
		return true
	}
	return false
}

func (r *FileReader) Len() int {
	return int(r.filesize)
}

func (r *FileReader) Close() {
	r.file.Close()
}
