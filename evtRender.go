package winlog

/*
#cgo LDFLAGS: -l wevtapi
#include "evt.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"time"
	"unsafe"
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

type SysRenderContext uint64
type ListenerHandle uint64
type PublisherHandle uint64
type EventHandle uint64
type RenderedFields unsafe.Pointer

type LogEventCallback interface {
  PublishError(error)
  PublishEvent(EventHandle)
}

type logEventCallbackWrapper struct {
	callback LogEventCallback
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

// Get a handle for a event log subscription on the given channel. Will begin at the
// next event recieved on the channel after the subscription is registered. 
// The resulting handle must be closed with CloseEventHandle.
func CreateListenerFromNow(channel string, watcher LogEventCallback) (ListenerHandle, error) {
	cChan := C.CString(channel)
	wrapper := &logEventCallbackWrapper{watcher}
	listenerHandle := C.CreateListenerFromNow(cChan, C.PVOID(wrapper))
	C.free(unsafe.Pointer(cChan))
	if listenerHandle == 0 {
		return 0, GetLastError()
	}
	return ListenerHandle(listenerHandle), nil
}

// Get a handle for an event log subscription on the given channel. Will begin at the
// bookmarked event, or the closest possible event if the log has been truncated.
// The resulting handle must be closed with CloseEventHandle.
func CreateListenerFromBookmark(channel string, watcher LogEventCallback, bookmarkHandle BookmarkHandle) (ListenerHandle, error) {
	cChan := C.CString(channel)
	wrapper := &logEventCallbackWrapper{watcher}
	listenerHandle := C.CreateListenerFromBookmark(cChan, C.PVOID(wrapper), C.ULONGLONG(bookmarkHandle))
	C.free(unsafe.Pointer(cChan))
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
	field := C.GetRenderedFileTimeValue(C.PVOID(fields), C.int(fieldIndex))
	return time.Unix(int64(field), 0), true
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
func RenderEventValues(renderContext SysRenderContext, eventHandle EventHandle) RenderedFields {
	return RenderedFields(C.RenderEventValues(C.ULONGLONG(renderContext), C.ULONGLONG(eventHandle)))
}

// Get a handle that represents the publisher of the event, given the rendered event values.
func GetEventPublisherHandle(renderedFields RenderedFields) PublisherHandle {
	return PublisherHandle(C.GetEventPublisherHandle(C.PVOID(renderedFields)))
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

/* These are entry points for the callback to hand the pointer to Go-land.
   Note: handles are only valid within the callback. Don't pass them out. */

//export eventCallbackError
func eventCallbackError(handle C.ULONGLONG, logWatcher unsafe.Pointer) {
	watcher := (*logEventCallbackWrapper)(logWatcher).callback
	watcher.PublishError(fmt.Errorf("Event log callback got error: %v", GetLastError()))
}

//export eventCallback
func eventCallback(handle C.ULONGLONG, logWatcher unsafe.Pointer) {
	watcher := (*logEventCallbackWrapper)(logWatcher).callback
	watcher.PublishEvent(EventHandle(handle))
}
