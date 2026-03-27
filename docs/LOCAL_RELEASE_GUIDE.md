# Local Release Guide

This guide explains how to use GoReleaser to compile and package `website-status-checker` on your local machine.

## Why is Local Building Complex?
Because this application relies on the system tray, it uses `CGO` to hook into native operating system user-interface components (like the Windows Taskbar or macOS Menu Bar). 
Trying to cross-compile for Windows or Linux from a Mac (or vice-versa) is difficult because the build process needs the target operating system's native C headers (`libgtk-3-dev`, `mingw-w64`, etc.).

Therefore, **we strongly recommend relying on the Github Actions pipeline `.github/workflows/release.yml` for actual releases.**

However, it is extremely beneficial to build and test *your native OS version* (e.g., macOS binary on a MacBook) locally before pushing a tag.

---

## 1. Prerequisites

Before you can build a release locally, you must install GoReleaser:

**macOS:**
```bash
brew install goreleaser/tap/goreleaser
```

**Linux:**
```bash
go install github.com/goreleaser/goreleaser/v2@latest
```

## 2. Test the configuration formatting

Before committing changes, you can validate the syntax of your configuration files:
```bash
goreleaser check --config .goreleaser.darwin.yaml
```

## 3. Create a Local Snapshot (Recommended)

A "Snapshot" release is the safest way to build the application natively.
It bypasses GitHub's API token requirements, it skips git tag validation, and compiles your application directly into a `dist/` directory on your machine.

**Run the build targeting your current OS:**

*(If you are on macOS:)*
```bash
goreleaser release --snapshot --clean --config .goreleaser.darwin.yaml
```

*(If you are on Windows in WSL/Linux:)*
```bash
goreleaser release --snapshot --clean --config .goreleaser.linux.yaml
```

**Where are the files?**
Look inside the `dist/` folder that was just created. You will find your raw binary executable and `.zip` archives.

> **Note:** The GoReleaser Community Edition generates `.zip` archives containing the raw `website-status-checker` binary on macOS. Generating a `.app` wrapper natively requires GoReleaser Pro.

## 4. Building other platforms locally (Advanced)

If you are on macOS but strictly need to build the Linux `.deb` locally, you must compile inside a container that has Linux UI headers pre-installed.

Instead of running GoReleaser natively, you can pass your configuration into a Docker container:

```bash
docker run --rm -v $PWD:/app -w /app goreleaser/goreleaser release --snapshot --clean --config .goreleaser.linux.yaml
```

*(Note: Ensure Docker Desktop is running before executing this command).*
