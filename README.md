# Website Status Checker

**TL;DR** — A tiny Go app that lives in your system tray and shows green/yellow/red dots for your websites. Configure URLs in a YAML file, get instant visibility on what's up and what's down. ~10 MB binary, ~15 MB RAM, zero dependencies to install.

---

## Features

- 🟢🟡🔴 **System tray icon** — aggregated status at a glance (green = all up, yellow = partial, red = all down)
- 📋 **Tray menu** — per-site status with response times
- ⏱️ **Periodic checks** — configurable interval (default: 30s)
- 📄 **YAML config** — add/remove sites by editing `sites.yaml`
- 🔄 **Hot-reload** — apply config changes without restarting (via tray menu)
- 🔔 **Desktop notifications** — toast alerts when a site goes down or recovers
- 🪶 **Lightweight** — single binary, no runtime, minimal resource usage

## Quick Start

### Windows
```bash
# Build (hides the console window)
go build -ldflags="-H=windowsgui" -o status-checker.exe .

# Run
./status-checker.exe
```

### macOS
```bash
# Build
go build -v -o website-status-checker .

# Run
./website-status-checker
```

### Linux
```bash
# Build
go build -v -o website-status-checker .

# Run
./website-status-checker
```

The app starts in the system tray. Click the tray icon to see your websites' current statuses.

## Configuration

Edit `sites.yaml` in the same directory as the executable:

```yaml
settings:
  check_interval: 30      # seconds between checks
  request_timeout: 10     # seconds before marking a site as down
  notify_on_change: true  # desktop notification on status change

sites:
  - name: "My Portfolio"
    url: "https://valentindumas.com"
    expected_status: 200
    check_interval: 10

  - name: "VSDP Productions"
    url: "https://vsdpproductions.com"

  - name: "Craft Agents"
    url: "https://craft-agents.com"
```

Use the **"Reload Config"** menu item in the tray to apply changes without restarting.

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go |
| System tray | [getlantern/systray](https://github.com/getlantern/systray) |
| Config | YAML ([gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)) |
| Notifications | Cross-platform Toast (`gen2brain/beeep`) |

## Project Structure

```
├── main.go              # Entry point
├── sites.yaml           # Your monitored URLs
├── assets/              # Tray icons (embedded in binary)
├── internal/
│   ├── config/          # YAML config loader
│   ├── checker/         # HTTP health checks
│   ├── monitor/         # Background monitoring engine
│   ├── tray/            # System tray UI
│   ├── notify/          # Desktop notifications
│   └── autostart/       # Auto-start on boot (bonus)
└── docs/                # Design decision references
```

## Development

```bash
# Run tests
go test ./...

# Build (Windows console mode, for debugging)
go build -v -o status-checker.exe .

# Build (Windows hidden console mode)
go build -ldflags="-H=windowsgui" -o status-checker.exe .

# Build (macOS / Linux)
go build -v -o website-status-checker .
```

## License

MIT
