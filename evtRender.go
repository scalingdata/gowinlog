package winlog

/*
#cgo CPPFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include
#cgo CFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include
#cgo LDFLAGS: -l wevtapi -L C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/lib
#include "evt.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"time"
	"unsafe"
)

/* Types for GetRenderedValueType */
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

func GetSystemRenderContext() uint64 {
	return uint64(C.CreateSystemRenderContext())
}

func CreateListenerFromNow(channel string, watcher *WinLogWatcher) (uint64, error) {
	cChan := C.CString(channel)
	listenerHandle := C.CreateListenerFromNow(cChan, C.PVOID(watcher))
	C.free(unsafe.Pointer(cChan))
	if listenerHandle == 0 {
		return 0, GetLastError()
	}
	return uint64(listenerHandle), nil
}

func CreateListenerFromBookmark(channel string, watcher *WinLogWatcher, bookmarkHandle uint64) (uint64, error) {
	cChan := C.CString(channel)
	listenerHandle := C.CreateListenerFromNow(cChan, C.PVOID(watcher))
	C.free(unsafe.Pointer(cChan))
	if listenerHandle == 0 {
		return 0, GetLastError()
	}
	return uint64(listenerHandle), nil
}

func RenderStringField(fields unsafe.Pointer, fieldIndex int) (string, bool) {
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

func RenderFileTimeField(fields unsafe.Pointer, fieldIndex int) (time.Time, bool) {
	fieldType := C.GetRenderedValueType(C.PVOID(fields), C.int(fieldIndex))
	if fieldType != EvtVarTypeFileTime {
		return time.Time{}, false
	}
	field := C.GetRenderedFileTimeValue(C.PVOID(fields), C.int(fieldIndex))
	return time.Unix(int64(field), 0), true
}

func RenderUIntField(fields unsafe.Pointer, fieldIndex int) (uint64, bool) {
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

func RenderIntField(fields unsafe.Pointer, fieldIndex int) (int64, bool) {
	var field C.LONGLONG
	fieldType := C.GetRenderedValueType(C.PVOID(fields), C.int(fieldIndex))
	switch fieldType {
	case EvtVarTypeByte:
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

func FormatMessage(eventPublisherHandle, eventHandle uint64, format int) (string, error) {
	cString := C.GetFormattedMessage(C.ULONGLONG(eventPublisherHandle), C.ULONGLONG(eventHandle), C.int(format))
	if cString == nil {
		return "", GetLastError()
	}
	value := C.GoString(cString)
	C.free(unsafe.Pointer(cString))
	return value, nil
}

func GetLastError() error {
	errStr := C.GetLastErrorString()
	err := errors.New(C.GoString(errStr))
	C.LocalFree(C.HLOCAL(errStr))
	return err
}

func RenderEventValues(renderContext, eventHandle uint64) unsafe.Pointer {
	return unsafe.Pointer(C.RenderEventValues(C.ULONGLONG(renderContext), C.ULONGLONG(eventHandle)))
}

func GetEventPublisherHandle(renderedFields unsafe.Pointer) uint64 {
	return uint64(C.GetEventPublisherHandle(C.PVOID(renderedFields)))
}

func CloseEventHandle(handle uint64) error {
	if C.CloseEvtHandle(C.ULONGLONG(handle)) != 1 {
		return GetLastError()
	}
	return nil
}

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
	watcher := (*WinLogWatcher)(logWatcher)
	watcher.publishError(fmt.Errorf("Event log callback got error: %v", GetLastError()))
}

//export eventCallback
func eventCallback(handle C.ULONGLONG, logWatcher unsafe.Pointer) {
	watcher := (*WinLogWatcher)(logWatcher)
	watcher.publishEvent(uint64(handle))
}
