package winlog

import (
  "fmt"
  "time"
)

type WinLogEvent struct {
  // From EvtRender
  ProviderName string
  EventId uint
  Qualifiers uint
  Level uint
  Task uint
  Opcode uint
  Created time.Time
  RecordId uint
  ProcessId uint
  ThreadId uint
  Channel string
  ComputerName string
  Version uint

  // From EvtFormatMessage
  Msg string
  LevelText string
  TaskText string
  OpcodeText string 
  EventSource string
  Keywords []string
  ChannelText string
  ProviderText string
  IdText string
}

type WinLogWatcher struct {
  errChan chan error
  eventChan chan *WinLogEvent

  renderContext uint64
}

func (self *WinLogWatcher) Event() chan *WinLogEvent {
  return self.eventChan
}

func (self *WinLogWatcher) Error() chan error {
  return self.errChan
}

func NewWinLogWatcher() (*WinLogWatcher, error) {
  cHandle := getSystemRenderContext()
  if cHandle == 0 {
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