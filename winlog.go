// +build windows

package winlog

import (
	"fmt"
	"unsafe"
)

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
		shutdown: make(chan interface{}),
		errChan:       make(chan error),
		eventChan:     make(chan *WinLogEvent),
		renderContext: cHandle,
		watches:       make(map[string]*channelWatcher),
	}, nil
}

// Subscribe to a Windows Event Log channel, starting with the first event
// in the log
func (self *WinLogWatcher) SubscribeFromBeginning(channel string) error {
	return self.subscribeWithoutBookmark(channel, EvtSubscribeStartAtOldestRecord)
}

// Subscribe to a Windows Event Log channel, starting with the next event
// that arrives.
func (self *WinLogWatcher) SubscribeFromNow(channel string) error {
	return self.subscribeWithoutBookmark(channel, EvtSubscribeToFutureEvents)
}

func (self *WinLogWatcher) subscribeWithoutBookmark(channel string, flags EVT_SUBSCRIBE_FLAGS) error {
	self.watchMutex.Lock()
	defer self.watchMutex.Unlock()
	if _, ok := self.watches[channel]; ok {
		return fmt.Errorf("A watcher for channel %q already exists", channel)
	}
	newBookmark, err := CreateBookmark()
	if err != nil {
		return fmt.Errorf("Failed to create new bookmark handle: %v", err)
	}
	callback := &LogEventCallbackWrapper{self}
	subscription, err := CreateListener(channel, flags, callback)
	if err != nil {
		CloseEventHandle(uint64(newBookmark))
		return err
	}
	self.watches[channel] = &channelWatcher{
		bookmark:     newBookmark,
		subscription: subscription,
		callback: callback,
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
		return fmt.Errorf("Failed to create bookmark from XML: %v", err)
	}
	callback := &LogEventCallbackWrapper{self}
	subscription, err := CreateListenerFromBookmark(channel, callback, bookmark)
	if err != nil {
		CloseEventHandle(uint64(bookmark))
		return fmt.Errorf("Failed to add listener: %v", err)
	}
	self.watches[channel] = &channelWatcher{
		bookmark:     bookmark,
		subscription: subscription,
		callback: callback,
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
	self.watchMutex.Lock()
	delete(self.watches, channel)
	self.watchMutex.Unlock()
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

// Remove all subscriptions from this watcher and shut down. Returns a map
// of channels to XML bookmarks, and a map of errors per channel. Each 
// channel will be in only one map.
func (self *WinLogWatcher) Shutdown() (map[string]string, map[string]error) {
	updatedXml := make(map[string]string)
	errors := make(map[string]error)
	self.watchMutex.Lock()
	watches := self.watches
	self.watchMutex.Unlock()
	close(self.shutdown)
	for channel, watch := range watches {
		xmlString, err := self.removeSubscriptionLocked(channel, watch)
		if err != nil {
			errors[channel] = err
		} else {
			updatedXml[channel] = xmlString
		}
	}
	CloseEventHandle(uint64(self.renderContext))
	close(self.errChan)
	close(self.eventChan)
	return updatedXml, errors
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
	if err != nil {
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

  select {
	case self.eventChan <- event:
	case <- self.shutdown:
		return
	}

	self.watchMutex.Lock()
	watch, ok := self.watches[event.Channel]
	if !ok {
		self.watchMutex.Unlock()
		self.errChan <- fmt.Errorf("No handle for channel bookmark %q", event.Channel)
		return
	}
	UpdateBookmark(watch.bookmark, handle)
	self.watchMutex.Unlock()
}
