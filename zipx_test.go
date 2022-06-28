package zipx

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestZipX_Concurrency(t *testing.T) {
	x := New()
	for _, n := range []int{0, 1, 2, 3, 10, 100, 1000} {
		act := x.WithConcurrency(n).Concurrency()
		if act != n {
			t.Fatalf("concurrency mismatch: want=%d got=%d", n, act)
		}
	}
}

type dummyDest struct {
	Names []string
	Dirs  []DirInfo
	Files []FileInfo
}

func (dd *dummyDest) CreateDir(name string, info DirInfo) error {
	dd.Names = append(dd.Names, name)
	dd.Dirs = append(dd.Dirs, info)
	return nil
}

func (dd *dummyDest) CreateFile(name string, info FileInfo) (io.Writer, error) {
	dd.Names = append(dd.Names, name)
	dd.Files = append(dd.Files, info)
	return nil, nil
}

const modeDir = int(os.ModeDir)

func TestExtractWithMode(t *testing.T) {
	bb := &bytes.Buffer{}
	zw := zip.NewWriter(bb)
	for _, d := range []struct {
		name string
		mode int
	}{
		{"f755", 0755},
		{"f123", 0123},
		{"f000", 0000},
		{"f666", 0666},
		{"f777", 0777},
		{"d777", 0777 | modeDir},
		{"d755", 0755 | modeDir},
	} {
		h := &zip.FileHeader{Name: d.name, Method: zip.Deflate}
		h.SetMode(os.FileMode(d.mode))
		_, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
	}
	err := zw.Close()
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(bb.Bytes()), int64(bb.Len()))
	if err != nil {
		t.Fatal(err)
	}
	act := &dummyDest{}
	err = New().WithConcurrency(1).Extract(context.Background(), zr, act)
	if err != nil {
		t.Fatal(err)
	}
	exp := &dummyDest{
		Names: []string{
			"f755",
			"f123",
			"f000",
			"f666",
			"f777",
			"d777",
			"d755",
		},
		Dirs: []DirInfo{
			{Mode: os.FileMode(0777) | os.ModeDir},
			{Mode: os.FileMode(0755) | os.ModeDir},
		},
		Files: []FileInfo{
			{Mode: os.FileMode(0755)},
			{Mode: os.FileMode(0123)},
			{Mode: os.FileMode(0000)},
			{Mode: os.FileMode(0666)},
			{Mode: os.FileMode(0777)},
		},
	}
	if d := cmp.Diff(exp, act, cmpopts.IgnoreFields(FileInfo{}, "Modified")); d != "" {
		t.Fatalf("unexpected extract: -want +got\n%s", d)
	}
}

func writeTestZipFile(t *testing.T, name string) {
	f, err := os.Create(name)
	if err != nil {
		t.Fatalf("failed to os.Create(%s): %s", name, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	zf, err := zw.Create("foo.txt")
	if err != nil {
		t.Fatalf("failed to Create(foo.txt): %s", err)
	}
	fmt.Fprint(zf, "foo")

	zf, err = zw.Create("bar.txt")
	if err != nil {
		t.Fatalf("failed to Create(bar.txt): %s", err)
	}
	fmt.Fprint(zf, "bar")
}

func TestExtractFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "ExtractFile*")
	if err != nil {
		t.Fatalf("failed to TempDir: %s", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	name := filepath.Join(dir, "test.zip")
	writeTestZipFile(t, name)

	outdir := filepath.Join(dir, "outdir")
	var lastProgress Progress
	err = New().WithConcurrency(1).
		WithMonitor(nil).
		WithMonitor(MonitorFunc(func(p Progress) {
			lastProgress = p
		})).
		ExtractFile(context.Background(), name, Dir(outdir))
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(Progress{NumDone: 2, NumTotal: 2}, lastProgress); d != "" {
		t.Fatalf("last progress mismatch: -want +got\n%s", d)
	}

	// FIXME: check content of outdir
}

func TestExtract_CreateFile_failed(t *testing.T) {
	dir, err := ioutil.TempDir("", "Extract_DestFailure*")
	if err != nil {
		t.Fatalf("failed to TempDir: %s", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	name := filepath.Join(dir, "test.zip")
	writeTestZipFile(t, name)

	errExp := &errDest{}
	err = New().WithConcurrency(1).ExtractFile(context.Background(), name, errExp)
	if err == nil {
		t.Fatal("unexpected success, must be failed")
	}
	if !errors.Is(err, errExp) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestExtract_CreateDir_failed(t *testing.T) {
	bb := &bytes.Buffer{}
	zw := zip.NewWriter(bb)
	h := &zip.FileHeader{Name: "foo", Method: zip.Deflate}
	h.SetMode(os.FileMode(0755) | os.ModeDir)
	_, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	zw.Close()
	zr, err := zip.NewReader(bytes.NewReader(bb.Bytes()), int64(bb.Len()))
	if err != nil {
		t.Fatal(err)
	}

	errExp := &errDest{}
	err = New().WithConcurrency(1).Extract(context.Background(), zr, errExp)
	if err == nil {
		t.Fatal("unexpected success, must be failed")
	}
	if !errors.Is(err, errExp) {
		t.Fatalf("unexpected error: %s", err)
	}
}

type errDest struct {
}

func (e *errDest) Error() string {
	return "error destination"
}

func (e *errDest) CreateDir(string, DirInfo) error {
	return e
}

func (e *errDest) CreateFile(string, FileInfo) (io.Writer, error) {
	return nil, e
}

func TestExtract_unsupported_mode(t *testing.T) {
	bb := &bytes.Buffer{}
	zw := zip.NewWriter(bb)
	h := &zip.FileHeader{Name: "foo", Method: zip.Deflate}
	h.SetMode(os.FileMode(0666) | os.ModeSymlink)
	_, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	zw.Close()
	zr, err := zip.NewReader(bytes.NewReader(bb.Bytes()), int64(bb.Len()))
	if err != nil {
		t.Fatal(err)
	}

	err = New().WithConcurrency(1).Extract(context.Background(), zr, Discard)
	if err == nil {
		t.Fatal("unexpected success, must be failed")
	}
	if s := err.Error(); !strings.HasPrefix(s, "unsupported file mode ") {
		t.Fatalf("unexpected error: %s", err)
	}
}
