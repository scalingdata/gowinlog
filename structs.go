package winlog

import "C"
import (
	"sync"
	"time"
	"unsafe"
)

// Stores the common fields from a log event
type WinLogEvent struct {
	// From EvtRender
	ProviderName      string
	EventId           uint64
	Qualifiers        uint64
	Level             uint64
	Task              uint64
	Opcode            uint64
	Created           time.Time
	RecordId          uint64
	ProcessId         uint64
	ThreadId          uint64
	Channel           string
	ComputerName      string
	Version           uint64
	RenderedFieldsErr error

	// From EvtFormatMessage
	Msg                string
	LevelText          string
	TaskText           string
	OpcodeText         string
	Keywords           []string
	ChannelText        string
	ProviderText       string
	IdText             string
	PublisherHandleErr error

	// XML body
	Xml    string
	XmlErr error

	// Serialied XML bookmark to
	// restart at this event
	Bookmark string

	// Subscribed channel from which the event was retrieved,
	// which may be different than the event's channel
	SubscribedChannel string
}

type channelWatcher struct {
	wrapperPointer *C.int
	subscription   ListenerHandle
	callback       *LogEventCallbackWrapper
	bookmark       BookmarkHandle
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
	shutdown      chan interface{}

	// Optionally render localized fields. EvtFormatMessage() is slow, so
	// skipping these fields provides a big speedup.
	renderMessage  bool
	renderLevel    bool
	renderTask     bool
	renderProvider bool
	renderOpcode   bool
	renderChannel  bool
	renderId       bool
}

type SysRenderContext uint64
type ListenerHandle uint64
type PublisherHandle uint64
type EventHandle uint64
type RenderedFields unsafe.Pointer
type BookmarkHandle uint64

type LogEventCallback interface {
	PublishError(error)
	PublishEvent(EventHandle, string)
}

type LogEventCallbackWrapper struct {
	callback          LogEventCallback
	subscribedChannel string
}
