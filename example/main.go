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
  watcher.SubscribeFromNow("Application")
  for {
    select {
    case evt := <- watcher.Event():
      fmt.Printf("Event: %v\n", evt)
      bookmark, _ := watcher.GetBookmark("Application")
      fmt.Printf("Bookmark: %v\n", bookmark)
    case err := <- watcher.Error():
      fmt.Printf("Error: %v\n\n", err)
    }
  }
}