// +build windows

package winlog

import (
	"fmt"
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
		shutdown:      make(chan interface{}),
		errChan:       make(chan error),
		eventChan:     make(chan *WinLogEvent),
		renderContext: cHandle,
		watches:       make(map[string]*channelWatcher),
	}, nil
}

// Subscribe to a Windows Event Log channel, starting with the first event
// in the log. `query` is an XPath expression for filtering events: to recieve
// all events on the channel, use "*" as the query.
func (self *WinLogWatcher) SubscribeFromBeginning(channel, query string) error {
	return self.subscribeWithoutBookmark(channel, query, EvtSubscribeStartAtOldestRecord)
}

// Subscribe to a Windows Event Log channel, starting with the next event
// that arrives. `query` is an XPath expression for filtering events: to recieve
// all events on the channel, use "*" as the query.
func (self *WinLogWatcher) SubscribeFromNow(channel, query string) error {
	return self.subscribeWithoutBookmark(channel, query, EvtSubscribeToFutureEvents)
}

func (self *WinLogWatcher) subscribeWithoutBookmark(channel, query string, flags EVT_SUBSCRIBE_FLAGS) error {
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
	subscription, err := CreateListener(channel, query, flags, callback)
	if err != nil {
		CloseEventHandle(uint64(newBookmark))
		return err
	}
	self.watches[channel] = &channelWatcher{
		bookmark:     newBookmark,
		subscription: subscription,
		callback:     callback,
	}
	return nil
}

// Subscribe to a Windows Event Log channel, starting with the first event in the log
// after the bookmarked event. There may be a gap if events have been purged. `query` 
// is an XPath expression for filtering events: to recieve all events on the channel, 
// use "*" as the query
func (self *WinLogWatcher) SubscribeFromBookmark(channel, query string, xmlString string) error {
	self.watchMutex.Lock()
	defer self.watchMutex.Unlock()
	if _, ok := self.watches[channel]; ok {
		return fmt.Errorf("A watcher for channel %q already exists", channel)
	}
	callback := &LogEventCallbackWrapper{self}
	bookmark, err := CreateBookmarkFromXml(xmlString)
	if err != nil {
		return fmt.Errorf("Failed to create new bookmark handle: %v", err)
	}
	subscription, err := CreateListenerFromBookmark(channel, query, callback, bookmark)
	if err != nil {
		CloseEventHandle(uint64(bookmark))
		return fmt.Errorf("Failed to add listener: %v", err)
	}
	self.watches[channel] = &channelWatcher{
		bookmark:     bookmark,
		subscription: subscription,
		callback:     callback,
	}
	return nil
}

func (self *WinLogWatcher) removeSubscription(channel string, watch *channelWatcher) error {
	cancelErr := CancelEventHandle(uint64(watch.subscription))
	closeErr := CloseEventHandle(uint64(watch.subscription))
	CloseEventHandle(uint64(watch.bookmark))
	self.watchMutex.Lock()
	delete(self.watches, channel)
	self.watchMutex.Unlock()
	if cancelErr != nil {
		return cancelErr
	}
	return closeErr
}

// Remove all subscriptions from this watcher and shut down.
func (self *WinLogWatcher) Shutdown() {
	self.watchMutex.Lock()
	watches := self.watches
	self.watchMutex.Unlock()
	close(self.shutdown)
	for channel, watch := range watches {
		self.removeSubscription(channel, watch)
	}
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
	if err != nil {
		return nil, fmt.Errorf("Failed to render event values: %v", err)
	}

	/* If fields don't exist we include the nil value */
	computerName, _ := renderedFields.String(EvtSystemComputer)
	providerName, _ := renderedFields.String(EvtSystemProviderName)
	channel, _ := renderedFields.String(EvtSystemChannel)
	level, _ := renderedFields.Uint(EvtSystemLevel)
	task, _ := renderedFields.Uint(EvtSystemTask)
	opcode, _ := renderedFields.Uint(EvtSystemOpcode)
	recordId, _ := renderedFields.Uint(EvtSystemEventRecordId)
	qualifiers, _ := renderedFields.Uint(EvtSystemQualifiers)
	eventId, _ := renderedFields.Uint(EvtSystemEventID)
	processId, _ := renderedFields.Uint(EvtSystemProcessID)
	threadId, _ := renderedFields.Uint(EvtSystemThreadID)
	version, _ := renderedFields.Uint(EvtSystemVersion)
	created, _ := renderedFields.FileTime(EvtSystemTimeCreated)

	msgText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageEvent)
	lvlText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageLevel)
	taskText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageTask)
	providerText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageProvider)
	opcodeText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageOpcode)
	channelText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageChannel)
	idText, _ := FormatMessage(publisherHandle, handle, EvtFormatMessageId)

	CloseEventHandle(uint64(publisherHandle))

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

	// Convert the event from the event log schema
	event, err := self.convertEvent(handle)
	if err != nil {
		self.PublishError(err)
		return
	}

	// Get the bookmark for the channel
	self.watchMutex.Lock()
	watch, ok := self.watches[event.Channel]
	self.watchMutex.Unlock()
	if !ok {
		self.errChan <- fmt.Errorf("No handle for channel bookmark %q", event.Channel)
		return
	}

	// Update the bookmark with the current event
	UpdateBookmark(watch.bookmark, handle)

	// Serialize the boomark as XML and include it in the event
	bookmarkXml, err := RenderBookmark(watch.bookmark)
	if err != nil {
		self.PublishError(fmt.Errorf("Error rendering bookmark for event - %v", err))
		return
	}
	event.Bookmark = bookmarkXml

	// Don't block when shutting down if the consumer has gone away
	select {
	case self.eventChan <- event:
	case <-self.shutdown:
		return
	}

}
