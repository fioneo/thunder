# thunder

A terminal music player written in Go.

- Plays MP3 files from your music directory
- Downloads songs from YouTube with `yt-dlp`
- Has a control socket that reports the player state as JSON and accepts commands

## Requirements

- Go 1.24 or newer
- `yt-dlp` and `ffmpeg` (only for downloading songs)

## Install

Run the installation script. It downloads the source code, builds it,
and installs `thunder` to `~/.local/bin/thunder`:

```sh
curl -fsSL https://raw.githubusercontent.com/fioneo/thunder/main/install.sh | sh
```

Make sure `~/.local/bin` is in your `PATH`.

Or build from source:

```sh
git clone https://github.com/fioneo/thunder
cd thunder
go build -o thunder .
```

## Usage

```sh
thunder [options] [song]
```

| Option | Description | Default |
|---|---|---|
| `-m, --music` | Music directory | `~/Music` |
| `-v, --volume` | Initial volume, 0-100 | `100` |
| `-s, --sock` | Control socket path | `thunder.sock` in the temp dir |
| `-l, --log` | Log file path | `thunder.log` in the temp dir |
| `-h, --help` | Print help and exit | |

Examples:

```sh
thunder
thunder -m ~/music
thunder -v 50 ~/Music/song.mp3
```

## Keys

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Switch between panels |
| `q` / `Ctrl+Q` | Quit |

## Control socket

Thunder writes the current state to the control socket as JSON:

```json
{
  "name": "song.mp3",
  "position": "1:23",
  "length": "3:45",
  "volume": 80,
  "muted": false,
  "playing": true,
  "paused": false
}
```

This makes it easy to show the current song in a status bar.

The socket also accepts commands:

| Command | Action |
|---|---|
| `TOGGLE_PAUSE` | Pause or resume playback |
| `TOGGLE_MUTE` | Mute or unmute the sound |
| `PREVIOUS` | Play the previous song |
| `NEXT` | Play the next song |
| `VOL_DOWN` | Decrease the volume |
| `VOL_UP` | Increase the volume |
| `STOP` | Stop playback |
| `CYCLE_MODE` | Switch to the next play mode |

Commands are case-sensitive. A trailing newline is ignored.

To control the player without switching to its window, write your own program
or use the existing one: [thctl](https://github.com/fioneo/thctl).

## Logs

Errors are shown on the screen in short form. The full details are written to
the log file.
