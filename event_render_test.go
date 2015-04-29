package winlog

import (
  . "testing"
  "unsafe"
  "encoding/xml"
)

type ProviderXml struct {
	ProviderName string `xml:"Name,attr"`
	EventSourceName string `xml:"EventSourceName,attr"`
}

type EventIdXml struct {
	EventID      uint64 `xml:",chardata"`
	Qualifiers   uint64 `xml:"Qualifiers,attr"`
}

type TimeCreatedXml struct {
	SystemTime string `xml:"SystemTime,attr"`
}

type ExecutionXml struct {
	ProcessId uint64 `xml:"ProcessID,attr`
}

type SystemXml struct {
	Provider     ProviderXml
	EventID      EventIdXml
	
	Level        uint64 `xml:"Level"`
	Task         uint64 `xml:"Task"`
	Opcode       uint64 `xml:"Opcode"`
	TimeCreated      TimeCreatedXml
	RecordId     uint64 `xml:"EventRecordID"`
	Execution    ExecutionXml
	Channel      string `xml:"Channel"`
	ComputerName string `xml:"Computer"`
	Version      uint64 `xml:"Version"`
}

type RenderingInfoXml struct {
	Msg          string `xml:"Message"`
	LevelText    string `xml:"Level"`
	TaskText     string `xml:"Task"`
	OpcodeText   string `xml:"Opcode"`
	Keywords     []string `xml:"Keywords"`
	ChannelText  string `xml:"Channel"`
	ProviderText string `xml:"Provider"`
}

type WinLogEventXml struct {
	System SystemXml
	RenderingInfo RenderingInfoXml
}	

func assertEqual(a, b interface{}, t *T) {
	if a != b { t.Fatalf("%v != %v", a, b) }
}

func TestXmlRenderMatchesOurs(t *T) {
	testEvent, err := getTestEventHandle()
	if err != nil { t.Fatal(err)}
	defer CloseEventHandle(uint64(testEvent))
	renderContext, err := GetSystemRenderContext()
    if err != nil { t.Fatal(err) }
    defer CloseEventHandle(uint64(renderContext))
    renderedFields, err := RenderEventValues(renderContext, testEvent)
    if err != nil { t.Fatal(err) }
    defer Free(unsafe.Pointer(renderedFields))
    publisherHandle, err := GetEventPublisherHandle(renderedFields)
    if err != nil { t.Fatal(err) }
    defer CloseEventHandle(uint64(publisherHandle))
    xmlString, err := FormatMessage(publisherHandle, testEvent, EvtFormatMessageXml)
    if err != nil { t.Fatal(err) }
    eventXml := WinLogEventXml{}
    if err = xml.Unmarshal([]byte(xmlString), &eventXml); err != nil {
      t.Fatal(err)
    }
    
    logWatcher, err := NewWinLogWatcher()
    defer logWatcher.Shutdown()
    event, err := logWatcher.convertEvent(testEvent)
    if err != nil {t.Fatal(err)}

    assertEqual(event.ProviderName, eventXml.System.Provider.ProviderName, t)
    assertEqual(event.EventId, eventXml.System.EventID.EventID, t)
    assertEqual(event.Qualifiers, eventXml.System.EventID.Qualifiers, t)
    assertEqual(event.Level, eventXml.System.Level, t)
	assertEqual(event.Task, eventXml.System.Task, t)
	assertEqual(event.Opcode, eventXml.System.Opcode, t)
	assertEqual(event.RecordId, eventXml.System.RecordId, t)
	assertEqual(event.ProcessId, eventXml.System.Execution.ProcessId, t)
	assertEqual(event.Channel, eventXml.System.Channel, t)
	assertEqual(event.ComputerName, eventXml.System.ComputerName, t)
	assertEqual(event.Msg, eventXml.RenderingInfo.Msg, t)
	assertEqual(event.LevelText, eventXml.RenderingInfo.LevelText, t)
	assertEqual(event.TaskText, eventXml.RenderingInfo.TaskText, t)
	assertEqual(event.OpcodeText, eventXml.RenderingInfo.OpcodeText, t)
	assertEqual(event.ChannelText, eventXml.RenderingInfo.ChannelText, t)
	assertEqual(event.ProviderText, eventXml.RenderingInfo.ProviderText, t)
	assertEqual(event.Created.Format("2006-01-02T15:04:05.000000000Z"), eventXml.System.TimeCreated.SystemTime, t)
}

func BenchmarkXmlDecode(b *B) {
	testEvent, err := getTestEventHandle()
	if err != nil { b.Fatal(err)}
	defer CloseEventHandle(uint64(testEvent))
	renderContext, err := GetSystemRenderContext()
    if err != nil { b.Fatal(err) }
    defer CloseEventHandle(uint64(renderContext))
    for i := 0; i < b.N; i++ {
    	renderedFields, err := RenderEventValues(renderContext, testEvent)
   		if err != nil { b.Fatal(err) }
		publisherHandle, err := GetEventPublisherHandle(renderedFields)
	    if err != nil { b.Fatal(err) }
	    xmlString, err := FormatMessage(publisherHandle, testEvent, EvtFormatMessageXml)
	    if err != nil { b.Fatal(err) }
	    eventXml := WinLogEventXml{}
	    if err = xml.Unmarshal([]byte(xmlString), &eventXml); err != nil {
	      b.Fatal(err)
	    }
	    Free(unsafe.Pointer(renderedFields))
	    CloseEventHandle(uint64(publisherHandle))
	}
}

func BenchmarkAPIDecode(b *B) {
	testEvent, err := getTestEventHandle()
	if err != nil { b.Fatal(err)}
	defer CloseEventHandle(uint64(testEvent))
	renderContext, err := GetSystemRenderContext()
    if err != nil { b.Fatal(err) }
    defer CloseEventHandle(uint64(renderContext))
    logWatcher, err := NewWinLogWatcher()
	for i := 0; i < b.N; i++ {
		_, err := logWatcher.convertEvent(testEvent)
    	if err != nil {b.Fatal(err)}
	}
}