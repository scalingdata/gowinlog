#include <windows.h>
#include "winevt.h"
#include <conio.h>
#include <stdio.h>
#include <stdlib.h>
int setupListener(char* channel, size_t channelLen, PVOID pWatcher);
PVOID RenderEventValues(ULONGLONG hContext, ULONGLONG hEvent);
int GetRenderedValueType(PVOID pRenderedValues, int property);
char* GetRenderedStringValue(PVOID pRenderedValues, int property);
LONGLONG GetRenderedSByteValue(PVOID pRenderedValues, int property);
LONGLONG GetRenderedInt16Value(PVOID pRenderedValues, int property);
LONGLONG GetRenderedInt32Value(PVOID pRenderedValues, int property);
LONGLONG GetRenderedInt64Value(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedByteValue(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedUInt16Value(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedUInt32Value(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedUInt64Value(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedFileTimeValue(PVOID pRenderedValues, int property);
char* GetFormattedMessage(ULONGLONG hEventPublisher, ULONGLONG hEvent, int format);
ULONGLONG GetEventPublisherHandle(PVOID pRenderedValues);
ULONGLONG CreateSystemRenderContext();