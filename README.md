# osu-touch

`osu-touch` is a Windows-only wireless touch keypad for osu!. A phone opens the web app from your PC over LAN/Wi-Fi, sends a 1-byte key-state mask over one WebSocket, and the Go server maps that directly to keyboard `Z`/`X` key down/up events with Win32 `SendInput`.

It does not auto-tap, time inputs, replay inputs, or press keys without direct user touch input.

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
http://192.168.1.23:8080
```

Your phone and PC must be on the same Wi-Fi/LAN. Windows Firewall may ask for permission; allow private network access so the phone can connect.

If osu! is running as Administrator, `osu-touch.exe` may also need to be run as Administrator. Windows can block lower-integrity processes from sending input to elevated applications.

## Touch Mapping

- First active finger = `Z`
- Second active finger = `X`
- Third and later fingers are ignored
- Finger identity is tracked with Pointer Events `pointerId`, so moving a finger around the screen does not change its key slot
- The client sends only a binary `Uint8Array([mask])` when the state changes

The mask format is:

```text
bit 0 = Z
bit 1 = X
0 = no key down
1 = Z down
2 = X down
3 = Z + X down
```

## Fail-Safe Behavior

- The server releases both keys when a WebSocket disconnects or errors.
- The server releases both keys during graceful shutdown.
- The phone page sends mask `0` on blur, hidden visibility, page hide, and unload.
- Invalid WebSocket messages are ignored safely.

## Windows SendInput Notes

This project uses Win32 `SendInput` through `user32.dll`, not `keybd_event`, so it is intended for Windows.

The `INPUT` struct in `sendinput_windows.go` is laid out for 64-bit Windows as:

- `type` as `uint32`
- explicit 4-byte padding so the union payload is 8-byte aligned
- `KEYBDINPUT` with `uintptr` for `dwExtraInfo`
- extra trailing padding so the `INPUT` union is large enough for `MOUSEINPUT`

That gives the expected 64-bit layout: 8-byte header/alignment plus a 32-byte union payload, for a 40-byte `INPUT`. This is the important alignment detail for reliable `SendInput` calls on Windows amd64.
