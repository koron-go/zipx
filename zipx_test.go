package zipx

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

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
		err = zw.Flush()
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
	err = New().WithConcurrency(1).Extract(context.Background(), *zr, act)
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
