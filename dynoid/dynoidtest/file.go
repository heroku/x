package dynoidtest

import (
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/heroku/x/dynoid"
)

// FS implements fs.ReadFileFS and is suitable for testing reading tokens.
type FS struct {
	tokens map[string]*file
}

var _ fs.ReadFileFS = &FS{}

// Create a new FS where the DynoID tokens have been populated.
//
// The tokens map keys are the expected audience and the values are the token contents.
func NewFS(tokens map[string]string) *FS {
	f := &FS{tokens: make(map[string]*file)}

	for audience, token := range tokens {
		path := dynoid.LocalTokenPath(audience)
		f.tokens[path] = newFile(audience, token)
	}

	return f
}

func (f *FS) Open(name string) (fs.File, error) {
	tokenFile, ok := f.tokens[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	return &openFile{tokenFile, 0}, nil
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	token, ok := f.tokens[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	return []byte(token.data), nil
}

type file struct {
	name string
	data string
}

func newFile(audience, data string) *file {
	return &file{
		name: path.Base(dynoid.LocalTokenPath(audience)),
		data: data,
	}
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Size() int64 {
	return int64(len(f.data))
}

func (*file) Mode() fs.FileMode {
	return 0444
}

func (*file) ModTime() time.Time {
	return time.Time{}
}

func (*file) IsDir() bool {
	return false
}

func (*file) Sys() any {
	return nil
}

type openFile struct {
	f      *file
	offset int64
}

func (f *openFile) Stat() (fs.FileInfo, error) {
	return f.f, nil
}

func (f *openFile) Read(dst []byte) (int, error) {
	if f.offset >= int64(len(f.f.data)) {
		return 0, io.EOF
	}

	if f.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.f.name, Err: fs.ErrInvalid}
	}

	n := copy(dst, f.f.data[f.offset:])
	f.offset += int64(n)

	return n, nil
}

func (f *openFile) Close() error {
	return nil
}
