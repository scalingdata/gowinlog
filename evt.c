#define _WIN32_WINNT 0x0602

#include "evt.h"
#include "_cgo_export.h"

// Extract an array of all the attributes specified in the context
// Allocates a buffer to hold the attributes and puts it in *pRenderedValues
int RenderEventValues(EVT_HANDLE hContext, EVT_HANDLE hEvent, PVOID* pRenderedValues, int* pdwUsed, int* pdwPropertyCount) {
  DWORD dwBufferSize = 0;
  EvtRender(hContext, hEvent, EvtRenderEventValues, dwBufferSize, *pRenderedValues, (PDWORD)pdwUsed, (PDWORD)pdwPropertyCount);
  *(char**)pRenderedValues = malloc(*pdwUsed);
  if (*(char**)pRenderedValues == 0) {
    return -1;
  }
  dwBufferSize = *pdwUsed;
  if (! EvtRender(hContext, hEvent, EvtRenderEventValues, dwBufferSize, *pRenderedValues, (PDWORD)pdwUsed, (PDWORD)pdwPropertyCount)){
    return GetLastError();
  }
  return 0;
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
char* GetRenderedStringValue(PVOID pRenderedValues, int property) {
  wchar_t const * propVal = ((PEVT_VARIANT)pRenderedValues)[property].StringVal;
  size_t lenNarrowPropVal = wcslen(propVal);
  char* narrowPropVal = malloc(lenNarrowPropVal);
  wcstombs(narrowPropVal, propVal, lenNarrowPropVal);
  return narrowPropVal;
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
    size_t maxWideChannelLen = channelLen * 2;
    LPWSTR lChannel = malloc(maxWideChannelLen);

    // Convert Go string to wide characters
    mbstowcs_s(&wideChannelLen, lChannel, maxWideChannelLen, channel, maxWideChannelLen);

    // Subscribe to events beginning in the present. All future events will trigger the callback.
    hSubscription = EvtSubscribe(NULL, NULL, lChannel, NULL, NULL, pWatcher, (EVT_SUBSCRIBE_CALLBACK)SubscriptionCallback, EvtSubscribeToFutureEvents);
    free(lChannel);
    if (NULL == hSubscription)
    {   
        return GetLastError();
    }
    return 0;
}