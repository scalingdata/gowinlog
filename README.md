# gowinlog
Go library for subscribing to Windows Event Log

Usage
=======

In Go, create a watcher and subscribe to some log channels. Listen on the Event() and Error() channels to receive events:

```
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
  
  watcher.Subscribe("Application")
  for {
    select {
    case evt := <- watcher.Event():
      fmt.Printf("Event: %v\n\n", evt)
    case err := <- watcher.Error():
      fmt.Printf("Error: %v\n\n", err)
    }
  }
}
```

Low-level API
------

`evtRender.go` contains wrappers around the C events API. 
