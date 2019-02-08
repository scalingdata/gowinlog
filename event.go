// +build windows

package winlog

/*
#cgo LDFLAGS: -l wevtapi
// Set windows version to winVista - minimal required for used event log API.
// (Some of mingw installations uses too old windows headers which prevents us
// from using that API) Looks like for cgo that declaration affetcs only
// current file, so for more modern API just create a new file and define
// necessary minimal version.
#define _WIN32_WINNT 0x0600
#include "event.h"
#include "winevt.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type EVT_SUBSCRIBE_FLAGS int

const (
	_ = iota
	EvtSubscribeToFutureEvents
	EvtSubscribeStartAtOldestRecord
	EvtSubscribeStartAfterBookmark
)

type EVT_VARIANT_TYPE int

const (
	EvtVarTypeNull = iota
	EvtVarTypeString
	EvtVarTypeAnsiString
	EvtVarTypeSByte
	EvtVarTypeByte
	EvtVarTypeInt16
	EvtVarTypeUInt16
	EvtVarTypeInt32
	EvtVarTypeUInt32
	EvtVarTypeInt64
	EvtVarTypeUInt64
	EvtVarTypeSingle
	EvtVarTypeDouble
	EvtVarTypeBoolean
	EvtVarTypeBinary
	EvtVarTypeGuid
	EvtVarTypeSizeT
	EvtVarTypeFileTime
	EvtVarTypeSysTime
	EvtVarTypeSid
	EvtVarTypeHexInt32
	EvtVarTypeHexInt64
	EvtVarTypeEvtHandle
	EvtVarTypeEvtXml
)

/* Fields that can be rendered with GetRendered*Value */
type EVT_SYSTEM_PROPERTY_ID int

const (
	EvtSystemProviderName = iota
	EvtSystemProviderGuid
	EvtSystemEventID
	EvtSystemQualifiers
	EvtSystemLevel
	EvtSystemTask
	EvtSystemOpcode
	EvtSystemKeywords
	EvtSystemTimeCreated
	EvtSystemEventRecordId
	EvtSystemActivityID
	EvtSystemRelatedActivityID
	EvtSystemProcessID
	EvtSystemThreadID
	EvtSystemChannel
	EvtSystemComputer
	EvtSystemUserID
	EvtSystemVersion
)

/* Formatting modes for GetFormattedMessage */
type EVT_FORMAT_MESSAGE_FLAGS int

const (
	_ = iota
	EvtFormatMessageEvent
	EvtFormatMessageLevel
	EvtFormatMessageTask
	EvtFormatMessageOpcode
	EvtFormatMessageKeyword
	EvtFormatMessageChannel
	EvtFormatMessageProvider
	EvtFormatMessageId
	EvtFormatMessageXml
)

// Structure for channel setting.
type ChannelConfig struct {
	handle C.EVT_HANDLE
}

// `NewChannelConfig` creates `ChannelConfig`
func NewChannelConfig(channelName string) (*ChannelConfig, error) {
	wChannelName, err := syscall.UTF16FromString(channelName)
	if err != nil {
		return nil, fmt.Errorf("failed to conver string to utf16; %v", err)
	}
	hChannel := C.EvtOpenChannelConfig(nil, (*C.wchar_t)(&wChannelName[0]), 0)
	if hChannel == nil {
		err := windows.GetLastError()
		return nil, fmt.Errorf("falied to open channel config; %v", err)
	}
	return &ChannelConfig{
		handle: hChannel,
	}, nil
}

// `Save` saves config.
func (c *ChannelConfig) Save() error {
	if res := C.EvtSaveChannelConfig(c.handle, 0); res == 0 {
		err := windows.GetLastError()
		return fmt.Errorf("failed to save config; %v", err)
	}
	return nil
}

// `EnableChannel` enables event logging for channel.
func (c *ChannelConfig) EnableChannel(status bool) error {
	if res := C.EnableChannel(c.handle, C.int(btoi(status))); res != 0 {
		err := windows.GetLastError()
		return fmt.Errorf("failed to change channel enable status; %v", err)
	}
	return nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// `SetBufferSizeB` sets log buffer size.
func (c *ChannelConfig) SetBufferSize(bufferSizeB int) error {
	if res := C.SetBufferSizeB(c.handle, C.int(bufferSizeB)); res != 0 {
		err := windows.GetLastError()
		return fmt.Errorf("failed to change channel buffer size; %v", err)
	}
	return nil
}

// Get a handle to a render context which will render properties from the System element.
// Wraps EvtCreateRenderContext() with Flags = EvtRenderContextSystem. The resulting
// handle must be closed with CloseEventHandle.
func GetSystemRenderContext() (SysRenderContext, error) {
	context := SysRenderContext(C.CreateSystemRenderContext())
	if context == 0 {
		return 0, GetLastError()
	}
	return context, nil
}

// Get a handle for a event log subscription on the given channel.
// `query` is an XPath expression to filter the events on the channel - "*" allows all events.
// The resulting handle must be closed with CloseEventHandle.
func CreateListener(channel, query string, startpos EVT_SUBSCRIBE_FLAGS, callbackWrapperKey uint32) (ListenerHandle, error) {
	cChan := C.CString(channel)
	cQuery := C.CString(query)
	listenerHandle := C.CreateListener(cChan, cQuery, C.int(startpos), C.PVOID(uintptr(callbackWrapperKey)))
	C.free(unsafe.Pointer(cChan))
	C.free(unsafe.Pointer(cQuery))
	if listenerHandle == 0 {
		return 0, GetLastError()
	}
	return ListenerHandle(listenerHandle), nil
}

// Get a handle for an event log subscription on the given channel. Will begin at the
// bookmarked event, or the closest possible event if the log has been truncated.
// `query` is an XPath expression to filter the events on the channel - "*" allows all events.
// The resulting handle must be closed with CloseEventHandle.
func CreateListenerFromBookmark(channel, query string, callbackWrapperKey uint32, bookmarkHandle BookmarkHandle) (ListenerHandle, error) {
	cChan := C.CString(channel)
	cQuery := C.CString(query)
	listenerHandle := C.CreateListenerFromBookmark(cChan, cQuery, C.PVOID(uintptr(callbackWrapperKey)), C.ULONGLONG(bookmarkHandle))
	C.free(unsafe.Pointer(cChan))
	C.free(unsafe.Pointer(cQuery))
	if listenerHandle == 0 {
		return 0, GetLastError()
	}
	return ListenerHandle(listenerHandle), nil
}

// Get the Go string for the field at the given index. Returns
// false if the type of the field isn't EvtVarTypeString.
func RenderStringField(fields RenderedFields, fieldIndex EVT_SYSTEM_PROPERTY_ID) (string, bool) {
	fieldType := C.GetRenderedValueType(C.PVOID(fields), C.int(fieldIndex))
	if fieldType != EvtVarTypeString {
		return "", false
	}

	cString := C.GetRenderedStringValue(C.PVOID(fields), C.int(fieldIndex))
	if cString == nil {
		return "", false
	}

	value := C.GoString(cString)
	C.free(unsafe.Pointer(cString))
	return value, true
}

// Get the timestamp of the field at the given index. Returns false if the
// type of the field isn't EvtVarTypeFileTime.
func RenderFileTimeField(fields RenderedFields, fieldIndex EVT_SYSTEM_PROPERTY_ID) (time.Time, bool) {
	fieldType := C.GetRenderedValueType(C.PVOID(fields), C.int(fieldIndex))
	if fieldType != EvtVarTypeFileTime {
		return time.Time{}, false
	}
	stamp := uint64(C.GetRenderedFileTimeValue(C.PVOID(fields), C.int(fieldIndex)))
	stamp -= 116444736000000000
	stamp *= 100
	return time.Unix(0, int64(stamp)).UTC(), true
}

// Get the unsigned integer at the given index. Returns false if the field
// type isn't EvtVarTypeByte, EvtVarTypeUInt16, EvtVarTypeUInt32, or EvtVarTypeUInt64.
func RenderUIntField(fields RenderedFields, fieldIndex EVT_SYSTEM_PROPERTY_ID) (uint64, bool) {
	var field C.ULONGLONG
	fieldType := C.GetRenderedValueType(C.PVOID(fields), C.int(fieldIndex))
	switch fieldType {
	case EvtVarTypeByte:
		field = C.GetRenderedByteValue(C.PVOID(fields), C.int(fieldIndex))
	case EvtVarTypeUInt16:
		field = C.GetRenderedUInt16Value(C.PVOID(fields), C.int(fieldIndex))
	case EvtVarTypeUInt32:
		field = C.GetRenderedUInt32Value(C.PVOID(fields), C.int(fieldIndex))
	case EvtVarTypeUInt64:
		field = C.GetRenderedUInt64Value(C.PVOID(fields), C.int(fieldIndex))
	default:
		return 0, false
	}

	return uint64(field), true
}

// Get the signed integer at the given index. Returns false if the type of
// the field isn't EvtVarTypeSByte, EvtVarTypeInt16, EvtVarTypeInt32, EvtVarTypeInt64.
func RenderIntField(fields RenderedFields, fieldIndex EVT_SYSTEM_PROPERTY_ID) (int64, bool) {
	var field C.LONGLONG
	fieldType := C.GetRenderedValueType(C.PVOID(fields), C.int(fieldIndex))
	switch fieldType {
	case EvtVarTypeSByte:
		field = C.GetRenderedSByteValue(C.PVOID(fields), C.int(fieldIndex))
	case EvtVarTypeInt16:
		field = C.GetRenderedInt16Value(C.PVOID(fields), C.int(fieldIndex))
	case EvtVarTypeInt32:
		field = C.GetRenderedInt32Value(C.PVOID(fields), C.int(fieldIndex))
	case EvtVarTypeInt64:
		field = C.GetRenderedInt64Value(C.PVOID(fields), C.int(fieldIndex))
	default:
		return 0, false
	}

	return int64(field), true
}

// Get the formatted string that represents this message. This method wraps EvtFormatMessage.
func FormatMessage(eventPublisherHandle PublisherHandle, eventHandle EventHandle, format EVT_FORMAT_MESSAGE_FLAGS) (string, error) {
	cString := C.GetFormattedMessage(C.ULONGLONG(eventPublisherHandle), C.ULONGLONG(eventHandle), C.int(format))
	if cString == nil {
		return "", GetLastError()
	}
	value := C.GoString(cString)
	C.free(unsafe.Pointer(cString))
	return value, nil
}

// Get the formatted string for the last error which occurred. Wraps GetLastError and FormatMessage.
func GetLastError() error {
	errStr := C.GetLastErrorString()
	err := errors.New(C.GoString(errStr))
	C.LocalFree(C.HLOCAL(errStr))
	return err
}

// Render the system properties from the event and returns an array of properties.
// Properties can be accessed using RenderStringField, RenderIntField, RenderFileTimeField,
// or RenderUIntField depending on type. This buffer must be freed after use.
func RenderEventValues(renderContext SysRenderContext, eventHandle EventHandle) (RenderedFields, error) {
	values := RenderedFields(C.RenderEventValues(C.ULONGLONG(renderContext), C.ULONGLONG(eventHandle)))
	if values == nil {
		return nil, GetLastError()
	}
	return values, nil
}

// Render the event as XML.
func RenderEventXML(eventHandle EventHandle) (string, error) {
	xml := C.RenderEventXML(C.ULONGLONG(eventHandle))
	if xml == nil {
		return "", GetLastError()
	}
	xmlString := C.GoString(xml)
	C.free(unsafe.Pointer(xml))
	return xmlString, nil
}

// Get a handle that represents the publisher of the event, given the rendered event values.
func GetEventPublisherHandle(renderedFields RenderedFields) (PublisherHandle, error) {
	handle := PublisherHandle(C.GetEventPublisherHandle(C.PVOID(renderedFields)))
	if handle == 0 {
		return 0, GetLastError()
	}
	return handle, nil
}

// Close an event handle.
func CloseEventHandle(handle uint64) error {
	if C.CloseEvtHandle(C.ULONGLONG(handle)) != 1 {
		return GetLastError()
	}
	return nil
}

// Cancel pending actions on the event handle.
func CancelEventHandle(handle uint64) error {
	if C.CancelEvtHandle(C.ULONGLONG(handle)) != 1 {
		return GetLastError()
	}
	return nil
}

func Free(ptr unsafe.Pointer) {
	C.free(ptr)
}

/* Get the first event in the log, for testing */
func getTestEventHandle() (EventHandle, error) {
	handle := C.GetTestEventHandle()
	if handle == 0 {
		return 0, GetLastError()
	}
	return EventHandle(handle), nil
}

/* These are entry points for the callback to hand the pointer to Go-land.
   Note: handles are only valid within the callback. Don't pass them out. */

//export eventCallbackError
func eventCallbackError(errCode C.ULONGLONG, callbackWrapperKey unsafe.Pointer) {
	wrapper := getWrapper(uint32(uintptr(callbackWrapperKey)))
	watcher := wrapper.callback
	// The provided errCode can be looked up in the Microsoft System Error Code table:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681382(v=vs.85).aspx
	watcher.PublishError(fmt.Errorf("Event log callback got error code: %v", errCode))
}

//export eventCallback
func eventCallback(handle C.ULONGLONG, callbackWrapperKey unsafe.Pointer) {
	wrapper := getWrapper(uint32(uintptr(callbackWrapperKey)))
	watcher := wrapper.callback
	watcher.PublishEvent(EventHandle(handle), wrapper.subscribedChannel)
}
