# Manual Validation

Use this checklist before calling a build "release candidate".

## Compose

1. Start the daemon and dashboard API locally.
2. Run `stateguard protect compose -f examples/windows-wsl2-compose/compose.yaml`.
3. Confirm bundle directories appear under the configured artifact root.
4. Run `stateguard intercept compose down -f examples/windows-wsl2-compose/compose.yaml`.
5. Run `stateguard restore artifact -id <generated-artifact-id>`.
6. Repeat with `runtime.compose.live_execution: true` for supported adapters.

## Kubernetes Beta

1. Run `stateguard kube inspect -f examples/kubernetes-beta/manifests.yaml`.
2. Run `stateguard kube protect -f examples/kubernetes-beta/manifests.yaml`.
3. Run `stateguard kube guard-delete -f examples/kubernetes-beta/manifests.yaml`.
4. Run `stateguard kube enforce-delete -f examples/kubernetes-beta/manifests.yaml`.
5. Confirm stateful resources generate artifacts and delete remains gated.

## Installers

1. Build release binaries into `dist/windows`, `dist/linux`, and `dist/macos`.
2. Run the validation wrappers first:
   - `install/validation/windows.ps1`
   - `install/validation/linux.sh`
   - `install/validation/macos.sh`
3. Run each platform installer on a disposable target.
4. Confirm:
   - binaries are copied
   - `safedata.yaml` is written
   - artifact directory exists
   - daemon/dashboard startup services are registered
   - safe Compose wrapper is created

## Dashboard/API

1. Open the dashboard root route.
2. Confirm status, adapters, artifacts, and scheduler cards render.
3. Use the daemon action panel to guard/protect a Compose file.
4. Confirm daemon endpoints respond:
   - `/api/v1/protect/compose`
   - `/api/v1/restore/artifact`
   - `/api/v1/intercept/compose`
   - `/api/v1/intercept/docker`
   - `/api/v1/protect/kube`
   - `/api/v1/enforce/kube-delete`

## Release

1. Run `go test ./...`.
2. Run `goreleaser check`.
3. Run `goreleaser release --snapshot --clean`.
4. Confirm archives and checksums are produced.
5. Validate installer docs, setup docs, and `install/validation` against generated artifacts.
