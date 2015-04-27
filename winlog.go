package winlog

import (
  "fmt"
  "sync"
  "time"
)

type WinLogEvent struct {
  // From EvtRender
  ProviderName string
  EventId uint64
  Qualifiers uint64
  Level uint64
  Task uint64
  Opcode uint64
  Created time.Time
  RecordId uint64
  ProcessId uint64
  ThreadId uint64
  Channel string
  ComputerName string
  Version uint64

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
  bookmarkHandles map[string]uint64
  bookmarkMutex sync.Mutex
}

func (self *WinLogWatcher) Event() chan *WinLogEvent {
  return self.eventChan
}

func (self *WinLogWatcher) Error() chan error {
  return self.errChan
}

func NewWinLogWatcher() (*WinLogWatcher, error) {
  cHandle := GetSystemRenderContext()
  if cHandle == 0 {
    return nil, GetLastError()
  }
  return &WinLogWatcher {
    errChan: make(chan error, 1),
    eventChan: make(chan *WinLogEvent, 1),
    renderContext: cHandle,
    bookmarkHandles: make(map[string]uint64),
  }, nil
}

func (self *WinLogWatcher) SubscribeFromNow(channel string) error { 
  newBookmark, err := CreateBookmark()
  if err != nil {
    return err
  }
  err = SetupListener(channel, self)
  if err != nil {
    return err
  }
  self.bookmarkMutex.Lock()
  defer self.bookmarkMutex.Unlock()
  self.bookmarkHandles[channel] = newBookmark
  return nil
}

func (self *WinLogWatcher) SubscribeFromBookmark(channel string, xmlString string) error { 
  bookmark, err := CreateBookmarkFromXml(xmlString)
  if err != nil {
    return err
  }
  err = SetupListener(channel, self)
  if err != nil {
    return err
  }
  self.bookmarkMutex.Lock()
  defer self.bookmarkMutex.Unlock()
  self.bookmarkHandles[channel] = bookmark
  return nil
}

func (self *WinLogWatcher) GetBookmark(channel string) (string, error) {
  self.bookmarkMutex.Lock()
  bookmarkHandle, ok := self.bookmarkHandles[channel]
  self.bookmarkMutex.Unlock()
  if !ok {
    return "", fmt.Errorf("No handle for %v", channel)
  } 
  bookmarkXml, err := RenderBookmark(bookmarkHandle)
  if err != nil {
    return "", err
  }
  return bookmarkXml, nil
}

func (self *WinLogWatcher) publishError(err error) {
  self.errChan <- err;
}

func (self *WinLogWatcher) publishEvent(handle uint64) {
  renderedFields := RenderEventValues(self.renderContext, handle)
  if renderedFields == nil {
      err := GetLastError()
      self.publishError(fmt.Errorf("Failed to render event values: %v", err))
      return
  }
  
  publisherHandle := GetEventPublisherHandle(renderedFields)
  if publisherHandle == 0 {
      err := GetLastError()
      self.publishError(fmt.Errorf("Failed to render event values: %v", err))
      return
  }

  /* If fields don't exist we include the nil value */
  computerName, _ := RenderStringField(renderedFields, EvtSystemComputer)
  providerName, _ := RenderStringField(renderedFields, EvtSystemProviderName)
  channel, _ := RenderStringField(renderedFields, EvtSystemChannel)
  level, _ := RenderUIntField(renderedFields, EvtSystemLevel)
  task, _ := RenderUIntField(renderedFields, EvtSystemTask)
  opcode, _ := RenderUIntField(renderedFields, EvtSystemOpcode)
  recordId, _ := RenderUIntField(renderedFields, EvtSystemEventRecordId)
  qualifiers, _ := RenderUIntField(renderedFields, EvtSystemQualifiers)
  eventId, _ := RenderUIntField(renderedFields, EvtSystemEventID)
  processId, _ := RenderUIntField(renderedFields, EvtSystemProcessID)
  threadId, _ := RenderUIntField(renderedFields, EvtSystemThreadID)
  version, _ := RenderUIntField(renderedFields, EvtSystemVersion)
  created, _ := RenderFileTimeField(renderedFields, EvtSystemTimeCreated)

  msgText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageEvent)
  lvlText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageLevel)
  taskText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageTask)
  providerText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageProvider)
  opcodeText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageOpcode)
  channelText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageChannel)
  idText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageId)

  CloseEventHandle(publisherHandle)
  Free(renderedFields)

  event := WinLogEvent {
    ProviderName: providerName,
    EventId: eventId,
    Qualifiers: qualifiers,
    Level: level,
    Task: task,
    Opcode: opcode,
    Created: created,
    RecordId: recordId,
    ProcessId: processId,
    ThreadId: threadId,
    Channel: channel,
    ComputerName: computerName, 
    Version: version,
    

    Msg: msgText,
    LevelText: lvlText,
    TaskText: taskText,
    OpcodeText: opcodeText,
    ChannelText: channelText,
    ProviderText: providerText,
    IdText: idText,
  }

  self.eventChan <- &event

  self.bookmarkMutex.Lock()
  bookmarkHandle, ok := self.bookmarkHandles[channel]
  self.bookmarkMutex.Unlock()
  if !ok {
    self.errChan <- fmt.Errorf("No handle for channel bookmark %q", channel)
    return
  } 
  UpdateBookmark(bookmarkHandle, handle)
}