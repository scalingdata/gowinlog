// +build windows

package winlog

import "C"
import (
	"fmt"
	"sync"
	"time"
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
		shutdown:       make(chan interface{}),
		errChan:        make(chan error),
		eventChan:      make(chan *WinLogEvent),
		renderContext:  cHandle,
		watches:        make(map[string]*channelWatcher),
		renderMessage:  true,
		renderLevel:    true,
		renderTask:     true,
		renderProvider: true,
		renderOpcode:   true,
		renderChannel:  true,
		renderId:       true,
	}, nil
}

// Whether to use EvtFormatMessage to render the event message
func (self *WinLogWatcher) SetRenderMessage(render bool) {
	self.renderMessage = render
}

// Whether to use EvtFormatMessage to render the event level
func (self *WinLogWatcher) SetRenderLevel(render bool) {
	self.renderLevel = render
}

// Whether to use EvtFormatMessage to render the event task
func (self *WinLogWatcher) SetRenderTask(render bool) {
	self.renderTask = render
}

// Whether to use EvtFormatMessage to render the event provider
func (self *WinLogWatcher) SetRenderProvider(render bool) {
	self.renderProvider = render
}

// Whether to use EvtFormatMessage to render the event opcode
func (self *WinLogWatcher) SetRenderOpcode(render bool) {
	self.renderOpcode = render
}

// Whether to use EvtFormatMessage to render the event channel
func (self *WinLogWatcher) SetRenderChannel(render bool) {
	self.renderChannel = render
}

// Whether to use EvtFormatMessage to render the event ID
func (self *WinLogWatcher) SetRenderId(render bool) {
	self.renderId = render
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

var wrappers = make(map[*C.int]*LogEventCallbackWrapper)
var wrappersMutex = &sync.RWMutex{}

func newEventCallbackWrapper(watcher *WinLogWatcher, channel string) *C.int {
	cKey := C.int(0)
	wrappersMutex.Lock()
	wrappers[&cKey] = &LogEventCallbackWrapper{callback: watcher, subscribedChannel: channel}
	wrappersMutex.Unlock()
	return &cKey
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
	callback := newEventCallbackWrapper(self, channel)
	subscription, err := CreateListener(channel, query, flags, callback)
	if err != nil {
		CloseEventHandle(uint64(newBookmark))
		return err
	}
	wrappersMutex.Lock()
	self.watches[channel] = &channelWatcher{
		wrapperPointer: callback,
		bookmark:       newBookmark,
		subscription:   subscription,
		callback:       wrappers[callback],
	}
	wrappersMutex.Unlock()
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
	callback := newEventCallbackWrapper(self, channel)
	bookmark, err := CreateBookmarkFromXml(xmlString)
	if err != nil {
		return fmt.Errorf("Failed to create new bookmark handle: %v", err)
	}
	subscription, err := CreateListenerFromBookmark(channel, query, callback, bookmark)
	if err != nil {
		CloseEventHandle(uint64(bookmark))
		return fmt.Errorf("Failed to add listener: %v", err)
	}
	wrappersMutex.Lock()
	self.watches[channel] = &channelWatcher{
		wrapperPointer: callback,
		bookmark:       bookmark,
		subscription:   subscription,
		callback:       wrappers[callback],
	}
	wrappersMutex.Unlock()
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
	wrappersMutex.Lock()
	delete(wrappers, watch.wrapperPointer)
	wrappersMutex.Unlock()
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
	// Publish the received error to the errChan, but
	// discard if shutdown is in progress
	select {
	case self.errChan <- err:
	case <-self.shutdown:
	}
}

func (self *WinLogWatcher) convertEvent(handle EventHandle, subscribedChannel string) (*WinLogEvent, error) {
	// Rendered values
	var computerName, providerName, channel string
	var level, task, opcode, recordId, qualifiers, eventId, processId, threadId, version uint64
	var created time.Time

	// Localized fields
	var msgText, lvlText, taskText, providerText, opcodeText, channelText, idText string

	// Publisher fields
	var publisherHandle PublisherHandle
	var publisherHandleErr error

	// Render XML, any error is stored in the returned WinLogEvent
	xml, xmlErr := RenderEventXML(handle)

	// Render the values
	renderedFields, renderedFieldsErr := RenderEventValues(self.renderContext, handle)
	if renderedFieldsErr == nil {
		// If fields don't exist we include the nil value
		computerName, _ = RenderStringField(renderedFields, EvtSystemComputer)
		providerName, _ = RenderStringField(renderedFields, EvtSystemProviderName)
		channel, _ = RenderStringField(renderedFields, EvtSystemChannel)
		level, _ = RenderUIntField(renderedFields, EvtSystemLevel)
		task, _ = RenderUIntField(renderedFields, EvtSystemTask)
		opcode, _ = RenderUIntField(renderedFields, EvtSystemOpcode)
		recordId, _ = RenderUIntField(renderedFields, EvtSystemEventRecordId)
		qualifiers, _ = RenderUIntField(renderedFields, EvtSystemQualifiers)
		eventId, _ = RenderUIntField(renderedFields, EvtSystemEventID)
		processId, _ = RenderUIntField(renderedFields, EvtSystemProcessID)
		threadId, _ = RenderUIntField(renderedFields, EvtSystemThreadID)
		version, _ = RenderUIntField(renderedFields, EvtSystemVersion)
		created, _ = RenderFileTimeField(renderedFields, EvtSystemTimeCreated)

		// Render localized fields
		publisherHandle, publisherHandleErr = GetEventPublisherHandle(renderedFields)
		if publisherHandleErr == nil {
			if self.renderMessage {
				msgText, _ = FormatMessage(publisherHandle, handle, EvtFormatMessageEvent)
			}

			if self.renderLevel {
				lvlText, _ = FormatMessage(publisherHandle, handle, EvtFormatMessageLevel)
			}

			if self.renderTask {
				taskText, _ = FormatMessage(publisherHandle, handle, EvtFormatMessageTask)
			}

			if self.renderProvider {
				providerText, _ = FormatMessage(publisherHandle, handle, EvtFormatMessageProvider)
			}

			if self.renderOpcode {
				opcodeText, _ = FormatMessage(publisherHandle, handle, EvtFormatMessageOpcode)
			}

			if self.renderChannel {
				channelText, _ = FormatMessage(publisherHandle, handle, EvtFormatMessageChannel)
			}

			if self.renderId {
				idText, _ = FormatMessage(publisherHandle, handle, EvtFormatMessageId)
			}
		}

		CloseEventHandle(uint64(publisherHandle))
		Free(unsafe.Pointer(renderedFields))
	}

	// Return an error if we couldn't render anything useful
	if xmlErr != nil && renderedFieldsErr != nil {
		return nil, fmt.Errorf("Failed to render event values and XML: %v", []error{renderedFieldsErr, xmlErr})
	}

	event := WinLogEvent{
		Xml:    xml,
		XmlErr: xmlErr,

		ProviderName:      providerName,
		EventId:           eventId,
		Qualifiers:        qualifiers,
		Level:             level,
		Task:              task,
		Opcode:            opcode,
		Created:           created,
		RecordId:          recordId,
		ProcessId:         processId,
		ThreadId:          threadId,
		Channel:           channel,
		ComputerName:      computerName,
		Version:           version,
		RenderedFieldsErr: renderedFieldsErr,

		Msg:                msgText,
		LevelText:          lvlText,
		TaskText:           taskText,
		OpcodeText:         opcodeText,
		ChannelText:        channelText,
		ProviderText:       providerText,
		IdText:             idText,
		PublisherHandleErr: publisherHandleErr,

		SubscribedChannel: subscribedChannel,
	}
	return &event, nil
}

func (self *WinLogWatcher) PublishEvent(handle EventHandle, subscribedChannel string) {

	// Convert the event from the event log schema
	event, err := self.convertEvent(handle, subscribedChannel)
	if err != nil {
		self.PublishError(err)
		return
	}

	// Get the bookmark for the channel
	self.watchMutex.Lock()
	watch, ok := self.watches[subscribedChannel]
	self.watchMutex.Unlock()
	if !ok {
		self.errChan <- fmt.Errorf("No handle for channel bookmark %q", subscribedChannel)
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
