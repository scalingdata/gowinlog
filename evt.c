// +build windows

#define _WIN32_WINNT 0x0602

#include "evt.h"
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
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	return pRenderedValues;
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
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
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
	return (time / 10000000) - 11644473600;
}

// Dispatch events and errors appropriately
DWORD WINAPI SubscriptionCallback(EVT_SUBSCRIBE_NOTIFY_ACTION action, PVOID pContext, EVT_HANDLE hEvent)
{    
	switch(action)
	{
		case EvtSubscribeActionError:
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

ULONGLONG SetupListener(char* channel, PVOID pWatcher, EVT_HANDLE hBookmark, EVT_SUBSCRIBE_FLAGS flags)
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

    // Convert Go string to wide characters
	mbstowcs(lChannel, channel, maxWideChannelLen);

    // Subscribe to events beginning in the present. All future events will trigger the callback.
	hSubscription = EvtSubscribe(NULL, NULL, lChannel, NULL, hBookmark, pWatcher, (EVT_SUBSCRIBE_CALLBACK)SubscriptionCallback, flags);
	free(lChannel);
	return (ULONGLONG)hSubscription;
}

ULONGLONG CreateListener(char* channel, int startPos, PVOID pWatcher) {
	return SetupListener(channel, pWatcher, NULL, startPos);
}

ULONGLONG CreateListenerFromBookmark(char* channel, PVOID pWatcher, ULONGLONG hBookmark) {
	return SetupListener(channel, pWatcher, (EVT_HANDLE)hBookmark, EvtSubscribeStartAfterBookmark);
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
