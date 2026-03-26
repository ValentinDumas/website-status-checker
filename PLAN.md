# Website Status Checker — Implementation Plan

> [!IMPORTANT]
> **Development philosophy**: This plan is implemented **step by step**, with each step followed by a verification step to ensure everything is working, running, and tested before moving on. The core values are **QUALITY**, **MAINTAINABILITY**, and **READABILITY**.

A lightweight system tray application that periodically checks the availability of your websites and displays their status as colored indicators directly in the taskbar notification area.

---

## Technology Choice

**Go + systray** — chosen for this project.

| Criteria | Value |
|---|---|
| **Language** | Go |
| **Tray library** | `github.com/getlantern/systray` |
| **Config format** | YAML (`gopkg.in/yaml.v3`) |
| **Binary size** | ~8–12 MB |
| **Memory (idle)** | ~10–20 MB |
| **Startup time** | Instant |
| **Cross-platform** | Win / Mac / Linux from single codebase |
| **Distribution** | Single `.exe`, no runtime dependencies |

---

## High-Level Architecture

```mermaid
graph TB
    subgraph "Website Status Checker"
        CONFIG["📄 sites.yaml<br/>URL list + settings"]
        MONITOR["🔄 Monitor Loop<br/>(goroutine per site)"]
        CHECKER["🌐 HTTP Checker<br/>(GET request + timeout)"]
        STATE["📊 Status Store<br/>(in-memory state)"]
        TRAY["🖥️ System Tray<br/>(icon + menu)"]
        NOTIF["🔔 Notifications<br/>(desktop toast)"]
    end

    CONFIG -->|"Load URLs"| MONITOR
    MONITOR -->|"Periodic check"| CHECKER
    CHECKER -->|"Update"| STATE
    STATE -->|"Render"| TRAY
    STATE -->|"On status change"| NOTIF
    TRAY -->|"Click: show menu"| STATE
    TRAY -->|"Reload Config"| CONFIG
```

### How It Works

1. **On startup**: Load `sites.yaml`, start background goroutines for each site
2. **Every 30 seconds** (configurable): Perform HTTP GET on each URL, record status code + response time
3. **Update tray icon**: Aggregate all statuses → green (all up), yellow (some down), red (all down)
4. **Tray tooltip**: Shows summary (e.g., "3/3 sites up")
5. **Tray menu** (click): Shows each site with a ● green/red indicator + response time
6. **Menu items**: "Refresh Now", "Reload Config", separator, "Quit"
7. **On status change**: Desktop notification (Phase 5)

---

## URL Management — Config File: `sites.yaml`

```yaml
# Website Status Checker Configuration

settings:
  check_interval: 30        # seconds between checks
  request_timeout: 10       # seconds before marking a site as down
  notify_on_change: true    # desktop notification when status changes

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

> The tray menu includes a **"Reload Config"** item to hot-reload `sites.yaml` without restarting the application.

### Reference Documents

- [URL management comparison](file:///s:/IdeaProjects/website-status-checker/docs/url-management-comparison.md) — why YAML was chosen over alternatives
- [Monitoring approach comparison](file:///s:/IdeaProjects/website-status-checker/docs/monitoring-approach-comparison.md) — simple vs. pooling vs. worker pool vs. queue
- [Authentication strategies](file:///s:/IdeaProjects/website-status-checker/docs/authentication-strategies.md) — auth types explained with use cases
- [Sample auth config](file:///s:/IdeaProjects/website-status-checker/docs/sample-auth-config.yaml) — example YAML for protected endpoints (for future use)

---

## Phase 1 — Project Setup & Config Loading

> **Goal**: Compilable Go project that loads and validates `sites.yaml`.

### Low-Level Design — Phase 1

```mermaid
classDiagram
    class Config {
        +Settings Settings
        +Sites []Site
    }
    class Settings {
        +int CheckInterval
        +int RequestTimeout
        +bool NotifyOnChange
    }
    class Site {
        +string Name
        +string URL
        +int CheckInterval
        +int ExpectedStatus
    }

    Config "1" *-- "1" Settings : contains
    Config "1" *-- "1..*" Site : contains

    class ConfigLoader {
        -string filePath
        +LoadConfig(path string) Config, error
        +ReloadConfig() Config, error
        -validate(config Config) error
        -applyDefaults(config *Config)
    }

    ConfigLoader ..> Config : produces
```

```mermaid
sequenceDiagram
    participant Main as main.go
    participant CL as ConfigLoader
    participant FS as sites.yaml

    Main->>CL: LoadConfig("sites.yaml")
    CL->>FS: Read file
    FS-->>CL: Raw YAML bytes
    CL->>CL: yaml.Unmarshal → Config struct
    CL->>CL: applyDefaults()
    CL->>CL: validate()
    CL-->>Main: Config, nil
```

### Files

#### [NEW] `go.mod`

Go module definition with dependencies:
- `github.com/getlantern/systray`
- `gopkg.in/yaml.v3`

#### [NEW] `sites.yaml`

Default configuration with your real sites (Portfolio, VSDP Productions, Craft Agents).

#### [NEW] `internal/config/config.go`

- `Config`, `Settings`, `Site` structs with YAML tags
- `LoadConfig(path string) (*Config, error)` — read + unmarshal + validate
- `applyDefaults()` — fill missing optional fields with sensible values
- `validate()` — ensure required fields (name, URL) are present, URL is valid

### ✅ Verification — Phase 1

- `go build ./...` compiles without errors
- Unit tests for `config.go`: valid YAML, invalid YAML, missing file, default values, validation errors
- Run: `go test ./internal/config/...`

---

## Phase 2 — HTTP Health Checker

> **Goal**: A tested checker module that can probe a URL and return structured results.

### Low-Level Design — Phase 2

```mermaid
classDiagram
    class Checker {
        -http.Client httpClient
        +NewChecker(timeout time.Duration) *Checker
        +Check(site Site) Result
    }
    class Result {
        +string SiteName
        +string URL
        +int StatusCode
        +time.Duration ResponseTime
        +bool IsUp
        +error Error
        +time.Time CheckedAt
    }
    class Site {
        +string Name
        +string URL
        +int ExpectedStatus
    }

    Checker ..> Result : produces
    Checker ..> Site : reads
```

```mermaid
sequenceDiagram
    participant C as Checker
    participant HTTP as net/http Client
    participant S as Target Website

    C->>C: Record start time
    C->>HTTP: GET site.URL (with timeout)
    HTTP->>S: HTTP GET request

    alt Site responds
        S-->>HTTP: HTTP Response (status code)
        HTTP-->>C: *http.Response
        C->>C: Calculate ResponseTime
        C->>C: Compare StatusCode vs ExpectedStatus
        C->>C: Build Result{IsUp: true/false}
    else Timeout or connection error
        HTTP-->>C: error (timeout / refused)
        C->>C: Build Result{IsUp: false, Error: err}
    end

    C-->>C: Return Result
```

### Files

#### [NEW] `internal/checker/checker.go`

- `Checker` struct wrapping `http.Client` with configurable timeout
- `NewChecker(timeout time.Duration) *Checker`
- `Check(site config.Site) Result` — performs GET, measures response time
- `Result` struct: `SiteName`, `URL`, `StatusCode`, `ResponseTime`, `IsUp`, `Error`, `CheckedAt`
- IsUp logic: if `ExpectedStatus` is set, match exactly; otherwise accept any `2xx`
- Follows redirects, validates TLS

### ✅ Verification — Phase 2

- Unit tests with `httptest.NewServer` (mock HTTP server)
- Test cases: 200 OK, 500 error, timeout, connection refused, redirect, custom expected status
- Run: `go test ./internal/checker/...`

---

## Phase 3 — Monitoring Engine

> **Goal**: Background monitor that periodically checks all sites and maintains state.

### Low-Level Design — Phase 3

```mermaid
classDiagram
    class Monitor {
        -Config config
        -Checker checker
        -map~string,SiteStatus~ statuses
        -sync.RWMutex mu
        -context.CancelFunc cancel
        -OnStatusChange func(SiteStatus)
        +NewMonitor(config, checker) *Monitor
        +Start()
        +Stop()
        +GetStatuses() []SiteStatus
        +RefreshAll()
        +ReloadConfig(config Config)
        -monitorSite(site Site, ctx context.Context)
        -updateStatus(name string, result Result)
    }
    class SiteStatus {
        +Site Site
        +Result LatestResult
        +Result PreviousResult
        +bool StatusChanged
    }
    class Checker {
        +Check(site Site) Result
    }
    class Config {
        +Settings Settings
        +Sites []Site
    }

    Monitor "1" --> "1" Checker : uses
    Monitor "1" --> "1" Config : reads
    Monitor "1" *-- "*" SiteStatus : maintains
```

```mermaid
sequenceDiagram
    participant Main as main.go
    participant M as Monitor
    participant G as Goroutine (per site)
    participant C as Checker
    participant S as StatusStore

    Main->>M: Start()
    loop For each site in Config
        M->>G: go monitorSite(site, ctx)
    end

    loop Every check_interval seconds
        G->>C: Check(site)
        C-->>G: Result
        G->>S: updateStatus(result)
        S->>S: Compare previous vs current
        alt Status changed
            S->>M: OnStatusChange(SiteStatus)
        end
    end

    Main->>M: Stop()
    M->>G: Cancel context (all goroutines exit)
```

```mermaid
flowchart LR
    subgraph "Thread Safety"
        W1["Goroutine: Site 1"] -->|"Lock()"| MU["sync.RWMutex"]
        W2["Goroutine: Site 2"] -->|"Lock()"| MU
        W3["Goroutine: Site 3"] -->|"Lock()"| MU
        MU -->|"write"| MAP["map[string]SiteStatus"]
        TRAY["Tray (read)"] -->|"RLock()"| MU
        MU -->|"read"| MAP
    end
```

### Files

#### [NEW] `internal/monitor/monitor.go`

- `Monitor` struct with config, checker, and thread-safe status map
- `NewMonitor(config *Config, checker *Checker) *Monitor`
- `Start()` — launches one goroutine per site using `context.Context` for cancellation
- `Stop()` — cancels context, waits for goroutines to exit
- `monitorSite()` — per-site loop: check → update → sleep
- `updateStatus()` — acquire write lock, update map, detect status change
- `GetStatuses() []SiteStatus` — acquire read lock, return copy
- `RefreshAll()` — trigger immediate check on all sites
- `ReloadConfig(config *Config)` — stop all goroutines, restart with new config
- `OnStatusChange` callback — invoked on up→down or down→up transitions

### ✅ Verification — Phase 3

- Unit tests: start/stop lifecycle, status updates, concurrent access safety
- Integration test: monitor with mock HTTP endpoints, verify state transitions
- Test status change detection: mock server toggles between 200 and 500
- Run: `go test ./internal/monitor/...`

---

## Phase 4 — System Tray UI

> **Goal**: Working tray application with icon, tooltip, and menu on Windows.

### Low-Level Design — Phase 4

```mermaid
classDiagram
    class TrayManager {
        -Monitor monitor
        -ConfigLoader configLoader
        -[]menuItem siteItems
        +NewTrayManager(monitor, configLoader) *TrayManager
        +OnReady()
        +OnExit()
        -setupIcon()
        -buildMenu()
        -updateLoop()
        -handleRefresh()
        -handleReloadConfig()
        -handleQuit()
        -aggregateStatus(statuses []SiteStatus) StatusLevel
    }
    class StatusLevel {
        <<enumeration>>
        AllUp
        PartialDown
        AllDown
    }
    class IconAssets {
        <<embedded>>
        +[]byte IconGreen
        +[]byte IconYellow
        +[]byte IconRed
    }

    TrayManager ..> StatusLevel : determines
    TrayManager ..> IconAssets : uses
    TrayManager --> Monitor : reads statuses
    TrayManager --> ConfigLoader : triggers reload
```

```mermaid
sequenceDiagram
    participant SYS as systray.Run()
    participant TM as TrayManager
    participant M as Monitor
    participant UI as System Tray (OS)

    SYS->>TM: OnReady()
    TM->>TM: setupIcon(green)
    TM->>TM: buildMenu()
    TM->>UI: Set icon, tooltip, menu items
    TM->>TM: go updateLoop()

    loop Every 2 seconds (UI refresh)
        TM->>M: GetStatuses()
        M-->>TM: []SiteStatus
        TM->>TM: aggregateStatus()
        TM->>UI: Update icon (green/yellow/red)
        TM->>UI: Update tooltip ("3/3 sites up")
        TM->>UI: Update menu items (● Site Name 142ms)
    end

    alt User clicks "Refresh Now"
        UI->>TM: handleRefresh()
        TM->>M: RefreshAll()
    end

    alt User clicks "Reload Config"
        UI->>TM: handleReloadConfig()
        TM->>TM: configLoader.ReloadConfig()
        TM->>M: ReloadConfig(newConfig)
        TM->>TM: buildMenu() (rebuild for new sites)
    end

    alt User clicks "Quit"
        UI->>TM: handleQuit()
        TM->>M: Stop()
        TM->>SYS: systray.Quit()
    end
```

```mermaid
graph TB
    subgraph "Tray Menu Layout"
        TITLE["─── Website Status ───"]
        S1["● My Portfolio          12ms"]
        S2["● VSDP Productions      89ms"]
        S3["● Craft Agents         142ms"]
        SEP1["──────────────────────"]
        REFRESH["🔄 Refresh Now"]
        RELOAD["📄 Reload Config"]
        SEP2["──────────────────────"]
        QUIT["❌ Quit"]
    end

    TITLE --- S1 --- S2 --- S3 --- SEP1 --- REFRESH --- RELOAD --- SEP2 --- QUIT
```

### Files

#### [NEW] `assets/icon_green.ico`, `assets/icon_yellow.ico`, `assets/icon_red.ico`

Tray icon assets (16×16 and 32×32 `.ico`):
- 🟢 Green = all sites up
- 🟡 Yellow = some sites down
- 🔴 Red = all sites down
- Embedded in binary via `//go:embed`

#### [NEW] `internal/tray/tray.go`

- `TrayManager` struct coordinating monitor and UI
- `OnReady()` — called by systray, sets up icon + menu + starts update loop
- `OnExit()` — cleanup on quit
- `buildMenu()` — dynamically creates menu entries from config
- `updateLoop()` — polls monitor every 2s, updates icon/tooltip/menu text
- `aggregateStatus()` — returns `AllUp` / `PartialDown` / `AllDown`
- Menu handlers: Refresh Now, Reload Config, Quit

#### [NEW] `main.go`

Application entry point:
- Load config → create checker → create monitor → create tray manager
- `systray.Run(trayManager.OnReady, trayManager.OnExit)`
- Graceful shutdown on Quit

### ✅ Verification — Phase 4

- Build: `go build -ldflags="-H=windowsgui" -o status-checker.exe`
- Manual test: tray icon appears with green icon
- Tooltip shows "Website Status: 3/3 sites up"
- Menu shows all 3 sites with ● green indicators and response times
- Stop one site (edit URL to invalid) → icon turns yellow, menu shows ● red for that site
- Use "Reload Config" → verify changes apply without restart
- Use "Refresh Now" → verify immediate check
- Use "Quit" → verify clean exit, no zombie processes

---

## Phase 5 — Desktop Notifications

> **Goal**: Toast notifications on status changes (site goes down or recovers).

### Low-Level Design — Phase 5

```mermaid
classDiagram
    class Notifier {
        <<interface>>
        +Send(title string, message string) error
    }
    class WindowsNotifier {
        +Send(title string, message string) error
    }
    class Monitor {
        +OnStatusChange func(SiteStatus)
    }

    Notifier <|.. WindowsNotifier : implements
    Monitor ..> Notifier : calls on status change
```

```mermaid
sequenceDiagram
    participant M as Monitor
    participant N as WindowsNotifier
    participant OS as Windows Toast API

    Note over M: Site status changes (up → down)
    M->>M: OnStatusChange(SiteStatus)
    M->>N: Send("🔴 Site Down", "VSDP Productions is unreachable")
    N->>OS: Show toast notification
    OS-->>N: Displayed

    Note over M: Site recovers (down → up)
    M->>M: OnStatusChange(SiteStatus)
    M->>N: Send("🟢 Site Recovered", "VSDP Productions is back up (142ms)")
    N->>OS: Show toast notification
```

### Files

#### [NEW] `internal/notify/notify.go`

- `Notifier` interface: `Send(title, message string) error`
- `WindowsNotifier` struct implementing `Notifier`
- Uses `go-toast` library or PowerShell fallback
- Only triggers on transitions (up→down, down→up), never on stable states

#### [MODIFY] `internal/monitor/monitor.go`

- Wire `OnStatusChange` callback to Notifier

### ✅ Verification — Phase 5

- Edit `sites.yaml` to add a known-down URL → verify "Site Down" notification appears
- Fix the URL → verify "Site Recovered" notification appears
- Verify no duplicate notifications on consecutive checks with same status
- Run: `go test ./internal/notify/...`

---

## Phase 6 (Bonus) — Auto-Start on Boot

> Nice-to-have. Only implement if previous phases are solid.

### Low-Level Design — Phase 6

```mermaid
classDiagram
    class AutoStarter {
        <<interface>>
        +Enable(exePath string) error
        +Disable() error
        +IsEnabled() bool
    }
    class WindowsAutoStarter {
        -string registryKey
        -string appName
        +Enable(exePath string) error
        +Disable() error
        +IsEnabled() bool
    }

    AutoStarter <|.. WindowsAutoStarter : implements
```

```mermaid
sequenceDiagram
    participant U as User (tray menu)
    participant TM as TrayManager
    participant AS as WindowsAutoStarter
    participant REG as Windows Registry

    U->>TM: Click "Start on Boot"
    TM->>AS: IsEnabled()
    AS->>REG: Read HKCU\...\Run
    REG-->>AS: exists? true/false
    AS-->>TM: bool

    alt Currently disabled → enable
        TM->>AS: Enable(exePath)
        AS->>REG: Set value at HKCU\Software\Microsoft\Windows\CurrentVersion\Run
        TM->>TM: Update menu item "Start on Boot ✓"
    else Currently enabled → disable
        TM->>AS: Disable()
        AS->>REG: Delete value
        TM->>TM: Update menu item "Start on Boot ✗"
    end
```

### Files

#### [NEW] `internal/autostart/autostart.go`

- `AutoStarter` interface for cross-platform future
- `WindowsAutoStarter`: reads/writes `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- Toggle via tray menu item: "Start on Boot ✓/✗"

#### [MODIFY] `internal/tray/tray.go`

- Add "Start on Boot" toggle menu item before "Quit"

### ✅ Verification — Phase 6

- Toggle on → reboot → verify app starts automatically
- Toggle off → reboot → verify app does not start
- Verify registry key is correctly created/removed

---

## Cross-Platform Notes (Low Priority)

| Platform | Icon location | Menu behavior | Notification system | Auto-start |
|---|---|---|---|---|
| **Windows** | System tray (bottom-right) | Click → context menu | Windows Toast | Registry key |
| **macOS** | Menu bar (top-right) | Click → dropdown menu | Notification Center | LaunchAgent plist |
| **Linux** | Depends on DE (GNOME, KDE) | Click → context menu | `notify-send` / D-Bus | systemd user service |

> [!NOTE]
> The `getlantern/systray` library abstracts most platform differences. Cross-platform support mainly requires: (1) cross-compiling via `GOOS`/`GOARCH`, (2) platform-specific notification backends in `notify.go`, and (3) platform-specific auto-start registration.

---

## Project Structure

```
website-status-checker/
├── main.go                      # Entry point
├── go.mod                       # Module definition
├── go.sum                       # Dependency checksums
├── sites.yaml                   # User config (URLs to monitor)
├── PLAN.md                      # This implementation plan
├── docs/
│   ├── url-management-comparison.md      # Design decision: why YAML
│   ├── monitoring-approach-comparison.md  # Design decision: simple vs. pooling
│   ├── authentication-strategies.md       # Design reference: auth types
│   └── sample-auth-config.yaml            # Example: auth in sites.yaml
├── assets/
│   ├── icon_green.ico           # All sites up
│   ├── icon_yellow.ico          # Some sites down
│   └── icon_red.ico             # All sites down
├── internal/
│   ├── config/
│   │   └── config.go            # YAML config loader
│   ├── checker/
│   │   └── checker.go           # HTTP health checks
│   ├── monitor/
│   │   └── monitor.go           # Monitoring engine
│   ├── tray/
│   │   └── tray.go              # System tray UI
│   ├── notify/
│   │   └── notify.go            # Desktop notifications (Phase 5)
│   └── autostart/
│       └── autostart.go         # Auto-start on boot (Phase 6 bonus)
└── README.md
```

---

## Summary of Decisions

| Question | Decision |
|---|---|
| Technology | Go + systray |
| Config format | YAML file with hot-reload via tray menu |
| Check interval | 30 seconds default (10s for Portfolio) |
| Monitoring approach | Simple (one goroutine per site, < 20 sites) |
| Tray interaction | Click → menu only (KISS) |
| Tooltip | Summary: "3/3 sites up" |
| Authentication | Not needed now; reference docs + sample config saved for future |
| Notifications | Phase 5 — toast on status changes |
| Auto-start | Phase 6 — bonus, nice-to-have |
| Verification | At every phase, before moving on |
| Core values | Quality, Maintainability, Readability |
