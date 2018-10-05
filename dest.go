package zipx

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

// FileInfo describes meta information of a file.
type FileInfo struct {
	// NonUTF8 indicates name would be non UTF-8 encoding.
	NonUTF8 bool

	// Size is estimated size to write.
	Size uint64

	// Modified is last updated time of file.
	Modified time.Time
}

// Destination provides destination for extraction.
type Destination interface {
	// CreateDir creates a new directory in destination.
	CreateDir(name string) error

	// CreateFile creates a new file in destination.
	CreateFile(name string, info FileInfo) (io.Writer, error)
}

// Dir creates simple directory Destination.
func Dir(name string) Destination {
	return dir(name)
}

type dir string

func (d dir) CreateDir(name string) error {
	return os.MkdirAll(filepath.Join(string(d), name), 0777)
}

func (d dir) CreateFile(name string, info FileInfo) (io.Writer, error) {
	// FIXME: use info.NonUTF8
	name = filepath.Join(string(d), name)
	err := os.MkdirAll(filepath.Dir(name), 0777)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	return &file{
		File:  f,
		mtime: info.Modified,
	}, nil
}

type file struct {
	*os.File
	mtime time.Time
}

func (f file) Write(b []byte) (int, error) {
	return f.File.Write(b)
}

func (f file) Close() error {
	err := f.File.Close()
	if err != nil {
		return err
	}
	return os.Chtimes(f.File.Name(), f.mtime, f.mtime)
}
