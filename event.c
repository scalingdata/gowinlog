// +build windows

// This file contains convenience stub functions that call Win32 API
// functions and return values suitable for handling in Go.
// Note that some functions such as EvtRender() and EvtFormatMessage() support a model
// where they are called twice - once to determine the necessary buffer size,
// and once to copy values into the supplied buffer.

// Set windows version to winVista - minimal required for used event log API.
// (Some of mingw installations uses too old windows headers which prevents us
// from using that API) Looks like for cgo that declaration affetcs only
// current file, so for more modern API just create a new file and define
// necessary minimal version.
#define _WIN32_WINNT 0x0600

#include "event.h"
#include "_cgo_export.h"

int CloseEvtHandle(ULONGLONG hEvent) {
	EvtClose((EVT_HANDLE)hEvent);
}

int CancelEvtHandle(ULONGLONG hEvent) {
	EvtCancel((EVT_HANDLE)hEvent);
}

PVOID RenderEventValues(ULONGLONG hContext, ULONGLONG hEvent) {
	DWORD dwBufferSize = 0;
	DWORD dwUsed = 0;
	DWORD dwPropertyCount = 0;
	EvtRender((EVT_HANDLE)hContext, (EVT_HANDLE)hEvent, EvtRenderEventValues, dwBufferSize, NULL, &dwUsed, &dwPropertyCount);
	PVOID pRenderedValues = malloc(dwUsed);
	if (!pRenderedValues) {
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	dwBufferSize = dwUsed;
	if (! EvtRender((EVT_HANDLE)hContext, (EVT_HANDLE)hEvent, EvtRenderEventValues, dwBufferSize, pRenderedValues, &dwUsed, &dwPropertyCount)){
		free(pRenderedValues);
		return NULL;
	}
	return pRenderedValues;
}

// Render the event as XML. The returned buffer must be freed after use.
char* RenderEventXML(ULONGLONG hEvent) {
	DWORD dwBufferSize = 0;
	DWORD dwUsed = 0;
	DWORD dwPropertyCount = 0;
	EvtRender(NULL, (EVT_HANDLE)hEvent, EvtRenderEventXml, dwBufferSize, NULL, &dwUsed, &dwPropertyCount);

	// Allocate a buffer to hold the utf-16 encoded xml string. Although the xml
	// string is utf-16, the dwUsed value is in bytes, not characters
	// See https://msdn.microsoft.com/en-us/library/windows/desktop/aa385471(v=vs.85).aspx
	LPWSTR xmlWide = malloc(dwUsed);
	if (!xmlWide) {
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	dwBufferSize = dwUsed;
	if (! EvtRender(NULL, (EVT_HANDLE)hEvent, EvtRenderEventXml, dwBufferSize, xmlWide, &dwUsed, 0)){
		free(xmlWide);
		return NULL;
	}

	// Convert the xml string to multibyte
	size_t lenXml = wcstombs(NULL, xmlWide, 0) + 1;
	void* xml = malloc(lenXml);
	if (!xml) {
		free(xmlWide);
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	wcstombs(xml, xmlWide, lenXml);
	free(xmlWide);
	return xml;
}

char* GetLastErrorString() {
	DWORD dwErr = GetLastError();
	LPSTR lpszMsgBuf;
	FormatMessage(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_ALLOCATE_BUFFER | FORMAT_MESSAGE_IGNORE_INSERTS, 0, dwErr, 0, (LPSTR)&lpszMsgBuf, 0, NULL);
	return (char *)lpszMsgBuf;
}

char* GetFormattedMessage(ULONGLONG hEventPublisher, ULONGLONG hEvent, int format) {
	DWORD dwBufferSize = 0;
	DWORD dwBufferUsed = 0;
	int status;
	errno_t decodeReturn = EvtFormatMessage((EVT_HANDLE)hEventPublisher, (EVT_HANDLE)hEvent, 0, 0, NULL, format, 0, NULL, &dwBufferUsed);
	if ((status = GetLastError()) != ERROR_INSUFFICIENT_BUFFER) {
		return NULL;
	}
	dwBufferSize = dwBufferUsed + 1;
	LPWSTR messageWide = malloc((dwBufferSize) * sizeof(wchar_t));
	if (!messageWide) {
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	decodeReturn = EvtFormatMessage((EVT_HANDLE)hEventPublisher, (EVT_HANDLE)hEvent, 0, 0, NULL, format, dwBufferSize, messageWide, &dwBufferUsed);
	if (!decodeReturn) {
		free(messageWide);
		return NULL;
	}
	size_t lenMessage = wcstombs(NULL, messageWide, 0) + 1;
	void* message = malloc(lenMessage);
	if (!message) {
		free(messageWide);
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	wcstombs(message, messageWide, lenMessage);
	free(messageWide);
	return message;
}

ULONGLONG GetEventPublisherHandle(PVOID pRenderedValues) { 
	LPCWSTR publisher = ((PEVT_VARIANT)pRenderedValues)[EvtSystemProviderName].StringVal;
	return (ULONGLONG)EvtOpenPublisherMetadata(NULL, publisher, NULL, 0, 0);
}

ULONGLONG CreateSystemRenderContext() {
	return (ULONGLONG)EvtCreateRenderContext(0, NULL, EvtRenderContextSystem);
}

int GetRenderedValueType(PVOID pRenderedValues, int property) {
	return (int)((PEVT_VARIANT)pRenderedValues)[property].Type;
}

char* GetRenderedStringValue(PVOID pRenderedValues, int property) {
	wchar_t const * propVal = ((PEVT_VARIANT)pRenderedValues)[property].StringVal;
	size_t lenNarrowPropVal = wcstombs(NULL, propVal, 0) + 1;
	char* value = malloc(lenNarrowPropVal);
	if (!value) {
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	wcstombs(value, propVal, lenNarrowPropVal);
	return value;
}

ULONGLONG GetRenderedByteValue(PVOID pRenderedValues, int property) {
	return ((PEVT_VARIANT)pRenderedValues)[property].ByteVal; 
}

ULONGLONG GetRenderedUInt16Value(PVOID pRenderedValues, int property) {
	return ((PEVT_VARIANT)pRenderedValues)[property].UInt16Val; 
}

ULONGLONG GetRenderedUInt32Value(PVOID pRenderedValues, int property) {
	return ((PEVT_VARIANT)pRenderedValues)[property].UInt32Val; 
}

ULONGLONG GetRenderedUInt64Value(PVOID pRenderedValues, int property) {
	return ((PEVT_VARIANT)pRenderedValues)[property].UInt64Val; 
}

LONGLONG GetRenderedSByteValue(PVOID pRenderedValues, int property) {
	return ((PEVT_VARIANT)pRenderedValues)[property].SByteVal; 
}

LONGLONG GetRenderedInt16Value(PVOID pRenderedValues, int property) {
	return ((PEVT_VARIANT)pRenderedValues)[property].Int16Val; 
}

LONGLONG GetRenderedInt32Value(PVOID pRenderedValues, int property) {
	return ((PEVT_VARIANT)pRenderedValues)[property].Int32Val; 
}

LONGLONG GetRenderedInt64Value(PVOID pRenderedValues, int property) {
	return ((PEVT_VARIANT)pRenderedValues)[property].Int64Val; 
}

//FILETIME to unix epoch: https://support.microsoft.com/en-us/kb/167296
ULONGLONG GetRenderedFileTimeValue(PVOID pRenderedValues, int property) {
    FILETIME* ft = (FILETIME*) &(((PEVT_VARIANT)pRenderedValues)[property].FileTimeVal);
    ULONGLONG time = ft->dwHighDateTime;
    time = (time << 32) | ft->dwLowDateTime;
    return time;
}

// Dispatch events and errors appropriately
DWORD WINAPI SubscriptionCallback(EVT_SUBSCRIBE_NOTIFY_ACTION action, PVOID pContext, EVT_HANDLE hEvent)
{    
	switch(action)
	{
		case EvtSubscribeActionError:
		// In this case, hEvent is an error code, not a handle to an event
		// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385596(v=vs.85).aspx
		eventCallbackError((ULONGLONG)hEvent, pContext);
		break;

		case EvtSubscribeActionDeliver:
		eventCallback((ULONGLONG)hEvent, pContext);
		break;

		default:
            // TODO: signal unknown error
		eventCallbackError(0, pContext);
	}

    return ERROR_SUCCESS; // The service ignores the returned status.
}

ULONGLONG SetupListener(char* channel, char* query, PVOID pWatcher, EVT_HANDLE hBookmark, EVT_SUBSCRIBE_FLAGS flags)
{
	DWORD status = ERROR_SUCCESS;
	EVT_HANDLE hSubscription = NULL;
	size_t wideChannelLen;
	size_t maxWideChannelLen = mbstowcs(NULL, channel, 0) + 1;
	LPWSTR lChannel = malloc(maxWideChannelLen * sizeof(wchar_t));
	if (!lChannel) {
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return 0;
	}

	size_t maxWideQueryLen = mbstowcs(NULL, query, 0) + 1;
	LPWSTR lQuery = malloc(maxWideQueryLen * sizeof(wchar_t));
	if (!lQuery) {
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return 0;
	}

  // Convert Go string to wide characters
	mbstowcs(lChannel, channel, maxWideChannelLen);
	mbstowcs(lQuery, query, maxWideQueryLen);

  // Subscribe to events beginning in the present. All future events will trigger the callback.
	hSubscription = EvtSubscribe(NULL, NULL, lChannel, lQuery, hBookmark, pWatcher, (EVT_SUBSCRIBE_CALLBACK)SubscriptionCallback, flags);
	free(lChannel);
	return (ULONGLONG)hSubscription;
}

ULONGLONG CreateListener(char* channel, char* query, int startPos, PVOID pWatcher) {
	return SetupListener(channel, query, pWatcher, NULL, startPos);
}

ULONGLONG CreateListenerFromBookmark(char* channel, char* query, PVOID pWatcher, ULONGLONG hBookmark) {
	return SetupListener(channel, query, pWatcher, (EVT_HANDLE)hBookmark, EvtSubscribeStartAfterBookmark);
}

int EnableChannel(EVT_HANDLE hChannel, int status) {
    EVT_VARIANT ChannelProperty = {0};

    // Set status `Enable`
    ChannelProperty.Type = EvtVarTypeBoolean;
    ChannelProperty.BooleanVal = status == 0 ? FALSE : TRUE;
    if  (!EvtSetChannelConfigProperty((EVT_HANDLE)hChannel, EvtChannelConfigEnabled, 0, &ChannelProperty)) {
       return 1;
    }

    return 0;
}

int SetBufferSizeB(EVT_HANDLE hChannel, int bufferSizeB) {
    EVT_VARIANT ChannelProperty = {0};
    
    // Set buffer size.
    ChannelProperty.Type = EvtVarTypeUInt64;
    ChannelProperty.UInt64Val = bufferSizeB;
    if  (!EvtSetChannelConfigProperty(hChannel, EvtChannelLoggingConfigMaxSize, 0, &ChannelProperty)) {
        return 1;
    }

    return 0;
}

ULONGLONG GetTestEventHandle() {
	DWORD status = ERROR_SUCCESS;
	EVT_HANDLE record = 0;
	DWORD recordsReturned;
	EVT_HANDLE result = EvtQuery(NULL, L"Application", L"*", EvtQueryChannelPath);
	if (result == 0) {
		return 0;
	}
	if (!EvtNext(result, 1, &record, 500, 0, &recordsReturned)) {
		EvtClose(result);
		return 0;
	}
	EvtClose(result);
	return (ULONGLONG)record;
}
