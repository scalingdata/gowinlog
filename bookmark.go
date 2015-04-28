package winlog

/*
#cgo CPPFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include
#cgo CFLAGS: -I C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/include
#cgo LDFLAGS: -l wevtapi -L C:/mingw-w64/x86_64-4.9.2-posix-seh-rt_v4-rev2/mingw64/x86_64-w64-mingw32/lib
#include "bookmark.h"
*/
import "C"
import (
	"unsafe"
)

func CreateBookmark() (uint64, error) {
	bookmark := uint64(C.CreateBookmark())
	if bookmark == 0 {
		return 0, GetLastError()
	}
	return bookmark, nil
}

func CreateBookmarkFromXml(xmlString string) (uint64, error) {
	cString := C.CString(xmlString)
	bookmark := C.CreateBookmarkFromXML(cString)
	C.free(unsafe.Pointer(cString))
	if bookmark == 0 {
		return 0, GetLastError()
	}
	return uint64(bookmark), nil
}

func UpdateBookmark(bookmarkHandle, eventHandle uint64) error {
	if C.UpdateBookmark(C.ULONGLONG(bookmarkHandle), C.ULONGLONG(eventHandle)) == 0 {
		return GetLastError()
	}
	return nil
}

func RenderBookmark(bookmarkHandle uint64) (string, error) {
	cString := C.RenderBookmark(C.ULONGLONG(bookmarkHandle))
	if cString == nil {
		return "", GetLastError()
	}
	bookmarkXml := C.GoString(cString)
	C.free(unsafe.Pointer(cString))
	return bookmarkXml, nil
}
