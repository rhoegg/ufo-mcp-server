# ESP32 UFO WiFi Connectivity Diagnostics Summary

## 🎯 Key Findings

### 1. **WiFi Configuration Flow**
- **NVS Storage**: WiFi credentials are stored in NVS under namespace "Ufo Config" using keys:
  - `STASsid` - WiFi SSID
  - `STAPass` - WiFi password  
  - `WifiMode` - Mode selector (1=WPA2, 2=Enterprise PEAP, 3=Enterprise TLS)
  - `APMode` - Boolean flag for AP vs Station mode

- **Web Provisioning**: The `/config` endpoint in `DynamicRequestHandler.cpp` handles WiFi setup:
  - Saves credentials to NVS via `Config::Write()`
  - Sets `mbAPMode = false` to exit provisioning
  - Triggers `esp_restart()` after saving

### 2. **Critical Issue: Minimal WiFi Event Logging**
The WiFi event handler in `Wifi.cpp` uses mostly `ESP_LOGD` (debug level) for critical events:
- `SYSTEM_EVENT_STA_DISCONNECTED` - Only logs at debug level, no reason code exposed
- `SYSTEM_EVENT_STA_CONNECTED` - No channel/auth info logged
- WiFi scan results (`SYSTEM_EVENT_SCAN_DONE`) - Not handled at all
- No logging of authentication failures or specific disconnect reasons

### 3. **Boot Sequence Analysis**
1. `Ufo::Start()` reads config from NVS
2. If `mbAPMode == false`, attempts STA mode connection
3. `Wifi::Connect()` calls:
   - `esp_wifi_init()`
   - `esp_wifi_set_mode(WIFI_MODE_STA)`
   - `esp_wifi_set_config()` with SSID/password
   - `esp_wifi_start()`
   - `esp_wifi_connect()` - Returns immediately (async)
4. Serial output stops after "wifi: mode : sta" because subsequent events only log at debug level

### 4. **LED State Machine Observations**
- **Green briefly**: Initial state on boot (`mDisplayCharterLevel1/2.SetLeds(0, 15, 0x004400)`)
- **Blue (solid)**: AP mode, no clients connected (StateDisplay.cpp:76-77)
- **Yellow/Orange blinking**: STA mode, not connected (StateDisplay.cpp:109-110)
- The LED going from green→blue suggests the device might be reverting to AP mode

### 5. **Potential Blockers Identified**

#### a) **Missing Restart After Config Save**
The config save flow doesn't appear to be reaching the restart:
```cpp
// DynamicRequestHandler.cpp
mbRestart = true;  // Sets restart flag
// But UfoWebServer::HandleRequest() checks this AFTER response is sent
// If connection drops, restart may not execute
```

#### b) **No WiFi Channel/Auth Mode Validation**
- No explicit channel filtering (device should scan all 2.4GHz channels)
- No hardcoded WPA3/PMF requirements found
- Auth mode is set by router, not enforced by device

#### c) **Race Condition in Boot**
Button press detection happens early in boot:
```cpp
mbButtonPressed = !gpio_get_level(GPIO_NUM_0);  // Line 59 in Ufo.cpp
```
If button is held during boot, it may interfere with normal startup.

### 6. **Missing Diagnostics**
- No WiFi disconnect reason codes logged
- No RSSI/channel info during connection attempts  
- No authentication failure details
- No scan results to verify AP visibility

## 🔧 Recommended Diagnostic Steps

1. **Enable Info-Level Logging**: Change all WiFi event logging from `ESP_LOGD` to `ESP_LOGI`
2. **Add Disconnect Reason Logging**: Log the `reason` field from `system_event_sta_disconnected_t`
3. **Add Connection Details**: Log channel, auth mode, and BSSID on successful connection
4. **Add Scan Done Handler**: Implement `SYSTEM_EVENT_SCAN_DONE` to verify AP detection
5. **Verify Restart Execution**: Add logging before/after `esp_restart()` calls

## 🚨 Most Likely Root Cause
The device appears to save WiFi credentials correctly but fails to connect after restart. The solid blue LED suggests it's reverting to AP mode, possibly due to:
1. Connection failure with no visible error (due to debug-level logging)
2. The restart after config save not executing properly
3. WiFi credentials not persisting across restart (though NVS commit looks correct)

The lack of visible scan/auth output is simply due to insufficient logging levels, not necessarily a connection failure.