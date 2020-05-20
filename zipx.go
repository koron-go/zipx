package zipx

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"

	"golang.org/x/sync/semaphore"
)

// ZipX is a zip archive extractor.
type ZipX struct {
	c int
	m Monitor
}

// New creates a extractor for zip archive.
func New() *ZipX {
	return &ZipX{
		c: runtime.NumCPU(),
		m: NullMonitor,
	}
}

// Concurrency get current concurrency of extraction.
func (x *ZipX) Concurrency() int {
	return x.c
}

// SetConcurrency updates concurrency of extraction. 0 means no limitation.
func (x *ZipX) SetConcurrency(n int) {
	x.c = n
}

// WithConcurrency updates practical of extraction. 0 means no limitation.
func (x *ZipX) WithConcurrency(n int) *ZipX {
	x.SetConcurrency(n)
	return x
}

// SetMonitor updates a monitor of progress.
func (x *ZipX) SetMonitor(m Monitor) {
	if m == nil {
		m = NullMonitor
	}
	x.m = m
}

// WithMonitor updates a monitor of progress.
func (x *ZipX) WithMonitor(m Monitor) *ZipX {
	x.SetMonitor(m)
	return x
}

// ExtractFile extracts all files from a zip archive file "name".
func (x *ZipX) ExtractFile(ctx context.Context, name string, d Destination) error {
	zr, err := zip.OpenReader(name)
	if err != nil {
		return err
	}
	defer zr.Close()
	return x.Extract(ctx, zr.Reader, d)
}

// Extract extracts all files from zip.Reader as a zip archive.
func (x *ZipX) Extract(ctx context.Context, r zip.Reader, d Destination) error {
	ex := x.exCtx(ctx, d, len(r.File))
	for _, zf := range r.File {
		err := ex.acquire()
		if err != nil {
			return err
		}
		if len(ex.errs) > 0 {
			ex.release()
			break
		}
		go func(zf *zip.File) {
			defer ex.release()
			err := ex.extractOne(zf)
			if err != nil {
				ex.ml.Lock()
				ex.errs = append(ex.errs, err)
				ex.ml.Unlock()
			}
		}(zf)
	}
	ex.wait()
	if len(ex.errs) > 0 {
		return ex.errs[0]
	}
	return nil
}

type exCtx struct {
	x   *ZipX
	ctx context.Context
	d   Destination
	wg  sync.WaitGroup
	ml  sync.Mutex
	sem *semaphore.Weighted

	p    Progress
	errs []error
}

func (x *ZipX) exCtx(ctx context.Context, d Destination, total int) *exCtx {
	ex := &exCtx{
		x:   x,
		ctx: ctx,
		d:   d,
		p: Progress{
			NumDone:  -1,
			NumTotal: total,
		},
	}
	if x.c > 0 {
		ex.sem = semaphore.NewWeighted(int64(x.c))
	}
	ex.inc()
	return ex
}

func (ex *exCtx) inc() {
	ex.ml.Lock()
	ex.p.NumDone++
	ex.x.m.Monitor(ex.p)
	ex.ml.Unlock()
}

func (ex *exCtx) acquire() error {
	ex.wg.Add(1)
	if ex.sem != nil {
		err := ex.sem.Acquire(ex.ctx, 1)
		if err != nil {
			ex.wg.Done()
			return err
		}
	}
	return nil
}

func (ex *exCtx) extractOne(zf *zip.File) error {
	m := zf.Mode()
	if m.IsDir() {
		err := ex.d.CreateDir(zf.Name, DirInfo{NonUTF8: zf.NonUTF8, Mode: m})
		if err != nil {
			return err
		}
		return nil
	}

	if m.IsRegular() {
		fr, err := zf.Open()
		if err != nil {
			return err
		}
		defer fr.Close()

		fw, err := ex.d.CreateFile(zf.Name, FileInfo{
			NonUTF8:  zf.NonUTF8,
			Size:     zf.UncompressedSize64,
			Modified: zf.Modified,
			Mode:     m,
		})
		if err != nil {
			return err
		}
		if wc, ok := fw.(io.WriteCloser); ok {
			defer wc.Close()
		}

		_, err = io.Copy(fw, fr)
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("unsupported file mode %s for %s", m, zf.Name)
}

func (ex *exCtx) release() {
	ex.inc()
	if ex.sem != nil {
		ex.sem.Release(1)
	}
	ex.wg.Done()
}

func (ex *exCtx) wait() {
	ex.wg.Wait()
}
