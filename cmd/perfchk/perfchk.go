package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/koron-go/zipx"
)

var (
	outdir      string
	parallelism int
	writeToFile bool
)

var report = log.New(os.Stdout, "", log.Ltime)

func main() {
	defaultOutdir := filepath.Join("tmp", "outdir", strconv.FormatInt(time.Now().Unix(), 10))
	flag.StringVar(&outdir, "o", defaultOutdir, `name of output dir`)
	flag.IntVar(&parallelism, "p", runtime.NumCPU(), `degree of parallelism`)
	flag.BoolVar(&writeToFile, "f", false, `write to real file`)
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("need input zip file")
	}
	err := run(flag.Arg(0), outdir)
	if err != nil {
		log.Fatal(err)
	}
}

var lastPercent = -1

func monitor(p zipx.Progress) {
	curr := (p.NumDone * 10 / p.NumTotal) * 10
	if curr > lastPercent {
		lastPercent = curr
		report.Printf("progress %d%%", curr)
	}
}

func run(name string, outdir string) error {
	x := zipx.New().
		WithConcurrency(parallelism).
		WithMonitor(zipx.MonitorFunc(monitor))
	var dst zipx.Destination
	if writeToFile {
		report.Printf("output to dir %s", outdir)
		dst = zipx.Dir(outdir)
	} else {
		dst = zipx.Discard
	}
	start := time.Now()
	err := x.ExtractFile(context.Background(), name, dst)
	report.Printf("done in %s", time.Since(start))
	return err
}
