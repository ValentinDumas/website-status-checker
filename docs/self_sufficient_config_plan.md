# Self-Sufficient Executable & Editable Config Plan

The goal is to eliminate the need for having a `sites.yaml` file sitting next to the executable, while still allowing the user to update the configuration across all platforms via the system tray.

> [!NOTE]
> Because files embedded directly into a Go executable using `//go:embed` are **read-only**, we cannot save changes back into the `.exe` itself. 
> To solve this, we will use the embedded file as a "Default Template" and extract it to a standard, writable OS application folder on first boot.

## Proposed Implementation

### 1. OS-Specific Config Directory
Instead of looking for `sites.yaml` in the current folder, the app will use Go's built-in `os.UserConfigDir()` to store settings natively:
- **Windows**: `%AppData%\WebsiteStatusChecker\sites.yaml`
- **macOS**: `~/Library/Application Support/WebsiteStatusChecker/sites.yaml`
- **Linux**: `~/.config/WebsiteStatusChecker/sites.yaml`

### 2. Embed & Extract (First Run)
- We will use `//go:embed default.yaml` to bake your initial configuration directly into the Go binary.
- On startup, the app checks if the native config folder exists.
- If the file is missing (first run), it writes the embedded `default.yaml` into the OS config directory.
- It then reads the config from the native directory.

### 3. Cross-Platform "Edit Config" Menu
- We will add an **"✏️ Edit Configuration"** button to the tray menu.
- When clicked, it will open the `sites.yaml` file in the user's default text editor using native OS commands:
  - **Windows**: `cmd /c start <path>`
  - **macOS**: `open <path>`
  - **Linux**: `xdg-open <path>`
- The user can make their changes, save the file, and click the existing **"📄 Reload Config"** tray button to apply them instantly.

## Architecture

### Configuration Core
**`internal/config/paths.go`**
- `GetConfigPath()`: Uses `os.UserConfigDir()` to get the OS-native path.
- `EnsureConfigExists(defaultData []byte)`: Creates the directory and writes the default file if it doesn't exist.

**`internal/config/config.go`**
- Embeds `default.yaml` via `//go:embed default.yaml`.
- Updates `LoadConfig` to utilize `GetConfigPath` and `EnsureConfigExists`.

### Tray Menu Updates
**`internal/tray/editor.go`**
- Implements `openEditor(path string) error` using `runtime.GOOS` switches to execute `xdg-open`, `open`, or `start`.

**`internal/tray/tray.go`** 
- Adds `editConfigItem` to the menu.
- Adds `handleEditConfig()` click event.

### Refactoring `main.go`
- Removes the hardcoded local `sites.yaml` path.
- Initializes the system using the native configuration path.
