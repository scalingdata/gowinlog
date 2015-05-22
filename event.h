#include <windows.h>
#include "winevt.h"
#include <conio.h>
#include <stdio.h>
#include <stdlib.h>

#ifndef GOWINLOG_H
#define GOWINLOG_H

typedef struct renderedFields {
  PVOID fields;
  unsigned long nFields;
} RenderedFields;

/* Handles really should be EVT_HANDLE, but Go will sometimes
   try to check if pointers are valid, and handles aren't necessarily
   pointers (although they have type PVOID). So we pass handles to Go as
   64-bit unsigned ints. */

// Create a new listener on the given channel. Events will be passed
// to the callback of *pWatcher. Starts at the current position in the log
ULONGLONG CreateListener(char* channel, int startpos, PVOID pWatcher);

// Create a new listener on the given channel. Events will be passed
// to the callback of *pWatcher. Starts at the given bookmark handle.
// Note: This doesn't set the strict flag - if the log was truncated between
// the bookmark and now, it'll continue silently from the earliest event.
ULONGLONG CreateListenerFromBookmark(char* channel, PVOID pWatcher, ULONGLONG hBookmark);

// Get the string for the last error code
char* GetLastErrorString();

// Render the fields for the given context. Allocates an array
// of values based on the context, these can be accessed using
// GetRendered<type>Value. Buffer must be freed by the caller.
RenderedFields* RenderEventValues(ULONGLONG hContext, ULONGLONG hEvent);

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

// Cancel pending operations on a handle
int CancelEvtHandle(ULONGLONG hEvent);

// Create a context for RenderEventValues that decodes standard system properties.
// Properties in the resulting array can be accessed using the indices from 
// EvtSystem*
ULONGLONG CreateSystemRenderContext();

// Create a context for RenderEventValues that decodes event specific data.
// Properties in the resulting array can be iterated over.
ULONGLONG CreateUserRenderContext();

// For testing, get a handle on the first event in the log
ULONGLONG GetTestEventHandle();

#endif /* GOWINLOG_H */