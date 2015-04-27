#include "bookmark.h"

ULONGLONG CreateBookmark() {
	return (ULONGLONG)EvtCreateBookmark(NULL);
}

ULONGLONG CreateBookmarkFromXML(char* xmlString) {
	size_t xmlWideLen= mbstowcs(NULL, xmlString, 0) + 1;
	LPWSTR lxmlString = malloc(xmlWideLen * sizeof(wchar_t));
	if (!lxmlString) {
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return 0;
	}

    // Convert Go string to wide characters
	mbstowcs(lxmlString, xmlString, xmlWideLen);
	return (ULONGLONG)EvtCreateBookmark(lxmlString);
}

int UpdateBookmark(ULONGLONG hBookmark, ULONGLONG hEvent) {
	return EvtUpdateBookmark((EVT_HANDLE)hBookmark, (EVT_HANDLE)hEvent);
}

char* RenderBookmark(ULONGLONG hBookmark) {
	DWORD dwUsed;
	DWORD dwProps;
	DWORD dwSize = 0;
    EvtRender(NULL, (EVT_HANDLE)hBookmark, EvtRenderBookmark, dwSize, NULL, &dwUsed, &dwProps);
    if (GetLastError() != ERROR_INSUFFICIENT_BUFFER){
    	return NULL;
    }
    dwSize = dwUsed + 1;
	LPWSTR xmlWide = malloc((dwSize) * sizeof(wchar_t));
	if (!xmlWide) {
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	int renderResult = EvtRender(NULL, (EVT_HANDLE)hBookmark, EvtRenderBookmark, dwSize, xmlWide, &dwUsed, &dwProps);
    if (!renderResult) {
    	free(xmlWide);
    	return 0;
    }
    size_t xmlNarrowLen = wcstombs(NULL, xmlWide, 0) + 1;
	void* xmlNarrow = malloc(xmlNarrowLen);
	if (!xmlNarrow) {
		free(xmlWide);
		SetLastError(ERROR_NOT_ENOUGH_MEMORY);
		return NULL;
	}
	wcstombs(xmlNarrow, xmlWide, xmlNarrowLen);
	free(xmlWide);
	return xmlNarrow;
}