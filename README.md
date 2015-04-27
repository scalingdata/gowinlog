# gowinlog
Go library for subscribing to the Windows Event Log.

Usage
=======

In Go, create a watcher and subscribe to some log channels. Events and errors are coerced into Go structs and published on the `Event()` and `Error()` channels. Every channel maintains a bookmark which can be stored and used to resume processing at the last message. 

``` Go
package main

import (
  "fmt"
  "gowinlog"
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
      bookmark, err := watcher.GetBookmark("Application")
      if err != nil {
        fmt.Printf("Error getting bookmark: %v", err)
        continue
      }
      fmt.Printf("Bookmark XML: %v\n", bookmark)
    case err := <- watcher.Error():
      fmt.Printf("Error: %v\n\n", err)
    }
  }
}
```

Low-level API
------

`evtRender.go` contains wrappers around the C events API. `bookmark.go` has wrappers around the bookmark API.
