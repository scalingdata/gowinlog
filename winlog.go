package winlog

import (
  "fmt"
  "time"
  "unsafe"
)

type WinLogEvent struct {
  Msg string
  ProviderName string
  EventSource string
  EventId uint
  Qualifiers uint
  Version uint
  ProcessId uint
  ThreadId uint
  Level uint
  LevelName string
  Opcode uint
  Task uint
  Keywords []string
  Created time.Time
  RecordId uint
  Channel string
  ComputerName string
  TaskText string
  OpcodeText string
  ChannelText string
  ProviderText string
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