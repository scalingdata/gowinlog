package winlog

/*
#cgo CPPFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include
#cgo CFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include
#cgo LDFLAGS: -l wevtapi -L C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/lib
#include "evt.h"
*/
import "C"
import (
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
  	fmt.Printf("Not string type\n")
    return "", false, nil
  }

  cString := C.GetRenderedStringValue(fields, C.int(fieldIndex))
  fmt.Printf("rsf Got: %v\n", cString)
  if cString == nil {
  	fmt.Printf("Error\n")
  	return "", false, nil
  }
  value := C.GoString(cString)
  C.freeLog(C.PVOID(cString))
  return value, true, nil
}

func formatMessage(eventPublisherHandle, eventHandle C.PVOID, format int) (string, error) {
  cString := C.GetFormattedMessage(eventPublisherHandle, eventHandle, C.int(format))
  fmt.Printf("fm Got: %v\n", cString)
  if cString == nil {
  	return "", fmt.Errorf("Null message")
  }
  value := C.GoString(cString)
  C.freeLog(C.PVOID(cString))
  return value, nil
}

func (self *WinLogWatcher) eventCallback(handle C.HANDLE) {
  renderedFields := C.RenderEventValues(C.PVOID(self.renderContext), C.PVOID(handle))
  fmt.Printf("Rendered fields got: %v\n", renderedFields)
  if renderedFields == nil {
      fmt.Printf("Error while getting fields")
      return
  }
  
  publisherHandle := C.GetEventPublisherHandle(C.PVOID(renderedFields))
  fmt.Printf("Publisher handler %v\n", publisherHandle)
/*
  computerName, _, _ := renderStringField(C.PVOID(renderedFields), EvtSystemComputer)
  fmt.Printf("computerName: %v\n", computerName)

  providerName, _, _ := renderStringField(C.PVOID(renderedFields), EvtSystemProviderName)
  fmt.Printf("Provider: %v, computerName: %v\n", providerName, computerName)

  channel, _, _ := renderStringField(C.PVOID(renderedFields), EvtSystemChannel)
  fmt.Printf("Provider: %v, channel: %v, computerName: %v\n", providerName, channel, computerName)

  msgText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageEvent)
  fmt.Printf("Provider: %v, channel: %v, computerName: %v, msg: %v\n", providerName, channel, computerName, msgText)

  //lvlText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageLevel)
  //taskText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageTask)
  providerText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageProvider)
  fmt.Printf("Provider: %v, channel: %v, computerName: %v, msg: %v, providerText: %v \n", providerName, channel, computerName, msgText, providerText)

  opcodeText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageOpcode)
  fmt.Printf("Provider: %v, channel: %v, computerName: %v, msg: %v, opcodeText: %v, providerText: %v \n", providerName, channel, computerName, msgText, opcodeText, providerText)

  channelText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageChannel)
  

  fmt.Printf("Provider: %v, channel: %v, computerName: %v, msg: %v, channelText: %v, opcodeText: %v, providerText: %v \n", providerName, channel, computerName, msgText, channelText, opcodeText, providerText)
  */
  msgText, _ := formatMessage(C.PVOID(publisherHandle), C.PVOID(handle), EvtFormatMessageEvent);
  fmt.Printf("Msg: %v\n", msgText);
  C.freeLog(C.PVOID(renderedFields))
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