package httpbutler_test

import (
	"os"
	"testing"

	f "github.com/ncpa0cpl/http-butler"
	"github.com/stretchr/testify/assert"
)

var TEST_FILE_DATA = []byte{76, 111, 114, 101, 109, 32, 105, 112, 115, 117, 109, 32, 100, 111, 108, 111, 114, 32, 115, 105, 116, 32, 97, 109, 101, 116, 44, 32, 99, 111, 110, 115, 101, 99, 116, 101, 116, 117, 114, 32, 97, 100, 105, 112, 105, 115, 99, 105, 110, 103, 32, 101, 108, 105, 116, 46, 32, 70, 117, 115, 99, 101, 32, 109, 111, 108, 101, 115, 116, 105, 101, 32, 108, 97, 99, 105, 110, 105, 97, 32, 99, 111, 109, 109, 111, 100, 111, 46, 32, 86, 105, 118, 97, 109, 117, 115, 32, 102, 101, 117, 103, 105, 97, 116, 32, 117, 114, 110, 97, 32, 117, 116, 32, 110, 117, 110, 99, 32, 99, 117, 114, 115, 117, 115, 44, 32, 118, 101, 108, 32, 109, 97, 120, 105, 109, 117, 115, 32, 101, 114, 111, 115, 32, 115, 117, 115, 99, 105, 112, 105, 116, 46, 32, 83, 101, 100, 32, 102, 101, 114, 109, 101, 110, 116, 117, 109, 32, 109, 97, 120, 105, 109, 117, 115, 32, 112, 111, 114, 116, 116, 105, 116, 111, 114, 46}

const TEST_FILE_NAME = "testdata.txt"

func TestFileReader(t *testing.T) {
	assert := assert.New(t)

	err := os.WriteFile(TEST_FILE_NAME, TEST_FILE_DATA, 0644)
	assert.NoError(err)
	defer os.Remove(TEST_FILE_NAME)

	file, err := os.Open(TEST_FILE_NAME)
	assert.NoError(err)

	reader, err := f.NewFileReader(file)
	assert.NoError(err)

	var buff []byte

	done, err := reader.Read(16, &buff)
	assert.NoError(err)
	assert.Equal(false, done)
	assert.Equal(16, len(buff))
	assert.Equal([]byte{76, 111, 114, 101, 109, 32, 105, 112, 115, 117, 109, 32, 100, 111, 108, 111}, buff)

	done, err = reader.Read(8, &buff)
	assert.NoError(err)
	assert.Equal(false, done)
	assert.Equal(8, len(buff))
	assert.Equal([]byte{114, 32, 115, 105, 116, 32, 97, 109}, buff)

	done = reader.Skip(4)
	assert.Equal(false, done)

	done, err = reader.Read(8, &buff)
	assert.NoError(err)
	assert.Equal(false, done)
	assert.Equal(8, len(buff))
	assert.Equal([]byte{99, 111, 110, 115, 101, 99, 116, 101}, buff)

	done = reader.Skip(84)
	assert.Equal(false, done)

	done, err = reader.Read(32, &buff)
	assert.NoError(err)
	assert.Equal(false, done)
	assert.Equal(32, len(buff))
	assert.Equal([]byte{114, 115, 117, 115, 44, 32, 118, 101, 108, 32, 109, 97, 120, 105, 109, 117, 115, 32, 101, 114, 111, 115, 32, 115, 117, 115, 99, 105, 112, 105, 116, 46}, buff)

	done = reader.Skip(25)
	assert.Equal(false, done)

	done, err = reader.Read(8, &buff)
	assert.NoError(err)
	assert.Equal(true, done)
	assert.Equal(8, len(buff))
	assert.Equal([]byte{114, 116, 116, 105, 116, 111, 114, 46}, buff)
}

func TestBytesReader(t *testing.T) {
	assert := assert.New(t)

	reader := f.NewBytesReader(TEST_FILE_DATA)

	var buff []byte

	done, err := reader.Read(16, &buff)
	assert.NoError(err)
	assert.Equal(false, done)
	assert.Equal(16, len(buff))
	assert.Equal([]byte{76, 111, 114, 101, 109, 32, 105, 112, 115, 117, 109, 32, 100, 111, 108, 111}, buff)

	done, err = reader.Read(8, &buff)
	assert.NoError(err)
	assert.Equal(false, done)
	assert.Equal(8, len(buff))
	assert.Equal([]byte{114, 32, 115, 105, 116, 32, 97, 109}, buff)

	done = reader.Skip(4)
	assert.Equal(false, done)

	done, err = reader.Read(8, &buff)
	assert.NoError(err)
	assert.Equal(false, done)
	assert.Equal(8, len(buff))
	assert.Equal([]byte{99, 111, 110, 115, 101, 99, 116, 101}, buff)

	done = reader.Skip(84)
	assert.Equal(false, done)

	done, err = reader.Read(32, &buff)
	assert.NoError(err)
	assert.Equal(false, done)
	assert.Equal(32, len(buff))
	assert.Equal([]byte{114, 115, 117, 115, 44, 32, 118, 101, 108, 32, 109, 97, 120, 105, 109, 117, 115, 32, 101, 114, 111, 115, 32, 115, 117, 115, 99, 105, 112, 105, 116, 46}, buff)

	done = reader.Skip(25)
	assert.Equal(false, done)

	done, err = reader.Read(8, &buff)
	assert.NoError(err)
	assert.Equal(true, done)
	assert.Equal(8, len(buff))
	assert.Equal([]byte{114, 116, 116, 105, 116, 111, 114, 46}, buff)
}
