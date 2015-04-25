#define _WIN32_WINNT 0x0602

#include "evt.h"
#include "_cgo_export.h"

void freeLog(PVOID buf) {
    printf("Freeing %lu\n", buf);
    free(buf);
}

PVOID mallocLog(size_t len) {
    PVOID buf = malloc(len);
    printf("malloc %lu (%lu)\n", buf, len);
    return buf;
}

// Extract an array of all the attributes specified in the context
// Allocates a buffer to hold the attributes and points *pRenderedValues at it
PVOID RenderEventValues(EVT_HANDLE hContext, EVT_HANDLE hEvent) {
  DWORD dwBufferSize = 0;
  DWORD dwUsed = 0;
  DWORD dwPropertyCount = 0;
  EvtRender(hContext, hEvent, EvtRenderEventValues, dwBufferSize, NULL, &dwUsed, &dwPropertyCount);
  PVOID pRenderedValues = mallocLog(dwUsed);
  if (!pRenderedValues) {
      return NULL;
  }
  dwBufferSize = dwUsed;
  if (! EvtRender(hContext, hEvent, EvtRenderEventValues, dwBufferSize, pRenderedValues, &dwUsed, &dwPropertyCount)){
  	  free(pRenderedValues);
      return NULL;
  }
  return pRenderedValues;
}

char* GetFormattedMessage(EVT_HANDLE hEventPublisher, EVT_HANDLE hEvent, int format) {
   DWORD dwBufferSize = 0;
   DWORD dwBufferUsed = 0;
   int status;
   errno_t decodeReturn = EvtFormatMessage(hEventPublisher, hEvent, 0, 0, NULL, format, 0, NULL, &dwBufferUsed);
   if ((status = GetLastError()) != ERROR_INSUFFICIENT_BUFFER) {
       return NULL;
   }
   LPWSTR messageWide = mallocLog(dwBufferUsed);
   if (!messageWide) {
       return NULL;
   }
   dwBufferSize = dwBufferUsed;
   decodeReturn = EvtFormatMessage(hEventPublisher, hEvent, 0, 0, NULL, format, dwBufferSize, messageWide, &dwBufferUsed);
   if (!decodeReturn) {
       freeLog(messageWide);
       return NULL;
   }
   size_t lenMessage = wcstombs(NULL, messageWide, 0);
   printf("Got formatted message %lu %lu\n", dwBufferUsed, lenMessage);
   wprintf(messageWide);
   printf("\n");
   void* message = malloc(lenMessage);
   printf("Copying\n");
   if (!message) {
   	   printf("Malloc failed\n");
       freeLog(messageWide);
       return NULL;
   }
   wcstombs(message, messageWide, lenMessage);
   freeLog(messageWide);
   return message;
}

EVT_HANDLE GetEventPublisherHandle(PVOID pRenderedValues) { 
   LPCWSTR publisher = ((PEVT_VARIANT)pRenderedValues)[EvtSystemProviderName].StringVal;
   return EvtOpenPublisherMetadata(NULL, publisher, NULL, 0, 0);
}

// Create a render context that extracts all the System attributes
EVT_HANDLE CreateSystemRenderContext() {
  return EvtCreateRenderContext(0, NULL, EvtRenderContextSystem);
}

// Get the type of the rendered attribute at the given index
int GetRenderedValueType(PVOID pRenderedValues, int property) {
  return (int)((PEVT_VARIANT)pRenderedValues)[property].Type;
}

// Get the String value of the rendered attribute at the given index
// Allocates a string to put the property in
char* GetRenderedStringValue(PVOID pRenderedValues, int property) {
  wchar_t const * propVal = ((PEVT_VARIANT)pRenderedValues)[property].StringVal;
  size_t lenNarrowPropVal = wcstombs(NULL, propVal, 0) + 1;
  char* value = mallocLog(lenNarrowPropVal);
  printf("RFS %lu\n", value);
  if (!value) {
      return NULL;
  }
  wcstombs(value, propVal, lenNarrowPropVal);
  return value;
}

// Dispatch events and errors appropriately
DWORD WINAPI SubscriptionCallback(EVT_SUBSCRIBE_NOTIFY_ACTION action, PVOID pContext, EVT_HANDLE hEvent)
{    
    switch(action)
    {
        case EvtSubscribeActionError:
            EventCallbackError(hEvent, pContext);
            break;

        case EvtSubscribeActionDeliver:
            EventCallback(hEvent, pContext);
            break;

        default:
            // TODO: signal unknown error
            EventCallbackError(0, pContext);
    }

    return ERROR_SUCCESS; // The service ignores the returned status.
}

// Create a new subscription for the specified channel
// Takes the channel name, length of the channel name, 
// and a pointer to the Go WinLogWatcher.
int setupListener(char* channel, size_t channelLen, PVOID pWatcher)
{
    DWORD status = ERROR_SUCCESS;
    EVT_HANDLE hSubscription = NULL;
    size_t wideChannelLen;
    size_t maxWideChannelLen = mbstowcs(NULL, channel, 0);
    LPWSTR lChannel = mallocLog(maxWideChannelLen);
    if (!lChannel) {
    	return 1;
    }

    // Convert Go string to wide characters
    mbstowcs(lChannel, channel, maxWideChannelLen);

    // Subscribe to events beginning in the present. All future events will trigger the callback.
    hSubscription = EvtSubscribe(NULL, NULL, lChannel, NULL, NULL, pWatcher, (EVT_SUBSCRIBE_CALLBACK)SubscriptionCallback, EvtSubscribeToFutureEvents);
    freeLog(lChannel);
    if (NULL == hSubscription)
    {   
        return 2;
    }
    return 0;
}