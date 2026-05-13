# BOOTLEGGER

BOOTLEGGER downloads a recording, converts it to a WAV intermediate, and splits it into per-track FLAC files from a timestamped setlist.

It has two interfaces:

- A web server with a browser UI and JSON API
- A local terminal UI built with Bubble Tea

The pipeline is the same in both modes:

1. Download audio with `yt-dlp`
2. Extract PCM WAV with `ffmpeg`
3. Split tracks into FLAC files with sample-accurate timestamps


## Requirements

The following tools must be installed and available in `PATH`:

- `go 1.25.4+`
- [`yt-dlp`](https://github.com/yt-dlp/yt-dlp)
- [`ffmpeg`](https://ffmpeg.org/)

Go dependencies are managed through `go.mod` and `go.sum`; there are no extra runtime services or databases.

## Build

```bash
# Web server
go build -o bootlegger-server ./cmd/bootlegger-server

# Terminal UI
go build -o bootlegger ./cmd/bootlegger
```

## Quick Start

### Web UI

Run the server with the browser UI enabled:

```bash
./bootlegger-server --addr :8080 --web
```

Then open `http://localhost:8080`.

If you only want the JSON API:

```bash
./bootlegger-server --addr :8080
```

### Terminal UI

Run the local terminal interface:

```bash
./bootlegger
```

## Setlist Format

BOOTLEGGER accepts both timestamp-first and timestamp-last lines.

Timestamp first:

```text
0:00 Intro Jam
[2:30] - Song One
1:12:00 Long Finale
```

Timestamp last:

```text
1 - Intro Jam 0:00
02. Song One 2:30
Long Finale 1:12:00
```

Behavior:

- Empty lines are ignored
- Non-matching lines are skipped by the parser
- Each track ends where the next one starts
- The final track uses the full recording duration as its end time

## Web Server

The web server is the primary interface for the project. It serves an HTML frontend and streams pipeline progress back to the browser with Server-Sent Events.

### Flags

| Flag | Description |
|------|-------------|
| `--addr` | HTTP listen address. Default: `:8080` |
| `--web` | Serve the HTML UI in addition to the JSON API |

### API overview

All API routes are prefixed with `/api`.

#### Sessions

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/sessions` | Create a new session |
| `GET` | `/api/sessions/{id}` | Get session state and track list |
| `DELETE` | `/api/sessions/{id}` | Delete a session |

#### Pipeline control

| Method | Path | Body | Description |
|--------|------|------|-------------|
| `POST` | `/api/sessions/{id}/fetch` | `{"url":"..."}` | Fetch metadata and available formats |
| `POST` | `/api/sessions/{id}/setlist` | `{"tracks":[...]}` | Save the track list for a session |
| `POST` | `/api/sessions/{id}/start` | `{"url":"...","format_id":"..."}` | Start the pipeline |
| `POST` | `/api/sessions/{id}/cancel` | None | Cancel a running pipeline |

#### Events

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/events/{id}` | SSE stream of pipeline events |

Current event types include:

- `status`
- `progress-download`
- `progress-extract`
- `progress-split`
- `log`
- `tracks`
- `done`

### API example

```bash
# Create a session
SESSION=$(curl -s -X POST http://localhost:8080/api/sessions | jq -r .id)

# Fetch metadata
curl -s -X POST http://localhost:8080/api/sessions/$SESSION/fetch \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://www.youtube.com/watch?v=XXXXXXXXXXX"}'

# Save tracks
curl -s -X POST http://localhost:8080/api/sessions/$SESSION/setlist \
  -H 'Content-Type: application/json' \
  -d '{
    "tracks": [
      {"number": 1, "title": "Intro Jam", "startTime": 0, "endTime": 150000000000},
      {"number": 2, "title": "Song One", "startTime": 150000000000, "endTime": 465000000000},
      {"number": 3, "title": "Song Two", "startTime": 465000000000, "endTime": 0}
    ]
  }'

# Start the pipeline
curl -s -X POST http://localhost:8080/api/sessions/$SESSION/start \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://www.youtube.com/watch?v=XXXXXXXXXXX","format_id":"251"}'

# Watch progress
curl -N http://localhost:8080/events/$SESSION
```

`startTime` and `endTime` are Go `time.Duration` values encoded as nanoseconds. An `endTime` of `0` on the last track means "use the total recording duration."

## Terminal UI

The terminal UI is the local alternative to the web interface. It is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

Workflow:

1. Paste or type a URL and press `Enter`
2. Choose a source format
3. Enter a setlist
4. Confirm the parsed tracks
5. Start the pipeline

Keyboard reference:

| Key | Context | Action |
|-----|---------|--------|
| `Ctrl+V` | URL input, setlist input | Paste from clipboard |
| `Ctrl+E` | Setlist tab | Toggle textarea input mode |
| `Enter` | Active input/selection | Confirm current step |
| `Esc` | URL input or preview state | Clear or cancel |
| `Up` / `K` | Lists | Move up |
| `Down` / `J` | Lists | Move down |
| `Space` | Track list | Toggle track selection |
| `Q` / `Ctrl+C` | Non-input state | Quit |

## Output Layout

Session data is written under `~/BOOTLEGGER/`:

```text
~/BOOTLEGGER/
└─ sessions/
   └─ <session_id>/
      ├─ source/
      ├─ audio/
      ├─ splits/
      └─ session.json
```

Typical split output:

```text
01 - Intro Jam.flac
02 - Song One.flac
03 - Song Two.flac
```

Session IDs use `YYYYMMDD-HHMMSS`. Output filenames are zero-padded and sanitized for common filesystem-invalid characters.

## Pipeline Details

The stages map directly to external tool invocations.

Download:

```bash
yt-dlp -f <format_id> --no-playlist --newline --progress -o <output_template> <url>
```

If no format is specified, BOOTLEGGER defaults to `bestaudio/best`.

Extract:

```bash
ffmpeg -i <source> -vn -acodec pcm_s16le -ar 48000 -ac 2 <output.wav>
```

Split:

```bash
ffmpeg -y -ss <start> -to <end> -i <output.wav> -c:a flac -compression_level 8 <output.flac>
```

The pipeline intentionally keeps download, extraction, and splitting as separate stages rather than chaining them into one command.

## Dependencies

| Package | Purpose |
|---------|---------|
| [a-h/templ](https://github.com/a-h/templ) | HTML templating for the web UI |
| [gorilla/mux](https://github.com/gorilla/mux) | HTTP routing |
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | Terminal UI framework |
| [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) | Terminal UI components |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling and layout |

## Usage Note

Use this project only with recordings you have the right to download, process, and keep. Site terms, local law, and the status of the source material are your responsibility.
