# osu-touch

`osu-touch` is a wireless touch keypad for osu!. A mobile browser opens the web app from your PC over LAN/Wi-Fi, sends a tiny key-state mask over one WebSocket, and the Go server maps that directly to keyboard key down/up events on Windows or MIDI note events on Linux.

It does not auto-tap, time inputs, replay inputs, or press keys without direct user touch input.

## Disclaimer

This project is not affiliated with, endorsed by, or supported by osu!, ppy Pty Ltd, or the official osu! team.

Use this tool at your own risk. It sends synthetic keyboard input to your PC, and the osu! rules / anti-cheat policy may change or interpret external input tools differently. The author does not guarantee that using this tool is allowed or risk-free for online play.

## Requirements

- Windows with osu! stable or osu! lazer, or Linux x86_64 with osu! lazer.
- A smartphone and PC on the same Wi-Fi/LAN.
- Linux currently uses osu! lazer's MIDI input support through a virtual ALSA MIDI port.
- Linux releases require ALSA runtime library `libasound.so.2`, which most desktop distros already include.
- Go and platform build dependencies, if building from source.

If osu! is running as Administrator on Windows, `osu-touch.exe` may also need to be run as Administrator. Windows can block lower-integrity processes from sending input to elevated applications.

On Linux, the backend creates an ALSA sequencer MIDI port. If `libasound.so.2` is missing, install `libasound2` on Ubuntu/Debian/Linux Mint, `alsa-lib` on Fedora/Arch/Manjaro, or `libasound2` on openSUSE.

## Build

Windows:

```powershell
go build -trimpath -ldflags="-s -w" -o osu-touch.exe
```

Linux Ubuntu/Debian/Linux Mint build dependencies:

```bash
sudo apt install build-essential pkg-config libasound2-dev
go build -trimpath -ldflags="-s -w" -o osu-touch
```

Linux Arch/Manjaro build dependencies:

```bash
sudo pacman -S gcc pkgconf alsa-lib
go build -trimpath -ldflags="-s -w" -o osu-touch
```

## Run

Windows:

```powershell
./osu-touch.exe
```

Linux:

```bash
./osu-touch
```

Then open the LAN URL printed by the server in your mobile browser, for example:

```text
http://192.168.1.23:51155
```

Windows Firewall may ask for permission. Allow private network access so the browser client can connect. Linux firewalls can block the phone connection too; if the page cannot open, allow the configured TCP port on your private/LAN network.

For osu! lazer on Linux, enable `Device: MIDI` in input settings. Open the key binding settings, click the osu! left/right button bindings, then tap the mobile browser keys to bind them to the emitted notes (`C4` / `D4` by default).

The server also prints a random 6-digit pairing PIN on startup. Enter that PIN in the browser before using the touch surface. The PIN changes every time `osu-touch` starts and is required for the WebSocket control connection. If `osu-touch` restarts, the previous browser session expires and you must enter the new PIN.

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

### Input Keys / MIDI Notes

Windows default:

```text
OSU_TOUCH_KEYS=ZX
```

`OSU_TOUCH_KEYS` is used by the Windows SendInput backend. It must be exactly two different characters. Only `A-Z` and `0-9` are accepted. Invalid values are ignored and the app falls back to `ZX`.

Examples:

```cmd
set OSU_TOUCH_KEYS=AS && osu-touch.exe
```

```powershell
$env:OSU_TOUCH_KEYS = "DF"
./osu-touch.exe
```

Linux MIDI defaults:

```text
OSU_TOUCH_MIDI_NOTES=C4,D4
OSU_TOUCH_MIDI_CHANNEL=1
OSU_TOUCH_MIDI_VELOCITY=127
OSU_TOUCH_MIDI_PORT_NAME=osu-touch MIDI
```

`OSU_TOUCH_MIDI_NOTES` must be two different MIDI notes. Note names such as `C4,D4` or adjacent shorthand `C4D4` are accepted, and raw note numbers such as `60,62` still work. `OSU_TOUCH_MIDI_CHANNEL` must be `1` to `16`, and `OSU_TOUCH_MIDI_VELOCITY` must be `1` to `127`.

The mobile browser page displays compact MIDI note names such as `C4` and `D4`, and the server logs the active mapping at startup.

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

The browser can check the current startup pairing PIN before opening the WebSocket:

```text
/auth?pin=123456
```

The auth check returns `204 No Content` for a valid PIN and `401 Unauthorized` for an invalid or expired PIN.

The WebSocket endpoint requires the current startup pairing PIN as a query parameter:

```text
/ws?pin=123456
```

Connections with a missing or invalid PIN are rejected before the WebSocket is accepted.

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

- The server releases active input when a WebSocket disconnects or errors.
- The server releases all configured input during graceful shutdown.
- The mobile browser page sends mask `0` on blur, hidden visibility, page hide, and unload.
- Invalid WebSocket messages are ignored safely.

## Linux MIDI Notes

The Linux backend uses ALSA sequencer MIDI output through `libasound`. It creates a virtual MIDI source port named `osu-touch MIDI` by default. MIDI notes use middle C as `C4`, so the default `C4,D4` is equivalent to raw note numbers `60,62`. The port is tied to the process and is removed automatically when the app exits or is force closed.

`ReleaseAll()` sends note-off for both configured notes plus MIDI CC 123 (`all notes off`) so normal WebSocket disconnects and graceful shutdowns clear held input like the Windows backend.

## Windows SendInput Notes

The current input backend uses Win32 `SendInput` through `user32.dll`, not `keybd_event`.

The `INPUT` struct in `input/sendinput_windows.go` uses an architecture-aware Win32 layout:

- `type` as `uint32`
- pointer-size-based padding so the union payload starts at the correct offset
- `KEYBDINPUT` with `uintptr` for `dwExtraInfo`
- a union payload sized from `MOUSEINPUT`, because the `INPUT` union must be large enough for its largest member

That gives the expected `INPUT` sizes for release builds: 28 bytes on Windows x86, and 40 bytes on Windows x86_64 / ARM64. Matching these sizes is required for reliable `SendInput` calls; otherwise Windows returns `ERROR_INVALID_PARAMETER`.

## License

This project is licensed under the [MIT License](LICENSE).
