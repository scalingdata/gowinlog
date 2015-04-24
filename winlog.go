package winlog

/*
#cgo CPPFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include -Wno-pointer-to-int-cast
#cgo CFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include -Wno-pointer-to-int-cast
#cgo LDFLAGS: -l wevtapi -L C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/lib
#include "evt.h"
*/
import "C"
import (
  "fmt"
  "time"
  "unsafe"
)

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

type WinLogEvent struct {
  Msg string
  Provider string
  EventSource string
  EventId int
  Version int
  Level int
  Opcode int
  Keywords []string
  Created time.Time
  RecordId int
  Channel string
  Computer string
}

type WinLogWatcher struct {
  errChan chan error
  eventChan chan *WinLogEvent

  renderContext unsafe.Pointer
}

func NewWinLogWatcher() (*WinLogWatcher, error) {
  cHandle := C.CreateSystemRenderContext()
  if unsafe.Pointer(cHandle) == nil {
    return nil, fmt.Errorf("Error getting render context %v", cHandle)
  }
  return &WinLogWatcher {
    errChan: make(chan error),
    eventChan: make(chan *WinLogEvent),
    renderContext: unsafe.Pointer(cHandle),
  }, nil
}

func (self *WinLogWatcher) Subscribe(channel string) {
  cChan := C.CString(channel)
  C.setupListener(cChan, C.size_t(len(channel)), C.PVOID(unsafe.Pointer(self)))
}

func (self *WinLogWatcher) errorCallback(handle unsafe.Pointer) {
  fmt.Printf("Got error %v\n", uintptr(handle));
}

func renderStringField(fields C.PVOID, fieldIndex int) (string, bool){
  fieldType := C.GetRenderedValueType(fields, C.int(fieldIndex))
  if fieldType != EvtVarTypeString {
    return "", false
  }
  cString := C.GetRenderedStringValue(fields, C.int(fieldIndex))
  value := C.GoString(cString)
  C.free(unsafe.Pointer(cString))
  return value, true
}

func (self *WinLogWatcher) eventCallback(handle unsafe.Pointer) {
  var renderedFields unsafe.Pointer
  var fields C.int
  var dataLen C.int

  err := C.RenderEventValues(C.PVOID(self.renderContext), C.PVOID(handle), (*C.PVOID)(&renderedFields), &dataLen, &fields)
  if err != 0 {
      fmt.Printf("Error while getting fields %v", err)
  }
  fmt.Printf("Got fields %v\n", fields)
  providerName, _ := renderStringField(C.PVOID(renderedFields), 0)
  channel, _ := renderStringField(C.PVOID(renderedFields), 14)
  computerName, _ := renderStringField(C.PVOID(renderedFields), 15)
  fmt.Printf("Provider: %v, channel: %v, computerName: %v\n", providerName, channel, computerName)
  C.free(renderedFields)
}

/* These are entry points for the callback to hand the pointer to Go-land.
   Note: handles are only valid within the callback. Don't pass them out. */

//export EventCallbackError
func EventCallbackError(handle unsafe.Pointer, logWatcher unsafe.Pointer) {
  watcher := (*WinLogWatcher)(logWatcher)
  watcher.errorCallback(handle)
}

//export EventCallback
func EventCallback(handle unsafe.Pointer, logWatcher unsafe.Pointer) {
  watcher := (*WinLogWatcher)(logWatcher)
  watcher.eventCallback(handle)
}