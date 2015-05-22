// +build windows

package winlog

/*
#cgo LDFLAGS: -l wevtapi
#include "event.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"time"
	"unsafe"
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

// Get a handle to a render context which will render properties from the System element.
// Wraps EvtCreateRenderContext() with Flags = EvtRenderContextSystem. The resulting
// handle must be closed with CloseEventHandle.
func GetSystemRenderContext() (RenderContext, error) {
	context := RenderContext(C.CreateSystemRenderContext())
	if context == 0 {
		return 0, GetLastError()
	}
	return context, nil
}

// Get a handle to a render context which will render properties from the UserData or EventData element.
// Wraps EvtCreateRenderContext() with Flags = EvtRenderContextSystem. The resulting
// handle must be closed with CloseEventHandle.
func GetUserRenderContext() (RenderContext, error) {
	context := RenderContext(C.CreateUserRenderContext())
	if context == 0 {
		return 0, GetLastError()
	}
	return context, nil
}

// Get a handle for a event log subscription on the given channel.
// The resulting handle must be closed with CloseEventHandle.
func CreateListener(channel string, startpos EVT_SUBSCRIBE_FLAGS, watcher *LogEventCallbackWrapper) (ListenerHandle, error) {
	cChan := C.CString(channel)
	listenerHandle := C.CreateListener(cChan, C.int(startpos), C.PVOID(watcher))
	C.free(unsafe.Pointer(cChan))
	if listenerHandle == 0 {
		return 0, GetLastError()
	}
	return ListenerHandle(listenerHandle), nil
}

// Get a handle for an event log subscription on the given channel. Will begin at the
// bookmarked event, or the closest possible event if the log has been truncated.
// The resulting handle must be closed with CloseEventHandle.
func CreateListenerFromBookmark(channel string, watcher *LogEventCallbackWrapper, bookmarkHandle BookmarkHandle) (ListenerHandle, error) {
	cChan := C.CString(channel)
	listenerHandle := C.CreateListenerFromBookmark(cChan, C.PVOID(watcher), C.ULONGLONG(bookmarkHandle))
	C.free(unsafe.Pointer(cChan))
	if listenerHandle == 0 {
		return 0, GetLastError()
	}
	return ListenerHandle(listenerHandle), nil
}

// Get the Go string for the field at the given index. Returns
// false if the type of the field isn't EvtVarTypeString.
func RenderStringField(fields RenderedFields, fieldIndex int) (string, bool) {
	fieldType := C.GetRenderedValueType((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	if fieldType != EvtVarTypeString {
		return "", false
	}

	cString := C.GetRenderedStringValue((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	if cString == nil {
		return "", false
	}

	value := C.GoString(cString)
	C.free(unsafe.Pointer(cString))
	return value, true
}

// Get the timestamp of the field at the given index. Returns false if the
// type of the field isn't EvtVarTypeFileTime.
func RenderFileTimeField(fields RenderedFields, fieldIndex int) (time.Time, bool) {
	fieldType := C.GetRenderedValueType((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	if fieldType != EvtVarTypeFileTime {
		return time.Time{}, false
	}
	field := C.GetRenderedFileTimeValue((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	return time.Unix(int64(field), 0), true
}

// Get the unsigned integer at the given index. Returns false if the field
// type isn't EvtVarTypeByte, EvtVarTypeUInt16, EvtVarTypeUInt32, or EvtVarTypeUInt64.
func RenderUIntField(fields RenderedFields, fieldIndex int) (uint64, bool) {
	var field C.ULONGLONG
	fieldType := C.GetRenderedValueType((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	switch fieldType {
	case EvtVarTypeByte:
		field = C.GetRenderedByteValue((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	case EvtVarTypeUInt16:
		field = C.GetRenderedUInt16Value((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	case EvtVarTypeUInt32:
		field = C.GetRenderedUInt32Value((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	case EvtVarTypeUInt64:
		field = C.GetRenderedUInt64Value((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	default:
		return 0, false
	}

	return uint64(field), true
}

// Get the signed integer at the given index. Returns false if the type of
// the field isn't EvtVarTypeSByte, EvtVarTypeInt16, EvtVarTypeInt32, EvtVarTypeInt64.
func RenderIntField(fields RenderedFields, fieldIndex int) (int64, bool) {
	var field C.LONGLONG
	fieldType := C.GetRenderedValueType((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	switch fieldType {
	case EvtVarTypeSByte:
		field = C.GetRenderedSByteValue((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	case EvtVarTypeInt16:
		field = C.GetRenderedInt16Value((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	case EvtVarTypeInt32:
		field = C.GetRenderedInt32Value((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	case EvtVarTypeInt64:
		field = C.GetRenderedInt64Value((*C.RenderedFields)(fields).fields, C.int(fieldIndex))
	default:
		return 0, false
	}

	return int64(field), true
}

// Get a field of an unknown type, and render it as a string
func GetRenderedFieldAsString(renderedFields RenderedFields, fieldIndex int) (string) {
	fieldType := C.GetRenderedValueType((*C.RenderedFields)(renderedFields).fields, C.int(fieldIndex))
	var value interface{}
  if fieldType == EvtVarTypeSByte || fieldType == EvtVarTypeInt16 || fieldType == EvtVarTypeInt32 || fieldType == EvtVarTypeInt64 {
  	value, _ = RenderIntField(renderedFields, fieldIndex)
  } else if fieldType == EvtVarTypeByte || fieldType == EvtVarTypeUInt16 || fieldType == EvtVarTypeUInt32 || fieldType == EvtVarTypeUInt64 {
  	value, _ = RenderUIntField(renderedFields, fieldIndex)
  } else if fieldType == EvtVarTypeString {
  	value, _ = RenderStringField(renderedFields, fieldIndex)
  } else if fieldType == EvtVarTypeFileTime {
  	value, _ = RenderFileTimeField(renderedFields, fieldIndex)
  } else {
  	value = ""
  }
  return fmt.Sprintf("%v", value)
}

func RenderAllFieldsAsStrings(renderedFields RenderedFields) ([]string) {
  fieldCount := (*C.RenderedFields)(renderedFields).nFields
  fieldStrings := make([]string, 0, fieldCount)

  for i := 0 ; i < int(fieldCount); i++ {
    fieldStrings = append(fieldStrings, GetRenderedFieldAsString(renderedFields, i))
  }

  return fieldStrings
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

// Render the properties from the event and returns an array of properties.
// Properties can be accessed using RenderStringField, RenderIntField, RenderFileTimeField,
// or RenderUIntField depending on type. This buffer must be freed after use.
func RenderEventValues(renderContext RenderContext, eventHandle EventHandle) (RenderedFields, error) {
	values := C.RenderEventValues(C.ULONGLONG(renderContext), C.ULONGLONG(eventHandle))
	if values == nil {
		return nil, GetLastError()
	}
	return RenderedFields(values), nil
}

// Get a handle that represents the publisher of the event, given the rendered event values.
func GetEventPublisherHandle(renderedFields RenderedFields) (PublisherHandle, error) {
	fields := C.PVOID((*C.RenderedFields)(renderedFields).fields)
	handle := PublisherHandle(C.GetEventPublisherHandle(fields))
	if handle == 0 {
		return 0, GetLastError()
	}
	return handle, nil
}

// Free the inner array, and then the wrapper struct
func FreeRenderedFields(renderedFields RenderedFields) {
	fields := unsafe.Pointer((*C.RenderedFields)(renderedFields).fields)
	C.free(fields)
	C.free(unsafe.Pointer(renderedFields))
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
func eventCallbackError(handle C.ULONGLONG, logWatcher unsafe.Pointer) {
	watcher := (*LogEventCallbackWrapper)(logWatcher).callback
	watcher.PublishError(fmt.Errorf("Event log callback got error: %v", GetLastError()))
}

//export eventCallback
func eventCallback(handle C.ULONGLONG, logWatcher unsafe.Pointer) {
	watcher := (*LogEventCallbackWrapper)(logWatcher).callback
	watcher.PublishEvent(EventHandle(handle))
}
