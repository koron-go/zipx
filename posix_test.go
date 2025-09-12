package zipx

import (
	"syscall"
	"testing"
)

func testUmask(t *testing.T, mask int) {
	oldmask := syscall.Umask(0)
	t.Cleanup(func() {
		syscall.Umask(oldmask)
	})
}
