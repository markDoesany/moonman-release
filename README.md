# Moonman Release

Moonman Release is a Windows desktop workspace for streamlining the developer release workflow:

```text
Build → Validate → Package → Send via Dali → Log Release
```

Phase 4 adds local Dali CLI transfer after the Phase 3 packaging engine. Moonman Release invokes the installed Dali executable directly; Dali remains responsible for local-network peer discovery and file transfer.

## Technology

- Wails v2.12.0
- Go 1.26+
- Svelte + Vite + TypeScript
- YAML configuration using `gopkg.in/yaml.v3`
- Windows x64 primary target

## Windows distribution

Build a portable executable and ZIP with:

```powershell
.\scripts\build-windows.ps1 -PortableOnly
```

Build the per-user NSIS installer and portable ZIP with:

```powershell
.\scripts\build-windows.ps1
```

The installer build requires NSIS (`makensis.exe`) on `PATH`. Use `-PortableOnly` when NSIS is not installed.

The installer adds `moonman-release` to the current user's `PATH`, creates Moonman Release shortcuts, and stores writable application data in `%APPDATA%\MoonmanRelease`. Existing user data is kept when the application is uninstalled. The portable ZIP stores its data beside `moonman-release.exe`.

The installed GUI can be opened from Command Prompt or PowerShell with:

```powershell
moonman-release open
```

Running `moonman-release` without a command opens the same GUI. `moonman-release --help` displays the command usage.

## Requirements

- Windows 10 or Windows 11
- Go 1.26 or newer
- Node.js and npm
- Wails CLI v2.12.0
- WebView2 runtime for the built application; if it is missing, Moonman Release opens Microsoft's download guidance

Install the Wails CLI if needed:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
```

## Run in development

From the repository root:

```powershell
wails dev
```

The frontend can also be checked independently:

```powershell
cd frontend
npm install
npm run check
npm run build
npm run format:check
```

## Code organization and formatting

The backend is organized by responsibility under `backend/`: configuration and persistence, domain models, logging, build/package/transfer engines, and the Wails application boundary. The frontend keeps Wails calls in `src/backend.ts`, shared data contracts in `src/types.ts`, reusable UI in `src/components/`, and pure project helpers in `src/lib/`.

Run the formatter before committing frontend changes:

```powershell
cd frontend
npm run format
```

Use `npm run format:check` in CI or during review. Go files must remain `gofmt`-clean; the complete verification set is `go test ./...`, `go vet ./...`, `npm run format:check`, `npm run check`, and `npm run build`.

## Build the Windows executable

```powershell
wails build -platform windows/amd64 -o moonman-release.exe
```

The executable is written to `build/bin/` by Wails unless an alternate output is specified. Installed users can launch the GUI with `moonman-release open`.

## Configuration

For development and portable builds, the editable configuration is stored at:

```text
configs/projects.yaml
```

Installed builds store the same files under `%APPDATA%\MoonmanRelease\configs` and `%APPDATA%\MoonmanRelease\logs`. The application starts with no projects when the configuration is missing. A previous validated version is retained as `projects.yaml.bak` after saves. Application events are appended to `logs/app.log`; build, packaging, and Dali transfer attempts are appended as JSONL records to `logs/builds.jsonl`, `logs/packaging.jsonl`, and `logs/transfers.jsonl`.

Set `RELEASE_LAUNCHER_DATA_DIR` to use another configuration/log directory, for example during testing:

```powershell
$env:RELEASE_LAUNCHER_DATA_DIR = 'C:\ReleaseLauncherData'
```

Example format:

```yaml
projects:
  - id: lokalstore
    name: LokalStore
    components:
      - id: admin
        name: Admin
        path: 'C:\Projects\LokalStore\Admin'
        build_command: npm run build
        output_directory: build
        package:
          enabled: true
          filename: lokalstore-admin.zip
```

Project and component IDs are generated for new records and remain stable when names are edited. Component paths must exist when a build starts. Build commands run from the component path using Windows `cmd.exe /d /s /c`, so commands such as `npm run build` work with the developer's normal PATH. Existing Phase 1 flat package fields are accepted and upgraded to the nested format when saved.

Packages are written to the app data `releases` folder by default (or the portable folder's `releases` directory):

```text
releases/<Project Name>/<Version>/<package filename>
```

The default UI version is `1.0.0`; leaving the version blank uses a timestamp. Existing ZIP files require explicit replacement confirmation. `Package Existing Build` packages a validated output directory without running the build command again.

Dali settings are stored in the same YAML file:

```yaml
dali:
  executable: dali
  peer_name: DevOps
  peer_address: ''
  auto: true
  wait: false
```

Configure either a peer name, a peer address such as `192.168.1.20:45679`, or automatic selection of a single discovered peer. The default executable is `dali`, which must be available in `PATH`; a full path to `dali.exe` is also supported. Moonman Release runs `dali send file=...` directly and marks a transfer successful only when Dali prints its success confirmation.

## Current behavior

- Projects and components can be added, edited, and deleted.
- Changes are validated before being saved.
- Selected components build sequentially in YAML order.
- Stdout and stderr stream into the build console while the process runs.
- A successful command must produce its configured output directory.
- A failed build stops later components; active builds can be cancelled.
- Build & Package runs each selected component through build, output validation, ZIP creation, and ZIP validation sequentially.
- Send Packages transfers existing ZIP files sequentially through Dali.
- Build, Package & Send runs the complete local release workflow sequentially.
- No Git integration, database, authentication, cloud service, or remote API is included.

## Architecture

```text
Wails bindings: backend/app
Configuration:  backend/config
Models:         backend/models
Logging:        backend/logging
Build engine:   backend/build
Packaging:      backend/packaging
Pipeline:       backend/pipeline
Transfer:       backend/transfer
Frontend:       frontend/src
Data:           configs/ and logs/
```

## Build engine testing

Run the automated checks from the repository root:

```powershell
go test ./...
go vet ./...
cd frontend
npm run check
npm run build
```

Build the Windows executable with:

```powershell
wails build -platform windows/amd64 -o moonman-release.exe
```

The sample configuration uses placeholder paths. To exercise a real build, update a component path and command in Project Settings, then select the component and click Build Only, Build & Package, or Build, Package & Send. Package-only testing uses an existing configured output directory. To test Dali, run `dali open accept=auto` on a receiving machine, configure its peer name/address in Project Settings, then use Send Packages or Build, Package & Send.
