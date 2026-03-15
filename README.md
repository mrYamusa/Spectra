<p align="center">
  <img src="assets/specs.webp" alt="SPECTRA — Document Those Changes" width="780" />
</p>

<h1 align="center">Spectra</h1>
<p align="center"><strong>Document those changes.</strong></p>

<p align="center">
  Spectra watches your git commits and writes useful docs for you.
  It can update your <code>CHANGELOG.md</code> and (when needed) your <code>README.md</code>.
</p>

<p align="center">
  <a href="docs/releasing.md">Maintainers: Release Guide</a>
</p>

---

## Start Here

If you only do one thing, do this flow in order:

1. Download `spectra.exe` from GitHub Releases.
2. Put `spectra.exe` in a folder you can find (for example `C:\Tools\Spectra`).
3. Open your project folder in terminal.
4. Run `spectra init` and answer the setup questions.
5. Run `spectra doctor` to confirm setup.
6. Run `spectra track --commit HEAD` after a commit.

---

## Windows: Download EXE (No Build Needed)

### For users

1. On this page.
2. Click **Releases** on the right side.
3. Open the latest release.
4. Under **Assets**, download `spectra.exe`.

If you are a project maintainer and want release/publishing steps, see `docs/releasing.md`.

---

## What Spectra Does

- `track`: add commit summaries to `CHANGELOG.md`
- `untrack`: remove a previously tracked commit from `CHANGELOG.md`
- `readme`: decide if commit is important enough to update README section
- `doctor`: show if your setup is healthy

---

## First-Time Setup

### Step 1 — Open terminal in your project

Your project must already be a git repo.

```bash
git status
```

If this fails with “not a git repository”, run:

```bash
git init
```

### Step 2 — Start Spectra wizard

```bash
spectra init
```

The wizard asks beginner-friendly questions and lets you press Enter for defaults.

### Step 3 — Pick a mode

- `local` = use a model running on your own machine
- `api` = use an online model provider

If you are unsure, use `local` first.

### Step 4 — API mode users only

If you picked `api`, Spectra asks for:

- `api_base_url` (provider endpoint)
- `api_key_env` (name of your env var, not your secret key)
- `model` (provider model id)

Example API config values:

```yaml
mode: api
api_base_url: https://generativelanguage.googleapis.com/v1beta/openai
api_key_env: SPECTRA_API_KEY
model: gemini-2.0-flash
```

Set your real key in terminal:

```bash
export SPECTRA_API_KEY="your-real-api-key"
```

### Step 5 — Verify everything

```bash
spectra doctor
```

You want to see:

- `git: ok`
- `config: ok`
- `mode: ...`
- `api_key: present (...)` if using API mode

---

## Daily Use

After each commit, you can run:

```bash
spectra track --commit HEAD
```

If you tracked a commit by mistake, undo it:

```bash
spectra untrack --commit HEAD
```

Preview README update decision (safe):

```bash
spectra readme --commit HEAD
```

Apply README update:

```bash
spectra readme --commit HEAD --auto
```

---

## Quick Command Guide

| Command                               | What it means in plain words         |
| ------------------------------------- | ------------------------------------ |
| `spectra init`                        | Set up Spectra with guided questions |
| `spectra doctor`                      | Check if setup is working            |
| `spectra track --commit HEAD`         | Add the latest commit to changelog   |
| `spectra track --range main..HEAD`    | Add many commits at once             |
| `spectra untrack --commit HEAD`       | Remove latest commit from changelog  |
| `spectra readme --commit HEAD`        | Dry run for README update            |
| `spectra readme --commit HEAD --auto` | Actually update README section       |

---

## Common Mistakes and Fixes

### “spectra: command not found”

- Put `spectra.exe` somewhere on PATH, or run it with full path.
- Close and reopen terminal.

### “api_key missing”

- `api_key_env` must be the variable name (for example `SPECTRA_API_KEY`)
- Actual key must be exported in terminal:

```bash
export SPECTRA_API_KEY="your-real-api-key"
```

### “not a git repository”

- Run Spectra from inside your project folder.
- Ensure `.git` exists.

### README did not update

- Run with `--auto`.
- Commit may be below threshold (`readme_threshold`).

---

## Files Spectra Creates/Uses

- `.spectra.yaml` → your Spectra settings
- `.git/hooks/post-commit` → optional auto-track hook
- `CHANGELOG.md` → tracked commit summaries
- `README.md` → docs + Spectra managed update block

---

## Managed Recent Changes

The block below is managed by Spectra and can be auto-updated:

<!-- spectra:readme:start -->

## Recent Changes

_Last updated: 2026-03-15_

- **README improved** — Added full beginner guide, Windows `.exe` distribution steps, and safe setup instructions.
<!-- spectra:readme:end -->
