package winlog

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// Stores the common fields from a log event
type WinLogEvent struct {
	// From EvtRender
	ProviderName string 
	EventId      uint64 
	Qualifiers   uint64 
	Level        uint64 
	Task         uint64 
	Opcode       uint64 
	Created      time.Time 
	RecordId     uint64 
	ProcessId    uint64 
	ThreadId     uint64 
	Channel      string 
	ComputerName string 
	Version      uint64 

	// From EvtFormatMessage
	Msg          string 
	LevelText    string 
	TaskText     string 
	OpcodeText   string 
	Keywords     []string 
	ChannelText  string 
	ProviderText string 
	IdText       string
}

type channelWatcher struct {
	bookmark    BookmarkHandle
	subscription ListenerHandle
}

// Watches one or more event log channels
// and publishes events and errors to Go
// channels
type WinLogWatcher struct {
	errChan   chan error
	eventChan chan *WinLogEvent

	renderContext SysRenderContext
	watches       map[string]*channelWatcher
	watchMutex    sync.Mutex
}

func (self *WinLogWatcher) Event() <-chan *WinLogEvent {
	return self.eventChan
}

func (self *WinLogWatcher) Error() <-chan error {
	return self.errChan
}

// Create a new watcher
func NewWinLogWatcher() (*WinLogWatcher, error) {
	cHandle, err := GetSystemRenderContext()
	if err != nil {
		return nil, err
	}
	return &WinLogWatcher{
		errChan:       make(chan error, 1),
		eventChan:     make(chan *WinLogEvent, 1),
		renderContext: cHandle,
		watches:       make(map[string]*channelWatcher),
	}, nil
}

// Subscribe to a Windows Event Log channel, starting with the next event
// that arrives.
func (self *WinLogWatcher) SubscribeFromNow(channel string) error {
	self.watchMutex.Lock()
	defer self.watchMutex.Unlock()
	if _, ok := self.watches[channel]; ok {
		return fmt.Errorf("A watcher for channel %q already exists", channel)
	}
	newBookmark, err := CreateBookmark()
	if err != nil {
		return err
	}
	subscription, err := CreateListenerFromNow(channel, self)
	if err != nil {
		CloseEventHandle(uint64(newBookmark))
		return err
	}
	self.watches[channel] = &channelWatcher{
		bookmark:     newBookmark,
		subscription: subscription,
	}
	return nil
}

// Subscribe to a Windows Event Log channel, starting with the first event in the log
// after the bookmarked event. There may be a gap if events have been purged.
func (self *WinLogWatcher) SubscribeFromBookmark(channel string, xmlString string) error {
	self.watchMutex.Lock()
	defer self.watchMutex.Unlock()
	if _, ok := self.watches[channel]; ok {
		return fmt.Errorf("A watcher for channel %q already exists", channel)
	}
	bookmark, err := CreateBookmarkFromXml(xmlString)
	if err != nil {
		return err
	}
	subscription, err := CreateListenerFromBookmark(channel, self, bookmark)
	if err != nil {
		CloseEventHandle(uint64(bookmark))
		return err
	}
	self.watches[channel] = &channelWatcher{
		bookmark:     bookmark,
		subscription: subscription,
	}
	return nil
}

// Get an XML bookmark that represents the last event published to
// the Events channel for the specified log channel.
func (self *WinLogWatcher) GetBookmark(channel string) (string, error) {
	self.watchMutex.Lock()
	watch, ok := self.watches[channel]
	self.watchMutex.Unlock()
	if !ok {
		return "", fmt.Errorf("No bookmark for %v exists", channel)
	}
	bookmarkXml, err := RenderBookmark(watch.bookmark)
	if err != nil {
		return "", err
	}
	return bookmarkXml, nil
}

func (self *WinLogWatcher) removeSubscriptionLocked(channel string, watch *channelWatcher) (string, error) {
  cancelErr := CancelEventHandle(uint64(watch.subscription))
  closeErr := CloseEventHandle(uint64(watch.subscription))
  bookmarkXml, bookmarkErr := RenderBookmark(watch.bookmark)
  CloseEventHandle(uint64(watch.bookmark))
  delete(self.watches, channel)
  var err error
  if cancelErr != nil {
    err = cancelErr
  } else if closeErr != nil {
    err = closeErr
  } else if bookmarkErr != nil {
    err = bookmarkErr
  }
  return bookmarkXml, err
}

// Remove the subscription to a specific channel. Returns the XML bookmark
// of the last event handled on the channel.
func (self *WinLogWatcher) RemoveSubscription(channel string) (string, error) {
	self.watchMutex.Lock()
  defer self.watchMutex.Unlock()
	watch, ok := self.watches[channel]
	if !ok {
		return "", fmt.Errorf("No watcher for %q", channel)
	}
	return self.removeSubscriptionLocked(channel, watch)
}

// Remove all subscriptions from this watcher. Returns a map of channels to 
// XML bookmarks, and a map of errors per channel. Each channel will be in 
// only one map.
func (self *WinLogWatcher) RemoveAll() (map[string]string, map[string]error) {
  updatedXml := make(map[string]string)
  errors := make(map[string]error)
  self.watchMutex.Lock()
  defer self.watchMutex.Unlock()
  for channel, watch := range self.watches {
    xmlString, err := self.removeSubscriptionLocked(channel, watch)
    if err != nil {
      errors[channel] = err
    } else {
      updatedXml[channel] = xmlString
    }
  }
  return updatedXml, errors
}

// Close all go channels and remaining handles.
// Must be called after RemoveAll.
func (self *WinLogWatcher) Shutdown() {
  CloseEventHandle(uint64(self.renderContext))
  close(self.errChan)
  close(self.eventChan)
}

func (self *WinLogWatcher) PublishError(err error) {
	self.errChan <- err
}

func (self *WinLogWatcher) convertEvent(handle EventHandle) (*WinLogEvent, error) {
    renderedFields, err := RenderEventValues(self.renderContext, handle)
	if err != nil {
		return nil, fmt.Errorf("Failed to render event values: %v", err)
	}

	publisherHandle, err := GetEventPublisherHandle(renderedFields)
	if err != nil{
		return nil, fmt.Errorf("Failed to render event values: %v", err)
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

	CloseEventHandle(uint64(publisherHandle))
	Free(unsafe.Pointer(renderedFields))

	event := WinLogEvent{
		ProviderName: providerName,
		EventId:      eventId,
		Qualifiers:   qualifiers,
		Level:        level,
		Task:         task,
		Opcode:       opcode,
		Created:      created,
		RecordId:     recordId,
		ProcessId:    processId,
		ThreadId:     threadId,
		Channel:      channel,
		ComputerName: computerName,
		Version:      version,

		Msg:          msgText,
		LevelText:    lvlText,
		TaskText:     taskText,
		OpcodeText:   opcodeText,
		ChannelText:  channelText,
		ProviderText: providerText,
		IdText:       idText,
	}
    return &event, nil
}

func (self *WinLogWatcher) PublishEvent(handle EventHandle) {
	event, err := self.convertEvent(handle)
	if err != nil {
		self.PublishError(err)
		return
	}
	self.eventChan <- event

	self.watchMutex.Lock()
	defer self.watchMutex.Unlock()
	watch, ok := self.watches[event.Channel]
	if !ok {
		self.errChan <- fmt.Errorf("No handle for channel bookmark %q", event.Channel)
		return
	}
	UpdateBookmark(watch.bookmark, handle)
}
