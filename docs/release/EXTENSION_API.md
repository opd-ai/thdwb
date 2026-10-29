# thdwb WebExtensions API Reference

thdwb implements a Firefox-compatible subset of the WebExtensions API. This
document describes the supported APIs, their behavior, and any deviations from
the Firefox specification.

## Supported APIs

### chrome.runtime

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.runtime.id` | The extension's ID | ✅ |
| `chrome.runtime.getManifest()` | Returns the extension manifest | ✅ |
| `chrome.runtime.getURL(path)` | Converts a relative path to an absolute URL | ✅ |
| `chrome.runtime.getPlatformInfo()` | Returns platform information | ✅ |
| `chrome.runtime.sendMessage(message, options, callback)` | Sends a message to the extension background script | ✅ |
| `chrome.runtime.onMessage` | Fired when a message is sent from the extension | ✅ |
| `chrome.runtime.onInstalled` | Fired when the extension is installed | ✅ |
| `chrome.runtime.onStartup` | Fired when the browser starts | ✅ |

### chrome.tabs

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.tabs.query(queryInfo, callback)` | Queries tabs matching the criteria | ✅ |
| `chrome.tabs.get(tabId, callback)` | Gets details about a tab | ✅ |
| `chrome.tabs.update(tabId, updateProperties, callback)` | Modifies a tab | ✅ |
| `chrome.tabs.create(createProperties, callback)` | Creates a new tab | ✅ |
| `chrome.tabs.remove(tabId, callback)` | Closes a tab | ✅ |
| `chrome.tabs.onCreated` | Fired when a tab is created | ✅ |
| `chrome.tabs.onRemoved` | Fired when a tab is removed | ✅ |
| `chrome.tabs.onUpdated` | Fired when a tab is updated | ✅ |
| `chrome.tabs.onActivated` | Fired when the active tab changes | ✅ |

### chrome.windows

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.windows.getCurrent(getInfo, callback)` | Gets the current window | ✅ |
| `chrome.windows.getAll(getInfo, callback)` | Gets all windows | ✅ |
| `chrome.windows.create(createData, callback)` | Creates a new window | ✅ |
| `chrome.windows.remove(windowId, callback)` | Closes a window | ✅ |

### chrome.storage

| Method | Description | Status |
|--------|-------------|--------|
| `chrome.storage.local.get(keys, callback)` | Gets items from local storage | ✅ |
| `chrome.storage.local.set(items, callback)` | Sets items in local storage | ✅ |
| `chrome.storage.local.remove(keys, callback)` | Removes items from local storage | ✅ |
| `chrome.storage.local.clear(callback)` | Clears all local storage | ✅ |
| `chrome.storage.sync.get(keys, callback)` | Gets items from sync storage | ✅ |
| `chrome.storage.sync.set(items, callback)` | Sets items in sync storage | ✅ |
| `chrome.storage.sync.remove(keys, callback)` | Removes items from sync storage | ✅ |
| `chrome.storage.sync.clear(callback)` | Clears all sync storage | ✅ |
| `chrome.storage.managed.get(keys, callback)` | Gets managed (read-only) storage | ✅ |

### chrome.contextualIdentities

| Method | Description | Status |
|--------|-------------|--------|
| `chrome.contextualIdentities.query(queryInfo, callback)` | Queries containers | ✅ |
| `chrome.contextualIdentities.get(containerId, callback)` | Gets a container | ✅ |
| `chrome.contextualIdentities.create(details, callback)` | Creates a container | ✅ |
| `chrome.contextualIdentities.update(containerId, details, callback)` | Updates a container | ✅ |
| `chrome.contextualIdentities.remove(containerId, callback)` | Removes a container | ✅ |
| `chrome.contextualIdentities.onCreated` | Fired when a container is created | ✅ |
| `chrome.contextualIdentities.onUpdated` | Fired when a container is updated | ✅ |
| `chrome.contextualIdentities.onRemoved` | Fired when a container is removed | ✅ |

### chrome.scripting

| Method | Description | Status |
|--------|-------------|--------|
| `chrome.scripting.executeScript(injection, target, callback)` | Executes JavaScript in a page | ✅ |
| `chrome.scripting.insertCSS(injection, target, callback)` | Injects CSS into a page | ✅ |
| `chrome.scripting.removeCSS(injection, target, callback)` | Removes injected CSS | ✅ |
| `chrome.scripting.registerContentScripts(scripts, callback)` | Registers content scripts dynamically | ✅ |
| `chrome.scripting.getRegisteredContentScripts(callback)` | Gets registered content scripts | ✅ |
| `chrome.scripting.unregisterContentScripts(ids, callback)` | Unregisters content scripts | ✅ |

### chrome.contextMenus

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.contextMenus.create(createProperties, callback)` | Creates a context menu item | ✅ |
| `chrome.contextMenus.update(id, updateProperties, callback)` | Updates a context menu item | ✅ |
| `chrome.contextMenus.remove(id, callback)` | Removes a context menu item | ✅ |
| `chrome.contextMenus.removeAll(callback)` | Removes all context menu items | ✅ |
| `chrome.contextMenus.onClicked` | Fired when a context menu item is clicked | ✅ |

### chrome.permissions

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.permissions.request(permissions, callback)` | Requests permissions | ✅ |
| `chrome.permissions.contains(permissions, callback)` | Checks if permissions are granted | ✅ |
| `chrome.permissions.getAll(callback)` | Gets all granted permissions | ✅ |
| `chrome.permissions.onAdded` | Fired when permissions are added | ✅ |
| `chrome.permissions.onRemoved` | Fired when permissions are removed | ✅ |

### chrome.alarms

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.alarms.create(name, alarmInfo, callback)` | Creates an alarm | ✅ |
| `chrome.alarms.get(name, callback)` | Gets an alarm | ✅ |
| `chrome.alarms.getAll(callback)` | Gets all alarms | ✅ |
| `chrome.alarms.clear(name, callback)` | Clears an alarm | ✅ |
| `chrome.alarms.clearAll(callback)` | Clears all alarms | ✅ |
| `chrome.alarms.onAlarm` | Fired when an alarm fires | ✅ |

### chrome.i18n

| Method | Description | Status |
|--------|-------------|--------|
| `chrome.i18n.getMessage(messageName, substitutions)` | Gets a localized message | ✅ |
| `chrome.i18n.getAcceptLanguages(callback)` | Gets accepted languages | ✅ |

### chrome.webRequest (read-only)

| Event | Description | Status |
|-------|-------------|--------|
| `chrome.webRequest.onBeforeRequest` | Fired before a request is sent | ✅ |
| `chrome.webRequest.onBeforeSendHeaders` | Fired before headers are sent | ✅ |
| `chrome.webRequest.onHeadersReceived` | Fired when headers are received | ✅ |
| `chrome.webRequest.onCompleted` | Fired when a request completes | ✅ |
| `chrome.webRequest.onErrorOccurred` | Fired when an error occurs | ✅ |

### chrome.notifications

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.notifications.create(notificationId, options, callback)` | Creates a notification | ✅ |
| `chrome.notifications.clear(notificationId, callback)` | Clears a notification | ✅ |
| `chrome.notifications.getAll(callback)` | Gets all notifications | ✅ |
| `chrome.notifications.onClicked` | Fired when a notification is clicked | ✅ |
| `chrome.notifications.onClosed` | Fired when a notification is closed | ✅ |

### chrome.downloads

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.downloads.download(options, callback)` | Downloads a file | ✅ |
| `chrome.downloads.cancel(downloadId, callback)` | Cancels a download | ✅ |
| `chrome.downloads.pause(downloadId, callback)` | Pauses a download | ✅ |
| `chrome.downloads.resume(downloadId, callback)` | Resumes a download | ✅ |
| `chrome.downloads.search(query, callback)` | Searches downloads | ✅ |
| `chrome.downloads.erase(query, callback)` | Erases download records | ✅ |
| `chrome.downloads.onChanged` | Fired when a download changes | ✅ |
| `chrome.downloads.onCreated` | Fired when a download is created | ✅ |
| `chrome.downloads.onErased` | Fired when a download is erased | ✅ |

### chrome.cookies

| Method/Event | Description | Status |
|--------------|-------------|--------|
| `chrome.cookies.get(details, callback)` | Gets a cookie | ✅ |
| `chrome.cookies.getAll(details, callback)` | Gets all cookies | ✅ |
| `chrome.cookies.set(details, callback)` | Sets a cookie | ✅ |
| `chrome.cookies.remove(details, callback)` | Removes a cookie | ✅ |
| `chrome.cookies.getAllCookieStores(callback)` | Gets all cookie stores | ✅ |
| `chrome.cookies.onChanged` | Fired when a cookie changes | ✅ |

## Manifest Format

thdwb supports Manifest V2 and Manifest V3.

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `manifest_version` | number | 2 or 3 |
| `name` | string | Extension name |
| `version` | string | Extension version |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Extension description |
| `icons` | object | Icon sizes mapped to paths |
| `permissions` | string[] | Required permissions |
| `optional_permissions` | string[] | Optional permissions |
| `background` | object | Background script configuration |
| `content_scripts` | array | Content script definitions |
| `action` | object | Toolbar action (V3) |
| `browser_action` | object | Toolbar action (V2) |
| `options_ui` | object | Options page configuration |

## Deviations from Firefox

1. **Manifest V2 is supported** alongside V3 (Firefox is deprecating V2).
2. **Some permissions are not yet implemented**: `debugger`, `identity`, `proxy`, `management`, `pageCapture`, `printing`, `printingMetrics`, `topSites`, `tabHide`, `tabGroups`, `geolocation`, `unlimitedStorage`, `fileSystem`.
3. **Content scripts run in an isolated world** by default, matching Firefox's ISOLATED world behavior.
4. **Storage is file-based** per extension, not synced to a Firefox account.

## Error Handling

All API methods accept an optional callback as the last argument. If an error
occurs, `chrome.runtime.lastError` is set and the callback is invoked with no
arguments.

```js
chrome.tabs.get(999, (tab) => {
  if (chrome.runtime.lastError) {
    console.error(chrome.runtime.lastError.message);
    return;
  }
  console.log(tab.title);
});
```
