# go-usdt - USDT Probes for Go

This library allows dynamic creation of DTrace USDT probes from Go programs.
The current implementation depends on cgo, and has been tested only on Illumos
(specifically SmartOS) and Mac OS X Sierra.

## Installation

```
go get -u github.com/jen20/go-usdt
```

## Usage Example

```go
package main

import (
	"log"
	"reflect"
	"time"

	"github.com/jen20/go-usdt"
)

func main() {
	provider, err := usdt.NewProvider("examplep", "examplem")
	if err != nil {
		log.Fatalf("NewProvider: %s", err)
	}
	defer provider.Close()

	probe, err := usdt.NewProbe("examplef", "examplen", reflect.String, reflect.Int)
	if err != nil {
		log.Fatalf("NewProbe: %s", err)
	}

	err = provider.AddProbe(probe)
	if err != nil {
		log.Fatalf("AddProbe: %s", err)
	}

	err = provider.Enable()
	if err != nil {
		log.Fatalf("Enable: %s", err)
	}

	for i := 0; ; i++ {
		if i%2 == 0 {
			probe.Fire("string argument", 42)
		} else {
			probe.Fire("different argument", 21)
		}
		time.Sleep(1 * time.Second)
	}
}
```

Compile the binary using `go build`, then run it. Probe output can be accessed
using the following DTrace command:

```
dtrace -Z \
    -n 'examplep$target::: { trace(arg1); trace(copyinstr(arg0)) }' \
    -c ./go-usdt-test
```

The `-Z` option permits starting DTrace despite the probes being matched not
yet existing. `go-usdt-test` is the name of the compiled binary. Use of `sudo`
is required on OS X, and the DTrace SIP capability must be configured to allow
use of DTrace.

## TODO

The library is usable in this form if you are targeting only OS X and SmartOS,
and find cgo acceptable. However, future work should broaden the appeal:

- Work out a story for compilation on systems which do not support DTrace which
  has minimal cost in terms of both invasiveness and overhead.
- Investigate static generation (using `go generate`) of probe definitions and
  functions which do not require interface arguments.
- Complete a native Go implementation which does not require use of cgo.

## Credits

- The Go API is based on an older project: [github.com/ecin/go-dtrace](github.com/ecin/go-dtrace)

- The implementation is based on
  [github.com/chrisa/libusdt][github.com/chrisa/libusdt], by Chris Andrews,
  which is compiled statically into the compiled Go binary. The license for
  libusdt is included in the file `LICENSE.libusdt`.
