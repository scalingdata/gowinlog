package winlog

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

/* Convenience functions to get values out of
   an array of EvtVariant structures */

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

type evtVariant struct {
	Data  uint64
	Count uint32
	Type  uint32
}

type fileTime struct {
	lowDateTime  uint32
	highDateTime uint32
}

type EvtVariant []byte

/* Given a byte array from EvtRender, make an EvtVariant.
   EvtVariant wraps an array of variables. */
func NewEvtVariant(buffer []byte) EvtVariant {
	return EvtVariant(buffer)
}

func (e EvtVariant) elemAt(index uint32) *evtVariant {
	return (*evtVariant)(unsafe.Pointer(uintptr(16*index) + uintptr(unsafe.Pointer(&e[0]))))
}

/* Return the string value of the variable at `index`. If the
   variable isn't a string, an error is returned */
func (e EvtVariant) String(index uint32) (string, error) {
	elem := e.elemAt(index)
	if elem.Type != EvtVarTypeString {
		return "", fmt.Errorf("EvtVariant at index %v was not of type string, type was %v", index, elem.Type)
	}
	wideString := (*[1 << 30]uint16)(unsafe.Pointer(uintptr(elem.Data)))
	str := syscall.UTF16ToString(wideString[0 : elem.Count+1])
	return str, nil
}

/* Return the unsigned integer value at `index`. If the variable
   isn't a Byte, UInt16, UInt32 or UInt64 an error is returned. */
func (e EvtVariant) Uint(index uint32) (uint64, error) {
	elem := e.elemAt(index)
	switch elem.Type {
	case EvtVarTypeByte:
		return uint64(byte(elem.Data)), nil
	case EvtVarTypeUInt16:
		return uint64(uint16(elem.Data)), nil
	case EvtVarTypeUInt32:
		return uint64(uint32(elem.Data)), nil
	case EvtVarTypeUInt64:
		return uint64(elem.Data), nil
	default:
		return 0, fmt.Errorf("EvtVariant at index %v was not an unsigned integer, type is %v", index, elem.Type)
	}
}

/* Return the integer value at `index`. If the variable
   isn't a SByte, Int16, Int32 or Int64 an error is returned. */
func (e EvtVariant) Int(index uint32) (int64, error) {
	elem := e.elemAt(index)
	switch elem.Type {
	case EvtVarTypeSByte:
		return int64(byte(elem.Data)), nil
	case EvtVarTypeInt16:
		return int64(int16(elem.Data)), nil
	case EvtVarTypeInt32:
		return int64(int32(elem.Data)), nil
	case EvtVarTypeInt64:
		return int64(elem.Data), nil
	default:
		return 0, fmt.Errorf("EvtVariant at index %v was not an integer, type is %v", index, elem.Type)
	}
}

/* Return the FileTime at `index`, converted to Time.time. If the
   variable isn't a FileTime an error is returned */
func (e EvtVariant) FileTime(index uint32) (time.Time, error) {
	elem := e.elemAt(index)
	if elem.Type != EvtVarTypeFileTime {
		return time.Now(), fmt.Errorf("EvtVariant at index %v was not of type FileTime, type was %v", index, elem.Type)
	}
	var t *fileTime = (*fileTime)(unsafe.Pointer(&elem.Data))
	timeSecs := (((int64(t.highDateTime) << 32) | int64(t.lowDateTime)) / 10000000) - int64(11644473600)
	timeNano := (((int64(t.highDateTime) << 32) | int64(t.lowDateTime)) % 10000000) * 100
	return time.Unix(timeSecs, timeNano), nil
}

/* Return whether the variable was actually set, or whether it
   has null type */
func (e EvtVariant) IsNull(index uint32) bool {
	return e.elemAt(index).Type == EvtVarTypeNull
}
