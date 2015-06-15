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
	err = watcher.SubscribeFromBeginning("Application", "*")
	if err != nil {
		fmt.Printf("Couldn't subscribe to Application: %v", err)
	}
	for {
		select {
		case evt := <-watcher.Event():
			fmt.Printf("Event: %v\n", evt)
			bookmark := evt.Bookmark
			fmt.Printf("Bookmark: %v\n", bookmark)
		case err := <-watcher.Error():
			fmt.Printf("Error: %v\n\n", err)
		}
	}
}
