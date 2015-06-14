// +build windows

package winlog

import (
  "syscall"
  "unsafe"
)

var (
  winevtDll *syscall.DLL
  evtCreateBookmark *syscall.Proc
  evtUpdateBookmark *syscall.Proc
  evtRender *syscall.Proc
  evtClose *syscall.Proc   
  evtCancel *syscall.Proc
  evtFormatMessage *syscall.Proc
  evtCreateRenderContext *syscall.Proc
  evtSubscribe *syscall.Proc
  evtQuery *syscall.Proc     
)

func init() {
  winevtDll = syscall.MustLoadDLL("wevtapi.dll")
  evtCreateBookmark = winevtDll.MustFindProc("EvtCreateBookmark")
  evtUpdateBookmark = winevtDll.MustFindProc("EvtUpdateBookmark")
  evtRender = winevtDll.MustFindProc("EvtRender")
  evtClose = winevtDll.MustFindProc("EvtClose")
  evtCancel = winevtDll.MustFindProc("EvtCancel")
  evtFormatMessage = winevtDll.MustFindProc("EvtFormatMessage")
  evtCreateRenderContext = winevtDll.MustFindProc("EvtCreateRenderContext")
  evtSubscribe = winevtDll.MustFindProc("EvtSubscribe")
  evtQuery = winevtDll.MustFindProc("EvtQuery")
}

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

type EVT_RENDER_FLAGS uint32
const (
	EvtRenderEventValues = iota
	EvtRenderEventXml
	EvtRenderBookmark



func EvtCreateBookmark(BookmarkXml *uint16) syscall.Handle {
  r1, _, _ := evtCreateBookmark.Call(uintptr(unsafe.Pointer(BookmarkXml)))
  return syscall.Handle(r1)
}

func EvtUpdateBookmark(Bookmark, Event syscall.Handle) uint32 {
  r1, _, _ := evtUpdateBookmark.Call(uintptr(Bookmark), uintptr(Event))
  return uint32(r1)
}

func EvtRender(Context, Fragment syscall.Handle, Flags, BufferSize uint32, Buffer *uint16, BufferUsed, PropertyCount *uint32) uint32 {
  r1, _, _ := evtRender.Call(uintptr(Context), uintptr(Fragment), uintptr(Flags), uintptr(BufferSize), uintptr(unsafe.Pointer(Buffer)), uintptr(unsafe.Pointer(BufferUsed)), uintptr(unsafe.Pointer(PropertyCount)))
  return uint32(r1) 
}

func EvtClose(Object syscall.Handle) uint32 {
  r1, _, _ := evtClose.Call(uintptr(Object))
  return uint32(r1)
}

func EvtFormatMessage(PublisherMetadata, Event syscall.Handle, MessageId, ValueCount uint32, Values *byte, Flags, BufferSize uint32, Buffer *uint16, BufferUsed *uint32) uint32 {
  r1, _, _ := evtFormatMessage.Call(uintptr(PublisherMetadata), uintptr(Event), uintptr(MessageId), uintptr(ValueCount), uintptr(unsafe.Pointer(Values)), uintptr(Flags), uintptr(BufferSize), uintptr(unsafe.Pointer(Buffer)), uintptr(unsafe.Pointer(BufferUsed)))
  return uint32(r1)
}

func EvtCreateRenderContext(ValuePathsCount uint32, ValuePaths uintptr, Flags uint32) syscall.Handle {
  r1, _, _ := evtCreateRenderContext.Call(uintptr(ValuePathsCount), ValuePaths, uintptr(Flags))
  return syscall.Handle(r1)
}

func EvtSubscribe (Session, SignalEvent syscall.Handle, ChannelPath, Query *uint16, Bookmark syscall.Handle, context uintptr, Callback uintptr, Flags uint32) syscall.Handle {
  r1, _, _ := evtSubscribe.Call(uintptr(Session), uintptr(SignalEvent), uintptr(unsafe.Pointer(ChannelPath)), uintptr(unsafe.Pointer(Query)), uintptr(Bookmark), context, Callback, uintptr(Flags))
  return syscall.Handle(r1)
}
 
func EvtQuery (Session syscall.Handle, Path, Query *uint16, Flags uint32) syscall.Handle {
  r1, _, _ := evtQuery.Call(uintptr(Session), uintptr(unsafe.Pointer(Path)), uintptr(unsafe.Pointer(Query)), uintptr(Flags))
  return syscall.Handle(r1)
}
