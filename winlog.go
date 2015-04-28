package winlog

import (
	"fmt"
	"sync"
	"time"
)

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
	EventSource  string
	Keywords     []string
	ChannelText  string
	ProviderText string
	IdText       string
}

type ChannelWatcher struct {
	bookmark     uint64
	subscription uint64
}

type WinLogWatcher struct {
	errChan   chan error
	eventChan chan *WinLogEvent

	renderContext uint64
	watches       map[string]*ChannelWatcher
	watchMutex    sync.Mutex
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
	return &WinLogWatcher{
		errChan:       make(chan error, 1),
		eventChan:     make(chan *WinLogEvent, 1),
		renderContext: cHandle,
		watches:       make(map[string]*ChannelWatcher),
	}, nil
}

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
		CloseEventHandle(newBookmark)
		return err
	}
	self.watches[channel] = &ChannelWatcher{
		bookmark:     newBookmark,
		subscription: subscription,
	}
	return nil
}

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
		CloseEventHandle(bookmark)
		return err
	}
	self.watches[channel] = &ChannelWatcher{
		bookmark:     bookmark,
		subscription: subscription,
	}
	return nil
}

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

func (self *WinLogWatcher) RemoveSubscription(channel string) (string, error) {
	self.watchMutex.Lock()
	watch, ok := self.watches[channel]
	defer self.watchMutex.Unlock()
	if !ok {
		return "", fmt.Errorf("No watcher for %q", channel)
	}
	cancelErr := CancelEventHandle(watch.subscription)
	closeErr := CloseEventHandle(watch.subscription)
	bookmarkXml, bookmarkErr := RenderBookmark(watch.bookmark)
	CloseEventHandle(watch.bookmark)
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

func (self *WinLogWatcher) publishError(err error) {
	self.errChan <- err
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

	self.eventChan <- &event

	self.watchMutex.Lock()
	defer self.watchMutex.Unlock()
	watch, ok := self.watches[channel]
	if !ok {
		self.errChan <- fmt.Errorf("No handle for channel bookmark %q", channel)
		return
	}
	UpdateBookmark(watch.bookmark, handle)
}
