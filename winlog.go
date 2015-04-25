package winlog

import (
  "fmt"
  "time"
  "unsafe"
)

type WinLogEvent struct {
  Msg string
  Provider string
  EventSource string
  EventId int
  Version int
  Level int
  LevelName string
  Opcode int
  Keywords []string
  Created time.Time
  RecordId int
  Channel string
  Computer string
}

type WinLogWatcher struct {
  errChan chan error
  eventChan chan *WinLogEvent

  renderContext unsafe.Pointer
}

func NewWinLogWatcher() (*WinLogWatcher, error) {
  cHandle := getSystemRenderContext()
  if cHandle == nil {
    return nil, fmt.Errorf("Error getting render context %v", cHandle)
  }
  return &WinLogWatcher {
    errChan: make(chan error),
    eventChan: make(chan *WinLogEvent),
    renderContext: cHandle,
  }, nil
}

func (self *WinLogWatcher) Subscribe(channel string) { 
  setupListener(channel, self)
}