package winlog

/*
#cgo CPPFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include
#cgo CFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include
#cgo LDFLAGS: -l wevtapi -L C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/lib
#include "evt.h"
*/
import "C"
import (
  "time"
  "fmt"
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

func setupListener(channel string, watcher *WinLogWatcher) {
  cChan := C.CString(channel)
  C.setupListener(cChan, C.size_t(len(channel)), C.PVOID(watcher))
  C.free(unsafe.Pointer(cChan))
}

func getSystemRenderContext() unsafe.Pointer {
	return unsafe.Pointer(C.CreateSystemRenderContext())
}

func getError(err C.int) error {
  switch err {
  case 1:
    return fmt.Errorf("malloc failed")
  case 2:
  	//TODO: Get last error and get string representation
  	return fmt.Errorf("system error")
  default:
    return fmt.Errorf("unknown error %v", err)
  }
}

func renderStringField(fields C.PVOID, fieldIndex int) (string, bool, error) {
  fieldType := C.GetRenderedValueType(fields, C.int(fieldIndex))
  if fieldType != EvtVarTypeString {
    return "", false, nil
  }

  cString := C.GetRenderedStringValue(fields, C.int(fieldIndex))
  if cString == nil {
  	return "", false, nil
  }
  value := C.GoString(cString)
  C.free(unsafe.Pointer(cString))
  return value, true, nil
}

func renderFileTimeField(fields C.PVOID, fieldIndex int) (time.Time, bool, error) {
  fieldType := C.GetRenderedValueType(fields, C.int(fieldIndex))
  if fieldType != EvtVarTypeFileTime {
    return time.Time{}, false, nil
  }
  field := C.GetRenderedFileTimeValue(fields, C.int(fieldIndex))
  //fmt.Printf("Time: %v\n", int(field))
  return time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(100 * field) * time.Nanosecond), true, nil
}

func renderUIntField(fields C.PVOID, fieldIndex int) (uint, bool, error) {
  var field C.ulong
  fieldType := C.GetRenderedValueType(fields, C.int(fieldIndex))
  switch fieldType {
  case EvtVarTypeByte:
  	field = C.GetRenderedByteValue(fields, C.int(fieldIndex))
  case EvtVarTypeUInt16:
    field = C.GetRenderedUInt16Value(fields, C.int(fieldIndex))
  case EvtVarTypeUInt32:
    field = C.GetRenderedUInt32Value(fields, C.int(fieldIndex))
  case EvtVarTypeUInt64:
    field = C.GetRenderedUInt64Value(fields, C.int(fieldIndex))
  default:
    return 0, false, nil
  }

  return uint(field), true, nil
}

func renderIntField(fields C.PVOID, fieldIndex int) (int, bool, error) {
  var field C.long
  fieldType := C.GetRenderedValueType(fields, C.int(fieldIndex))
  switch fieldType {
  case EvtVarTypeByte:
  	field = C.GetRenderedSByteValue(fields, C.int(fieldIndex))
  case EvtVarTypeInt16:
    field = C.GetRenderedInt16Value(fields, C.int(fieldIndex))
  case EvtVarTypeInt32:
    field = C.GetRenderedInt32Value(fields, C.int(fieldIndex))
  case EvtVarTypeInt64:
    field = C.GetRenderedInt64Value(fields, C.int(fieldIndex))
  default:
    return 0, false, nil
  }

  return int(field), true, nil
}

func formatMessage(eventPublisherHandle, eventHandle C.PVOID, format int) (string, error) {
  cString := C.GetFormattedMessage(eventPublisherHandle, eventHandle, C.int(format))
  if cString == nil {
  	return "", fmt.Errorf("Null message")
  }
  value := C.GoString(cString)
  C.free(unsafe.Pointer(cString))
  return value, nil
}

func (self *WinLogWatcher) eventCallback(handle C.HANDLE) {
  renderedFields := C.RenderEventValues(C.PVOID(self.renderContext), C.PVOID(handle))
  if renderedFields == nil {
      return
  }
  
  publisherHandle := C.GetEventPublisherHandle(C.PVOID(renderedFields))
  computerName, _, _ := renderStringField(C.PVOID(renderedFields), EvtSystemComputer)
  providerName, _, _ := renderStringField(C.PVOID(renderedFields), EvtSystemProviderName)
  channel, _, _ := renderStringField(C.PVOID(renderedFields), EvtSystemChannel)
  level, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemLevel)
  task, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemTask)
  opcode, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemOpcode)
  recordId, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemEventRecordId)
  qualifiers, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemQualifiers)
  eventId, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemEventID)
  processId, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemProcessID)
  threadId, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemThreadID)
  version, _, _ := renderUIntField(C.PVOID(renderedFields), EvtSystemVersion)
  created, _, _ := renderFileTimeField(C.PVOID(renderedFields), EvtSystemTimeCreated)
  msgText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageEvent)
  lvlText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageLevel)
  taskText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageTask)
  providerText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageProvider)
  opcodeText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageOpcode)
  channelText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageChannel)
  
  C.free(unsafe.Pointer(renderedFields))
  event := WinLogEvent {
    Msg: msgText,
    ProviderName: providerName,
    Channel: channel,
    ComputerName: computerName, 
    Level: level,
    Task: task,
    Opcode: opcode,
    RecordId: recordId,
    EventId: eventId,
    Qualifiers: qualifiers,
    ProcessId: processId,
    ThreadId: threadId,
    Version: version,
    Created: created,

    LevelName: lvlText,
    TaskText: taskText,
    ProviderText: providerText,
    OpcodeText: opcodeText,
    ChannelText: channelText,
  }
  fmt.Printf("EvenT: %v\n", event)
}

func (self *WinLogWatcher) errorCallback(handle C.HANDLE) {
  fmt.Printf("Got error %v\n", uintptr(handle));
}

/* These are entry points for the callback to hand the pointer to Go-land.
   Note: handles are only valid within the callback. Don't pass them out. */

//export EventCallbackError
func EventCallbackError(handle C.HANDLE, logWatcher unsafe.Pointer) {
  watcher := (*WinLogWatcher)(logWatcher)
  watcher.errorCallback(handle)
}

//export EventCallback
func EventCallback(handle C.HANDLE, logWatcher unsafe.Pointer) {
  watcher := (*WinLogWatcher)(logWatcher)
  watcher.eventCallback(handle)
}