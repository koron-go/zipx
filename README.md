# zip extractor

[![GoDoc](https://godoc.org/github.com/koron-go/zipx?status.svg)](https://godoc.org/github.com/koron-go/zipx)
[![Actions/Go](https://github.com/koron-go/zipx/workflows/Go/badge.svg)](https://github.com/koron-go/zipx/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/koron-go/zipx)](https://goreportcard.com/report/github.com/koron-go/zipx)

Package to make **zip** e**x**traction easy (less codes) and efficient.

```console
$ go get github.com/koron-go/zipx
```

```go
import (
    "github.com/koron-go/zipx"
    "context"
)

func main() {
    err := zipx.New().ExtractFile(context.Background(), "archive.zip", zipx.Dir("outdir"))
    if err != nil {
        panic(err)
    }
}
```

See [GoDoc](https://godoc.org/github.com/koron-go/zipx) for references.

## Features

*   Concurrent extraction
    *   Auto concurrency. Use `runtime.NumCPU()` as default.
    *   Configurable by manual

        ```go
        // no concurrency (=sequential)
        x1 := zipx.New().WithConcurrency(1)

        // full concurrency, no limits. it would be slow.
        x0 := zipx.New().WithConcurrency(0)
        ```

*   Customizable progress monitor

    ```go
    var lastPercent = -1

    func monitor(p zipx.Progress) {
        curr := (p.NumDone * 10 / p.NumTotal) * 10
        if curr > lastPercent {
            lastPercent = curr
            report.Printf("progress %d%%", curr)
        }
    }

    func main() {
        err := zipx.New().
            WithMonitor(zipx.MonitorFunc(monitor)).
            ExtractFile(context.Background(), "archive.zip", zipx.Dir("outdir"))
        // ...(snip)...
    }
    ```

*   Configurable `zipx.Destination`
    *   Where to extract files are configurable by implementing
        `zipx.Destination` interface.
    *   zipx provides two `zipx.Destination` implementaions.
        *   for files: `zipx.Dir`
        *   black hole: `zipx.Discard`
    *   Using `zipx.Dir`, you can support names with other encodings rather
        than "UTF-8" easily by configuring `zip.DefaultReinterpreter`.

        ```go
        // Example to support ShiftJIS (Windows or so)
        import "golang.org/x/text/encoding/japanese"

        func shiftjis(s string) (string, error) {
            d := japanese.ShiftJIS.NewDecoder()
            return d.String(s)
        }

        zipx.DefaultReinterpreter = ReinterpretFunc(shiftjis)

        // ...(snip)...

        err := zipx.New().ExtractFile(context.Background(), "archive.zip",
            zipx.Dir("outdir"))
        // ...(snip)...
        ```
    

*   Cancelable by `context.Context`

    ```go
    // Example to cancel extraction after 5 seconds.
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go func() {
        time.Sleep(5 * time.Second)
        cancel()
    }()

    err := zipx.New().ExtractFile(ctx, "archive.zip", zipx.Dir("outdir"))
    if err != nil {
        // this would be "context canceled", if the extraction took over 5
        // seconds.
        log.Print(err)
    }
    ```

## Why use `zipx` ?

golang では `archive/zip` 使えば容易にZIPの解凍は実装できます。しかしファイル
作ったり並列にして速度だしたりプログレスだしたりといったよくある実装は、自分で
やらなきゃいけなくてちょっと面倒でした。

`zipx` ではZIPの解凍で頻繁に必要になるそれらの手続きをパッケージングしました。
