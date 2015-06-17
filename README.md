# gowinlog
Go library for subscribing to the Windows Event Log.

Godocs
=======
[gowinlog v0](https://gopkg.in/scalingdata/gowinlog.v0))

Installation
=======

gowinlog uses cgo, so it needs gcc. Installing [MinGW-w64](http://mingw-w64.yaxm.org/doku.php) should satisfy both requirements. Make sure the Go architecture and GCC architecture are the same.

Features
========

- Includes wrapper for wevtapi.dll, and a high level API
- Supports bookmarks for resuming consumption
- Filter events using XPath expressions 

Usage
=======

``` Go
package main

import (
  "fmt"
  "github.com/scalingdata/gowinlog"
)

func main() {
  watcher, err := winlog.NewWinLogWatcher()
  if err != nil {
    fmt.Printf("Couldn't create watcher: %v\n", err)
    return
  }
  // Recieve any future messages on the Application channel
  // "*" doesn't filter by any fields of the event
  watcher.SubscribeFromNow("Application", "*")
  for {
    select {
    case evt := <- watcher.Event():
      // Print the event struct
      fmt.Printf("Event: %v\n", evt)
    case err := <- watcher.Error():
      fmt.Printf("Error: %v\n\n", err)
    }
  }
}
```

Low-level API
------

`winevt.go` provides wrappers around the relevant functions in `wevtapi.dll`.
