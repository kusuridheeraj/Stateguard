# Packaging

This directory holds native packaging assets for Stateguard.

## Intended Distribution Channels

- Windows: `winget` and installer bundle
- macOS: `brew` and installer bundle
- Linux: package-manager friendly release artifacts plus direct installer

## Current State

The installer scripts under [install](../install/README.md) are now executable and perform real host setup. The manifests in this directory remain the packaging side of that work:

- `nfpm.yaml` tracks Linux package metadata and expected install layout
- `homebrew/stateguard.rb` tracks Homebrew packaging intent
- `winget/stateguard.yaml` tracks Windows package metadata

These files still need release-versioned checksums and distribution URLs before public publishing, but they are aligned with the current installer contract rather than placeholder-only scaffolds.

Release validation now has two layers:

- `install/validation/*` scripts exercise the installers in `validate-only` mode without mutating host services
- `.github/workflows/install-validation.yml` and the release workflow validate the packaging contract before publication
