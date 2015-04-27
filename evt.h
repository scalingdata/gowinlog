#include <windows.h>
#include "winevt.h"
#include <conio.h>
#include <stdio.h>
#include <stdlib.h>

/* Handles really should be EVT_HANDLE, but Go will sometimes
   try to check if pointers are valid, and handles aren't necessarily
   pointers (although they have type PVOID). So we pass handles to Go as
   64-bit unsigned ints. */

// Create a new listener on the given channel. Events will be passed
// to the callback of *pWatcher.
int setupListener(char* channel, size_t channelLen, PVOID pWatcher);

// Render the fields for the given context. Allocates an array
// of values based on the context, these can be accessed using
// GetRendered<type>Value. Buffer must be freed by the caller.
PVOID RenderEventValues(ULONGLONG hContext, ULONGLONG hEvent);

// Get the type of the variable at the given index in the array.
// Possible types are EvtVarType*
int GetRenderedValueType(PVOID pRenderedValues, int property);

// Get the value of the variable at the given index. You must know
// the type or this will go badly.
LONGLONG GetRenderedSByteValue(PVOID pRenderedValues, int property);
LONGLONG GetRenderedInt16Value(PVOID pRenderedValues, int property);
LONGLONG GetRenderedInt32Value(PVOID pRenderedValues, int property);
LONGLONG GetRenderedInt64Value(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedByteValue(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedUInt16Value(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedUInt32Value(PVOID pRenderedValues, int property);
ULONGLONG GetRenderedUInt64Value(PVOID pRenderedValues, int property);
// Returns a pointer to a string that must be freed by the caller
char* GetRenderedStringValue(PVOID pRenderedValues, int property);
// Returns a unix epoch timestamp in milliseconds, not a FileTime
ULONGLONG GetRenderedFileTimeValue(PVOID pRenderedValues, int property);

// Format the event into a string using details from the event publisher. 
// Valid formats are EvtFormatMessage*
char* GetFormattedMessage(ULONGLONG hEventPublisher, ULONGLONG hEvent, int format);

// Get the handle for the publisher, this must be closed by the caller.
// Needed to format messages since schema is publisher-specific.
ULONGLONG GetEventPublisherHandle(PVOID pRenderedValues);

// Cast the ULONGLONG back to a pointer and close it
int CloseEvtHandle(ULONGLONG hEvent);

// Create a context for RenderEventValues that decodes standard system properties.
// Properties in the resulting array can be accessed using the indices from 
// EvtSystem*
ULONGLONG CreateSystemRenderContext();