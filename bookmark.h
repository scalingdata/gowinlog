// Set windows version to winVista - minimal required for used event log API.
// (Some of mingw installations uses too old windows headers which prevents us
// from using that API) Looks like for cgo that declaration affetcs only
// current file, so for more modern API just create a new file and define
// necessary minimal version.
#undef _WIN32_WINNT
#define _WIN32_WINNT 0x0600

#include <windows.h>
#include "winevt.h"
#include <conio.h>
#include <stdio.h>
#include <stdlib.h>

// Make a new, empty bookmark to update
ULONGLONG CreateBookmark();
// Load an existing bookmark from a Go XML string
ULONGLONG CreateBookmarkFromXML(char* xmlString);
// Update a bookmark to the given event handle
int UpdateBookmark(ULONGLONG hBookmark, ULONGLONG hEvent);
// Render the XML string for a bookmark
char* RenderBookmark(ULONGLONG hBookmark);
