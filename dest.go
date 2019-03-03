package zipx

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// DirInfo describes meta information of a dir.
type DirInfo struct {
	// NonUTF8 indicates name would be non UTF-8 encoding.
	NonUTF8 bool
}

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
	CreateDir(name string, info DirInfo) error

	// CreateFile creates a new file in destination.
	//
	// This can return io.WriteCloser as 1st return parameter, in that case
	// zipx close it automatically after have finished to use.
	CreateFile(name string, info FileInfo) (io.Writer, error)
}

// Dir creates simple directory Destination.
func Dir(name string) Destination {
	return dir(name)
}

type dir string

func (d dir) CreateDir(name string, info DirInfo) error {
	// re-interpret name as different encoding.
	if info.NonUTF8 {
		n, err := DefaultReinterpreter.Reinterpret(name)
		if err != nil {
			return err
		}
		name = n
	}
	return os.MkdirAll(filepath.Join(string(d), name), 0777)
}

func (d dir) CreateFile(name string, info FileInfo) (io.Writer, error) {
	// re-interpret name as different encoding.
	if info.NonUTF8 {
		n, err := DefaultReinterpreter.Reinterpret(name)
		if err != nil {
			return nil, err
		}
		name = n
	}
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

// Discard is a destination which discard all extracted dirs and files.
var Discard Destination = &discard{}

type discard struct{}

func (*discard) CreateDir(name string, info DirInfo) error {
	// nothing to do.
	return nil
}

func (*discard) CreateFile(name string, info FileInfo) (io.Writer, error) {
	// nothing to do.
	return ioutil.Discard, nil
}

// Reinterpreter provides correct to encoding for name of files and dirs.
type Reinterpreter interface {
	Reinterpret(string) (string, error)
}

// ReinterpretFunc is used to implement Reinterpreter by function.
type ReinterpretFunc func(string) (string, error)

// Reinterpret re-interprets string with another encoding.
func (f ReinterpretFunc) Reinterpret(s string) (string, error) {
	return f(s)
}

func noReinterpret(s string) (string, error) {
	// nothing to do.
	return s, nil
}

// DefaultReinterpreter is used by Dir (Destination)
var DefaultReinterpreter = ReinterpretFunc(noReinterpret)
