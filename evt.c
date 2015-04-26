#define _WIN32_WINNT 0x0602

#include "evt.h"
#include "_cgo_export.h"

// Extract an array of all the attributes specified in the context
// Allocates a buffer to hold the attributes and points *pRenderedValues at it
PVOID RenderEventValues(EVT_HANDLE hContext, EVT_HANDLE hEvent) {
  DWORD dwBufferSize = 0;
  DWORD dwUsed = 0;
  DWORD dwPropertyCount = 0;
  EvtRender(hContext, hEvent, EvtRenderEventValues, dwBufferSize, NULL, &dwUsed, &dwPropertyCount);
  PVOID pRenderedValues = malloc(dwUsed);
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
   dwBufferSize = dwBufferUsed + 1;
   LPWSTR messageWide = malloc((dwBufferSize) * sizeof(wchar_t));
   if (!messageWide) {
       return NULL;
   }
   decodeReturn = EvtFormatMessage(hEventPublisher, hEvent, 0, 0, NULL, format, dwBufferSize, messageWide, &dwBufferUsed);
   if (!decodeReturn) {
       free(messageWide);
       return NULL;
   }
   size_t lenMessage = wcstombs(NULL, messageWide, 0) + 1;
   void* message = malloc(lenMessage);
   if (!message) {
       free(messageWide);
       return NULL;
   }
   wcstombs(message, messageWide, lenMessage);
   free(messageWide);
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
  char* value = malloc(lenNarrowPropVal);
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
    LPWSTR lChannel = malloc(maxWideChannelLen * sizeof(wchar_t));
    if (!lChannel) {
    	return 1;
    }

    // Convert Go string to wide characters
    mbstowcs(lChannel, channel, maxWideChannelLen);

    // Subscribe to events beginning in the present. All future events will trigger the callback.
    hSubscription = EvtSubscribe(NULL, NULL, lChannel, NULL, NULL, pWatcher, (EVT_SUBSCRIBE_CALLBACK)SubscriptionCallback, EvtSubscribeToFutureEvents);
    free(lChannel);
    if (NULL == hSubscription)
    {   
        return 2;
    }
    return 0;
}