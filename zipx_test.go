package zipx_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/koron-go/zipx"
)

func Test_extract_with_permission(t *testing.T) {
	xzip := filepath.Join("testdata", "x.zip")
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := zipx.New().ExtractFile(context.Background(), xzip, zipx.Dir(tmp)); err != nil {
		t.Fatal(err)
	}
	x := filepath.Join(tmp, "testdata", "x")
	info, err := os.Stat(x)
	if err != nil {
		t.Fatal(err)
	}
	if !isExecutable(info.Mode()) {
		t.Fatalf("x should be executable, mode=%s", info.Mode())
	}
}

func isExecutable(mode os.FileMode) bool {
	return mode&0100 != 0
}
