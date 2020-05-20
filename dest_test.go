package zipx

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDir_CreateDir_Mode(t *testing.T) {
	dir, err := ioutil.TempDir("", "CreateDir_Mode*")
	if err != nil {
		t.Fatalf("failed to TempDir: %s", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	d := Dir(dir)

	for _, tc := range []struct {
		name string
		mode os.FileMode
	}{
		{"foo", os.FileMode(0755)},
		{"bar", os.FileMode(0777)},
	} {
		err := d.CreateDir(tc.name, DirInfo{Mode: tc.mode})
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(dir, tc.name)
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !fi.Mode().IsDir() {
			t.Fatalf("%s is not directory", path)
		}
		if runtime.GOOS != "windows" {
			if m := fi.Mode(); m.Perm() != tc.mode {
				t.Fatalf("dir mode mismatch: want=0%o got=0%o", tc.mode, m.Perm())
			}
		}
	}
}

func TestDir_CreateFile_Mode(t *testing.T) {
	dir, err := ioutil.TempDir("", "CreateFile_Mode*")
	if err != nil {
		t.Fatalf("failed to TempDir: %s", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	d := Dir(dir)

	for _, tc := range []struct {
		name string
		mode os.FileMode
	}{
		{"foo", os.FileMode(0755)},
		{"bar", os.FileMode(0666)},
		{"baz", os.FileMode(0644)},
	} {
		w, err := d.CreateFile(tc.name, FileInfo{Mode: tc.mode})
		if err != nil {
			t.Fatal(err)
		}
		if wc, ok := w.(io.WriteCloser); ok {
			wc.Close()
		}

		path := filepath.Join(dir, tc.name)
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !fi.Mode().IsRegular() {
			t.Fatalf("%s is not regular file", path)
		}
		if runtime.GOOS != "windows" {
			if m := fi.Mode(); m.Perm() != tc.mode.Perm() {
				t.Fatalf("file mode mismatch: want=0%o got=0%o", tc.mode, m)
			}
		}
	}
}
