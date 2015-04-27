#define _WIN32_WINNT 0x0602

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