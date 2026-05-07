# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.0] - 2026-05-06

### Added

- **Edit entries** — Press `e` on any entry in the list to edit date, description, project, tags, time, or billable status.
- **Non-countable hours** — Entries tagged with `Freelance` (id 3) or `Overtime` (ids 104–106) are tracked separately and don't count toward the 44h weekly goal. Shown as yellow `+Xh` in day headers.
- **Changelog viewer** — Press `c` to see the full changelog. Auto-shown after updates.
- **Version display** — Current version shown in the header (hidden for `dev` builds).
- **Weekday summary** — Persistent Mon–Fri boxes showing daily hours while browsing entries or filling the form.
- **Empty-day picker** — On the date step, shows unfilled weekdays with `↑/↓` selection.
- **Description suggestions** — While typing description, press `↑/↓` to pick from previous entries; pre-fills project, tags, time, and billable.
- **Human-readable time input** — Accepts `5h30m`, `1h`, `30m`, `5:30`, or plain minutes.
- **Quick-add after save** — After creating an entry, keeps date and project pre-filled so you can log another task quickly.
- **Form navigation** — `Esc` goes back one step in the form instead of canceling everywhere except the date step.
- **Billable default** — New entries now default to billable (`true`).

### Changed

- Entries sorted newest-first (most recent date and ID at the top).
- Fetch range expanded to two weeks (Monday past week → Sunday current week).
- Progress bars and day summaries exclude freelance/overtime from goal calculations.

## [0.6.0] - 2026-05-03

### Added

- Loading spinner for async operations.
- Daily and weekly progress bars with per-day targets (Mon–Thu 9h, Fri 8h).
- Persistent progress section visible in the entries list.

## [0.5.0] - 2026-05-01

### Added

- PATH setup instructions for macOS/Linux after installation.
- Improved self-update binary replacement flow.

## [0.4.0] - 2026-05-01

### Added

- Update banner showing the latest version and download URL when an update is available.

## [0.3.0] - 2026-05-01

### Changed

- Cleaned up entries view layout.

## [0.2.0] - 2026-05-01

### Added

- In-app update checker via GitHub releases.
- Self-update with `u` key.
- Installation script (`install.sh`).

## [0.1.0] - 2026-05-01

### Added

- Initial release.
- Login with API Bearer token.
- List time entries with weekly / two-week toggle.
- Create new time entries.
- Delete time entries.

[Unreleased]: https://github.com/json-nan/var-cli/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/json-nan/var-cli/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/json-nan/var-cli/compare/v0.5.0...v0.6.0
