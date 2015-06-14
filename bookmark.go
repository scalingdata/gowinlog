// +build windows

package winlog

import (
	"syscall"
)

func CreateBookmark() (BookmarkHandle, error) {
	bookmark := BookmarkHandle(EvtCreateBookmark(nil))
	if bookmark == 0 {
		return 0, GetLastError()
	}
	return bookmark, nil
}

func CreateBookmarkFromXml(xmlString string) (BookmarkHandle, error) {
	wideXmlString, err := syscall.UTF16PtrFromString(xmlString)
	if err != nil {
		return 0, err
	}
	bookmark := BookmarkHandle(EvtCreateBookmark(wideXmlString))
	if bookmark == 0 {
		return 0, GetLastError()
	}
	return bookmark, nil
}

func UpdateBookmark(bookmarkHandle BookmarkHandle, eventHandle EventHandle) error {
	err := EvtUpdateBookmark(syscall.Handle(bookmarkHandle), syscall.Handle(eventHandle))
	if err == 0 {
		return GetLastError()
	}
	return nil
}

func RenderBookmark(bookmarkHandle BookmarkHandle) (string, error) {
	var dwUsed uint32
	var dwProps uint32
	EvtRender(0, syscall.Handle(bookmarkHandle), EvtRenderBookmark, 0, nil, &dwUsed, &dwProps)
	buf := make([]uint16, dwUsed)
	err := EvtRender(0, syscall.Handle(bookmarkHandle), EvtRenderBookmark, uint32(len(buf)), &buf[0], &dwUsed, &dwProps)
	if err == 0 {
		return "", GetLastError()
	} 
	return syscall.UTF16ToString(buf), nil
}
