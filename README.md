# gowinlog
Go library for subscribing to the Windows Event Log.

Installation
=======

gowinlog uses cgo, so it needs gcc, and `evt.h` must be available. Installing [MinGW-w64](http://mingw-w64.yaxm.org/doku.php) should satisfy both requirements. Make sure the Go architecture and GCC architecture are the same.

Usage
=======

In Go, create a watcher and subscribe to some log channels. Subscriptions can start at the beginning of the log, at the most recent event, or at a bookmark. Events and errors are coerced into Go structs and published on the `Event()` and `Error()` channels. Every event includes a `Bookmark` field which can be stored and used to resume processing at the same point.

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
  // Recieve any future messages
  watcher.SubscribeFromNow("Application")
  for {
    select {
    case evt := <- watcher.Event():
      // Print the event struct
      fmt.Printf("Event: %v\n", evt)
      // Print the updated bookmark for that channel
      fmt.Printf("Bookmark XML: %v\n", evt.Bookmark)
    case err := <- watcher.Error():
      fmt.Printf("Error: %v\n\n", err)
    }
  }
}
```

Low-level API
------

`event.go` contains wrappers around the C events API. `bookmark.go` has wrappers around the bookmark API.
