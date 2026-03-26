# URL Management Strategy — Comparison Table

Reference table for system design choices regarding how to store and manage the list of URLs to monitor.

| Approach | Pros | Cons |
|---|---|---|
| **YAML file** ⭐ (chosen) | Human-readable, easy to edit, version-controllable, no deps | Must restart or reload to pick up changes |
| Plain `.txt` (one URL per line) | Ultra simple | No per-site settings, no structure |
| GUI/interface in-app | User-friendly | Complex to build, overkill for few sites |
| SQLite database | Queryable, structured | Overkill, harder to edit manually |

## Why YAML?

For a personal tool monitoring < 20 sites, YAML strikes the perfect balance:
- **Edit with any text editor** — no special tooling needed
- **Per-site overrides** — custom check intervals, expected status codes
- **Version-controllable** — track changes in Git
- **Hot-reloadable** — the app includes a "Reload Config" tray menu item, so no restart needed after edits
