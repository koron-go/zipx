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
	p int
	m Monitor
}

// New creates a extractor for zip archive.
func New() *ZipX {
	return &ZipX{
		p: runtime.NumCPU(),
		m: NullMonitor,
	}
}

// Parallelism get current parallelism of extraction.
func (x *ZipX) Parallelism() int {
	return x.p
}

// SetParallelism updates parallelism of extraction. 0 means no limitation.
func (x *ZipX) SetParallelism(n int) {
	x.p = n
}

// SetMonitor updates a monitor of progress.
func (x *ZipX) SetMonitor(m Monitor) {
	if m == nil {
		m = NullMonitor
	}
	x.m = m
}

// ExtractFile extracts a file as zip archive.
func (x *ZipX) ExtractFile(ctx context.Context, name string, d Destination) error {
	zr, err := zip.OpenReader(name)
	if err != nil {
		return err
	}
	defer zr.Close()
	return x.Extract(ctx, zr.Reader, d)
}

// Extract extracts a zip archive with zip.Reader.
func (x *ZipX) Extract(ctx context.Context, r zip.Reader, d Destination) error {
	ex := x.exCtx(ctx, d, len(r.File))

	for _, zf := range r.File {
		err := ex.acquire()
		if err != nil {
			return err
		}
		go func(zf *zip.File) {
			defer ex.release()
			err := ex.extractOne(zf)
			if err != nil {
				// FIXME: record an error and terminate extractions.
			}
		}(zf)
	}
	ex.wait()
	// FIXME: check errors
	return nil
}

type exCtx struct {
	x   *ZipX
	ctx context.Context
	d   Destination
	p   Progress
	wg  sync.WaitGroup
	ml  sync.Mutex
	sem *semaphore.Weighted
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
	if x.p > 0 {
		ex.sem = semaphore.NewWeighted(int64(x.p))
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
		err := ex.d.CreateDir(zf.Name)
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
