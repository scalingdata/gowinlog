#include <windows.h>
#include "winevt.h"
#include <conio.h>
#include <stdio.h>
#include <stdlib.h>
int setupListener(char* channel, size_t channelLen, PVOID pWatcher);
int RenderEventValues(PVOID hContext, PVOID hEvent, PVOID* pRenderedValues, int* pdwUsed, int* pdwPropertyCount);
int GetRenderedValueType(PVOID pRenderedValues, int property);
char* GetRenderedStringValue(PVOID pRenderedValues, int property);
PVOID CreateSystemRenderContext();
