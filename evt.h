#include <windows.h>
#include "winevt.h"
#include <conio.h>
#include <stdio.h>
#include <stdlib.h>
int setupListener(char* channel, size_t channelLen, PVOID pWatcher);
PVOID RenderEventValues(PVOID hContext, PVOID hEvent);
int GetRenderedValueType(PVOID pRenderedValues, int property);
char* GetRenderedStringValue(PVOID pRenderedValues, int property);
long GetRenderedSByteValue(PVOID pRenderedValues, int property);
long GetRenderedInt16Value(PVOID pRenderedValues, int property);
long GetRenderedInt32Value(PVOID pRenderedValues, int property);
long GetRenderedInt64Value(PVOID pRenderedValues, int property);
unsigned long GetRenderedByteValue(PVOID pRenderedValues, int property);
unsigned long GetRenderedUInt16Value(PVOID pRenderedValues, int property);
unsigned long GetRenderedUInt32Value(PVOID pRenderedValues, int property);
unsigned long GetRenderedUInt64Value(PVOID pRenderedValues, int property);
unsigned long GetRenderedFileTimeValue(PVOID pRenderedValues, int property);
char* GetFormattedMessage(PVOID hEventPublisher, PVOID hEvent, int format);
PVOID GetEventPublisherHandle(PVOID pRenderedValues);
PVOID CreateSystemRenderContext();