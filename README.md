# Release Launcher

Release Launcher is a Windows desktop foundation for streamlining the developer release workflow:

```text
Build → Validate → Package → Send via Dali → Log Release
```

Phase 1 provides the project/component configuration system and desktop UI. Release execution is intentionally reserved for Phase 2.

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

The application creates the LokalStore sample configuration if the file is missing. A previous validated version is retained at `configs/projects.yaml.bak` after saves. Application events are appended to `logs/app.log`.

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

Project and component IDs are generated for new records and remain stable when names are edited. Placeholder paths are accepted in Phase 1; the application does not execute build commands yet.

## Phase 1 behavior

- Projects and components can be added, edited, and deleted.
- Changes are validated before being saved.
- Build buttons are intentionally disabled and marked as Phase 2 functionality.
- No build execution, ZIP creation, Dali transfer, Git integration, database, authentication, cloud service, or remote API is included.

## Architecture

```text
Wails bindings: backend/app
Configuration:  backend/config
Models:         backend/models
Logging:        backend/logging
Frontend:       frontend/src
Data:           configs/ and logs/
```

Phase 2 can add a release execution service alongside the existing configuration service without moving the Wails bindings or changing the YAML model.
