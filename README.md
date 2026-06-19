# osu-touch

`osu-touch` is a wireless touch keypad for osu!. A phone opens the web app from your PC over LAN/Wi-Fi, sends a tiny key-state mask over one WebSocket, and the Go server maps that directly to keyboard key down/up events.

It does not auto-tap, time inputs, replay inputs, or press keys without direct user touch input.

## Disclaimer

This project is not affiliated with, endorsed by, or supported by osu!, ppy Pty Ltd, or the official osu! team.

Use this tool at your own risk. It sends synthetic keyboard input to your PC, and the osu! rules / anti-cheat policy may change or interpret external input tools differently. The author does not guarantee that using this tool is allowed or risk-free for online play.

## Requirements

- Windows is currently required. The current input backend uses Win32 `SendInput`; Linux support may be added in the future.
- A phone and PC on the same Wi-Fi/LAN.
- Go, if building from source.

If osu! is running as Administrator, `osu-touch.exe` may also need to be run as Administrator. Windows can block lower-integrity processes from sending input to elevated applications.

## Build

```powershell
go build -trimpath -ldflags="-s -w" -o osu-touch.exe
```

## Run

```powershell
./osu-touch.exe
```

Then open the LAN URL printed by the server from your phone, for example:

```text
http://192.168.1.23:51155
```

Windows Firewall may ask for permission. Allow private network access so the phone can connect.

## Configuration

Configuration is done through environment variables.

### Listen Address

Default:

```text
OSU_TOUCH_ADDR=:51155
```

The default binds to all interfaces, so both `localhost` and LAN IPs can connect.

Examples:

```cmd
set OSU_TOUCH_ADDR=:8081 && osu-touch.exe
```

```cmd
set OSU_TOUCH_ADDR=127.0.0.1:51155 && osu-touch.exe
```

### Input Keys

Default:

```text
OSU_TOUCH_KEYS=ZX
```

`OSU_TOUCH_KEYS` must be exactly two different characters. Only `A-Z` and `0-9` are accepted. Invalid values are ignored and the app falls back to `ZX`.

Examples:

```cmd
set OSU_TOUCH_KEYS=AS && osu-touch.exe
```

```powershell
$env:OSU_TOUCH_KEYS = "DF"
./osu-touch.exe
```

The phone page displays the configured keys, and the server logs the active key pair at startup.

## Touch Behavior

The touch surface is not split into left/right zones. Instead, each accepted new touch alternates between the two configured keys:

```text
touch 1 -> key 1 down
touch 2 -> key 1 up, key 2 down
touch 3 -> key 2 up, key 1 down
...
```

Only one key is intentionally active at a time. Releasing the finger that owns the active key releases that key. Releasing older inactive fingers does nothing.

To avoid same-frame multi-touch bursts creating very short `key1/key2/key1` pulses, extra touches that arrive within a small idle-start guard window are tracked but do not trigger another key switch.

## WebSocket Protocol

The browser sends one binary byte only when the state changes:

```text
bit 0 = configured key 1
bit 1 = configured key 2
0 = no key down
1 = key 1 down
2 = key 2 down
3 = both keys down
```

The current web client intentionally sends only `0`, `1`, or `2`. The server still accepts mask `3` defensively for protocol compatibility.

## Fail-Safe Behavior

- The server releases keys when a WebSocket disconnects or errors.
- The server releases keys during graceful shutdown.
- The phone page sends mask `0` on blur, hidden visibility, page hide, and unload.
- Invalid WebSocket messages are ignored safely.

## Windows SendInput Notes

The current input backend uses Win32 `SendInput` through `user32.dll`, not `keybd_event`.

The `INPUT` struct in `input/sendinput_windows.go` is laid out for 64-bit Windows as:

- `type` as `uint32`
- explicit 4-byte padding so the union payload is 8-byte aligned
- `KEYBDINPUT` with `uintptr` for `dwExtraInfo`
- extra trailing padding so the `INPUT` union is large enough for `MOUSEINPUT`

That gives the expected 64-bit layout: 8-byte header/alignment plus a 32-byte union payload, for a 40-byte `INPUT`. This is the important alignment detail for reliable `SendInput` calls on Windows amd64.
