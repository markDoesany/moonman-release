# Release Launcher

Release Launcher is a Windows desktop foundation for streamlining the developer release workflow:

```text
Build → Validate → Package → Send via Dali → Log Release
```

Phase 2 adds the local build engine. Packaging and Dali sending remain intentionally reserved for later phases.

## Technology

- Wails v2.12.0
- Go 1.26+
- Svelte + Vite + TypeScript
- YAML configuration using `gopkg.in/yaml.v3`
- Windows x64 primary target

## Requirements

- Windows 10 or Windows 11
- Go 1.26 or newer
- Node.js and npm
- Wails CLI v2.12.0
- WebView2 runtime for the built application

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
```

## Build the Windows executable

```powershell
wails build -platform windows/amd64 -o release-launcher.exe
```

The executable is written to `build/bin/` by Wails unless an alternate output is specified.

## Configuration

The editable configuration is stored at:

```text
configs/projects.yaml
```

The application creates the LokalStore sample configuration if the file is missing. A previous validated version is retained at `configs/projects.yaml.bak` after saves. Application events are appended to `logs/app.log`, while build attempts are appended as one JSON record per line to `logs/builds.jsonl`.

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
        package_enabled: true
        package_filename: lokalstore-admin.zip
```

Project and component IDs are generated for new records and remain stable when names are edited. Component paths must exist when a build starts. Build commands run from the component path using Windows `cmd.exe /d /s /c`, so commands such as `npm run build` work with the developer's normal PATH.

## Current behavior

- Projects and components can be added, edited, and deleted.
- Changes are validated before being saved.
- Selected components build sequentially in YAML order.
- Stdout and stderr stream into the build console while the process runs.
- A successful command must produce its configured output directory.
- A failed build stops later components; active builds can be cancelled.
- No ZIP creation, Dali transfer, Git integration, database, authentication, cloud service, or remote API is included.

## Architecture

```text
Wails bindings: backend/app
Configuration:  backend/config
Models:         backend/models
Logging:        backend/logging
Build engine:   backend/build
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
wails build -platform windows/amd64 -o release-launcher.exe
```

The sample configuration uses placeholder paths. To exercise a real build, update a component path and command in Project Settings, then select the component and click Build Selected.
