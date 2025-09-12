package zipx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDir_CreateDir_Mode(t *testing.T) {
	dir := t.TempDir()
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
	dir := t.TempDir()
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

func TestDir_NoUTF8(t *testing.T) {
	dir := t.TempDir()
	d := Dir(dir)

	err := d.CreateDir("foo", DirInfo{NonUTF8: true, Mode: 0777})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "foo")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.Mode().IsDir() {
		t.Fatalf("%s is not directory", path)
	}

	w, err := d.CreateFile("bar", FileInfo{Mode: 0666, NonUTF8: true})
	if err != nil {
		t.Fatal(err)
	}
	if wc, ok := w.(io.WriteCloser); ok {
		wc.Close()
	}
	path = filepath.Join(dir, "bar")
	fi, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.Mode().IsRegular() {
		t.Fatalf("%s is not regular file", path)
	}
}

func TestDiscard(t *testing.T) {
	err := Discard.CreateDir("foo", DirInfo{})
	if err != nil {
		t.Fatal(err)
	}
	w, err := Discard.CreateFile("bar", FileInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if w != io.Discard {
		t.Fatal("Discard.CreateFile returns not io.Discard")
	}
}

func TestDir_FileWrite(t *testing.T) {
	dir := t.TempDir()
	d := Dir(dir)

	const exp = "The quick brown fox jumps over the lazy dog"

	w, err := d.CreateFile("foo.txt", FileInfo{Mode: 0666})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(w, exp)
	if wc, ok := w.(io.WriteCloser); ok {
		wc.Close()
	}

	b, err := os.ReadFile(filepath.Join(dir, "foo.txt"))
	if err != nil {
		t.Fatal(err)
	}
	act := string(b)
	if d := cmp.Diff(exp, act); d != "" {
		t.Fatal("content mismatch: -want +got", exp, act)
	}
}
