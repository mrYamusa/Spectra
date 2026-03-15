<p align="center">
	<img src="spec.png" alt="SPECTRA — Document Those Changes" width="760" />
</p>

<h1 align="center">Spectra</h1>
<p align="center"><strong>Document those changes.</strong></p>

<p align="center">
	A beginner-friendly CLI that watches your git commits, writes changelogs,
	and updates your README when changes are important enough.
</p>

---

## Why Spectra?

You make commits. Spectra turns those commits into useful project docs.

- ✅ Auto-writes `CHANGELOG.md` entries
- ✅ Can auto-update `README.md` for significant changes
- ✅ Works with local LLMs (like Ollama) or API-based models
- ✅ Safe defaults and readable output for beginners

---

## 5-Minute Beginner Tutorial

### 1) Put `spectra` on your PATH

If you already installed it to `~/go/bin`, you can skip this.

```bash
spectra --help
```

If that command works, Spectra is installed correctly.

### 2) Go into your git project

```bash
cd /path/to/your/project
git status
```

You should be inside a valid git repository.

### 3) Initialize Spectra once

```bash
spectra init --force
```

This does two things:

1. Creates `.spectra.yaml` (your config)
2. Installs a git `post-commit` hook so Spectra can run after each commit

### 4) Check health

```bash
spectra doctor
```

You should see checks for git, repo, config, and mode.

### 5) Generate changelog entries

For latest commit:

```bash
spectra track --commit HEAD
```

For a commit range:

```bash
spectra track --range main..HEAD
```

Now open `CHANGELOG.md` and you will see your entries.

### 6) Preview README updates (safe mode)

```bash
spectra readme --commit HEAD
```

This is a dry run (no file writes).

### 7) Actually update README

```bash
spectra readme --commit HEAD --auto
```

If the commit significance is high enough, Spectra updates the managed section in `README.md`.

---

## Core Commands (Plain English)

| Command                               | What it does                               |
| ------------------------------------- | ------------------------------------------ |
| `spectra init`                        | Creates config + installs post-commit hook |
| `spectra doctor`                      | Checks setup and prerequisites             |
| `spectra track --commit HEAD`         | Adds changelog entry for one commit        |
| `spectra track --range A..B`          | Adds changelog entries for a range         |
| `spectra readme --commit HEAD`        | Shows whether README would be updated      |
| `spectra readme --commit HEAD --auto` | Applies README managed-section update      |

---

## How Spectra Decides README Updates

Spectra scores each commit as:

- `low`
- `medium`
- `high`

Your config key `readme_threshold` controls when README updates happen:

- `low` → update for any commit
- `medium` → update for medium/high commits
- `high` → update only for high-impact commits

---

## Configuration (`.spectra.yaml`)

```yaml
mode: local
local_base_url: http://localhost:11434/v1
api_base_url: https://api.openai.com/v1
model: llama3.1
api_key_env: SPECTRA_API_KEY
readme_threshold: medium
request_timeout_seconds: 30
```

### Choose your model mode

- `mode: local` → local model server (e.g. Ollama)
- `mode: api` → cloud API model (requires env var from `api_key_env`)

---

## Typical Daily Workflow

```bash
# code as usual
git add .
git commit -m "feat: add export command"

# post-commit hook can auto-run track
# optional explicit command:
spectra track --commit HEAD

# only when needed:
spectra readme --commit HEAD --auto
```

---

## Troubleshooting

### `spectra: command not found`

- Ensure binary is in your PATH (for Go installs, usually `~/go/bin`)
- Restart terminal and run `spectra --help`

### `not a git repository`

- Run Spectra inside a folder that has `.git`

### API mode says key missing

- Export your key using the env var name from `api_key_env`

---

## Project Files You’ll See

- `.spectra.yaml` → Spectra settings
- `.git/hooks/post-commit` → auto track hook
- `CHANGELOG.md` → commit history summaries
- `README.md` → docs + managed “Recent Changes” section

---

## Managed Recent Changes

The block below is managed by Spectra and can be auto-updated:

<!-- spectra:readme:start -->

## Recent Changes

_Last updated: 2026-03-15_

- **Initial Spectra scaffold** — README tutorial and CLI workflow docs prepared.
<!-- spectra:readme:end -->
