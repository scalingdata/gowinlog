// +build windows

package winlog

import (
	"syscall"
)

/* Create a new, empty bookmark. Bookmark handles must be closed with CloseEventHandle. */
func CreateBookmark() (BookmarkHandle, error) {
	bookmark, err := EvtCreateBookmark(nil)
	if err != nil {
		return 0, err
	}
	return BookmarkHandle(bookmark), nil
}

/* Create a bookmark from a XML-serialized bookmark. Bookmark handles must be closed with CloseEventHandle. */
func CreateBookmarkFromXml(xmlString string) (BookmarkHandle, error) {
	wideXmlString, err := syscall.UTF16PtrFromString(xmlString)
	if err != nil {
		return 0, err
	}
	bookmark, err := EvtCreateBookmark(wideXmlString)
	if bookmark == 0 {
		return 0, err
	}
	return BookmarkHandle(bookmark), nil
}

/* Update a bookmark to store the channel and ID of the given event */
func UpdateBookmark(bookmarkHandle BookmarkHandle, eventHandle EventHandle) error {
	return EvtUpdateBookmark(syscall.Handle(bookmarkHandle), syscall.Handle(eventHandle))
}

/* Serialize the bookmark as XML */
func RenderBookmark(bookmarkHandle BookmarkHandle) (string, error) {
	var dwUsed uint32
	var dwProps uint32
	EvtRender(0, syscall.Handle(bookmarkHandle), EvtRenderBookmark, 0, nil, &dwUsed, &dwProps)
	buf := make([]uint16, dwUsed)
	err := EvtRender(0, syscall.Handle(bookmarkHandle), EvtRenderBookmark, uint32(len(buf)), &buf[0], &dwUsed, &dwProps)
	if err != nil {
		return "", err
	}
	return syscall.UTF16ToString(buf), nil
}
