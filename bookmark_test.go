// +build windows

package winlog

import (
	"encoding/xml"
	. "testing"
)

type bookmarkListXml struct {
	XMLName   xml.Name      `xml:"BookmarkList"`
	Bookmarks []bookmarkXml `xml:"Bookmark"`
}

type bookmarkXml struct {
	XMLName  xml.Name `xml:"Bookmark"`
	RecordId uint64   `xml:"RecordId,attr"`
	Channel  string   `xml:"Channel,attr"`
}

func TestSerializeBookmark(t *T) {
	testBookmarkXml := "<BookmarkList>\r\n  <Bookmark Channel='Application' RecordId='10811' IsCurrent='true'/>\r\n</BookmarkList>"
	bookmark, err := CreateBookmarkFromXml(testBookmarkXml)
	if err != nil {
		t.Fatal(err)
	}
	defer CloseEventHandle(uint64(bookmark))
	xmlString, err := RenderBookmark(bookmark)
	if err != nil {
		t.Fatal(err)
	}
	if xmlString != testBookmarkXml {
		t.Fatalf("Serialized bookmark not equal to original %q != %q", testBookmarkXml, xmlString)
	}
}

func TestCreateInvalidXml(t *T) {
	testBookmarkXml := "<BookmarkList>\r\n  <Bookmark Channel='Application' RecordId='10811' IsCurrent='true'/>"
	bookmark, err := CreateBookmarkFromXml(testBookmarkXml)
	if err == nil {
		t.Fatal("No error from invalid bookmark XML")
	}
	if bookmark != 0 {
		t.Fatal("Got handle from invalid bookmark XML")
	}
}

func TestUpdateBookmark(t *T) {
	// Create a bookmark and update it with an event
	bookmark, err := CreateBookmark()
	if err != nil {
		t.Fatal(err)
	}
	defer CloseEventHandle(uint64(bookmark))
	event, err := getTestEventHandle()
	if err != nil {
		t.Fatal(err)
	}
	err = UpdateBookmark(bookmark, event)
	if err != nil {
		t.Fatal(err)
	}

	// Decode the XML bookmark
	xmlString, err := RenderBookmark(bookmark)
	if err != nil {
		t.Fatal(err)
	}
	var bookmarkStruct bookmarkListXml
	if err := xml.Unmarshal([]byte(xmlString), &bookmarkStruct); err != nil {
		t.Fatal(err)
	}
	if len(bookmarkStruct.Bookmarks) != 1 {
		t.Fatalf("Got %v bookmarks, expected 1", len(bookmarkStruct.Bookmarks))
	}

	// Extract the corresponding Event properties
	renderContext, err := GetSystemRenderContext()
	if err != nil {
		t.Fatal(err)
	}
	defer CloseEventHandle(uint64(renderContext))
	renderedFields, err := RenderEventValues(renderContext, event)
	if err != nil {
		t.Fatal(err)
	}
	channel, _ := renderedFields.String(EvtSystemChannel)
	eventId, _ := renderedFields.Uint(EvtSystemEventRecordId)
	bookmarkChannel := bookmarkStruct.Bookmarks[0].Channel
	bookmarkId := bookmarkStruct.Bookmarks[0].RecordId

	// Check bookmark channel and record ID match
	if channel != bookmarkChannel {
		t.Fatalf("Bookmark channel %v not equal to event channel %v", bookmarkChannel, channel)
	}
	if bookmarkId != eventId {
		t.Fatalf("Bookmark recordId %v not equal to event id %v", bookmarkId, eventId)
	}
}

func TestUpdateInvalidBookmark(t *T) {
	// Create a bookmark and update it with a NULL event
	bookmark, err := CreateBookmark()
	if err != nil {
		t.Fatal(err)
	}
	err = UpdateBookmark(bookmark, 0)
	if err == nil {
		t.Fatal("No error when updating bookmark with invalid handle")
	}
}

func BenchmarkBookmark(b *B) {
	// Create a bookmark and update it with an event
	bookmark, err := CreateBookmark()
	if err != nil {
		b.Fatal(err)
	}
	defer CloseEventHandle(uint64(bookmark))
	for i := 0; i < b.N; i++ {
		event, err := getTestEventHandle()
		if err != nil {
			b.Fatal(err)
		}
		err = UpdateBookmark(bookmark, event)
		if err != nil {
			b.Fatal(err)
		}
		_, err = RenderBookmark(bookmark)
		if err != nil {
			b.Fatal(err)
		}
		CloseEventHandle(uint64(event))
	}
}

func BenchmarkNoBookmark(b *B) {
	// Create a bookmark and update it with an event
	bookmark, err := CreateBookmark()
	if err != nil {
		b.Fatal(err)
	}
	defer CloseEventHandle(uint64(bookmark))
	for i := 0; i < b.N; i++ {
		event, err := getTestEventHandle()
		if err != nil {
			b.Fatal(err)
		}
		err = UpdateBookmark(bookmark, event)
		if err != nil {
			b.Fatal(err)
		}
		CloseEventHandle(uint64(event))
	}
}
