# Releasing Spectra

This file is for maintainers.

## Goal

Publish a new Windows executable so users can download it from GitHub Releases.

## 1) Make sure `main` is up to date

```bash
git checkout main
git pull origin main
```

## 2) Build the Windows binary

```bash
go build -o spectra-windows-amd64.exe .
```

Optional sanity check:

```bash
./spectra-windows-amd64.exe --help
```

## 3) Create and push a version tag

Use semantic versioning (`v0.1.0`, `v0.1.1`, `v0.2.0`, etc):

```bash
git tag v0.1.0
git push origin v0.1.0
```

## 4) Publish GitHub Release

1. Open repository on GitHub
2. Go to **Releases**
3. Click **Draft a new release**
4. Select the tag you pushed (e.g. `v0.1.0`)
5. Title it (example: `Spectra v0.1.0`)
6. Upload `spectra-windows-amd64.exe` under **Assets**
7. Publish release

## 5) What users do

Users can then download the `.exe` directly from the release asset page.

## Notes

- You do not need Cloudinary for executable distribution.
- Keep API keys out of repo files (`.spectra.yaml` should only contain env var names, never secret values).
