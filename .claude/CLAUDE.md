# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kairo is a cross-platform video downloader desktop application built with Wails (Go + React). It uses yt-dlp for downloading videos from YouTube, Bilibili, and hundreds of other platforms. The application supports RSS feeds for automatic video downloads, AI-powered video analysis/highlighting, and automated publishing to platforms like Douyin.

## Commands

### Development
```bash
# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Start development mode with hot reload
wails dev

# Development mode without embedded binaries (smaller builds)
wails dev -tags shared
```

### Build
```bash
# Build production executable
wails build

# Build without embedded binaries
wails build -tags shared
```

### Frontend
```bash
cd frontend
pnpm run lint        # Run ESLint
pnpm run lint:fix    # Fix lint issues
pnpm run format      # Format with Prettier
```

### Go
```bash
go test ./...        # Run all tests
go test ./internal/task/...  # Run tests for specific package
```

## Architecture

### Backend (Go)

The Go backend follows a manager-based architecture where each domain has a dedicated Manager struct that handles business logic:

- **`app.go`** - Main Wails application binding. All methods exposed to frontend are defined here. Initializes all managers on startup.
- **`internal/task/`** - Download task management. `Manager` handles task lifecycle, `runner.go` executes yt-dlp commands and parses output.
- **`internal/video/`** - Video library management, subtitles, and AI analysis/highlights.
- **`internal/rss/`** - RSS feed management with auto-download capability.
- **`internal/publish/`** - Publishing automation to social platforms (Douyin, Xiaohongshu).
- **`internal/ai/`** - AI integration for video analysis and subtitle processing.
- **`internal/deps/`** - Manages external binaries (yt-dlp, ffmpeg).
- **`internal/db/`** - Database layer with GORM. `schema/` contains model definitions, `dal/` contains data access layers.
- **`internal/config/`** - Application settings stored in `~/Library/Application Support/Kairo/config.json` (macOS).

### Frontend (React + TypeScript)

- **`frontend/src/views/`** - Page-level components (downloads, tasks, rss, settings, videos, publish, categories).
- **`frontend/src/store/`** - Zustand state management stores. Each domain has its own store.
- **`frontend/src/components/`** - Reusable UI components.
- **`frontend/src/App.tsx`** - Main app component with event listeners setup.

### Backend-Frontend Communication

Wails provides two-way communication:
1. **Method calls** - Frontend calls Go methods via generated bindings in `wailsjs/go/main/App.js`
2. **Events** - Backend emits events using `wailsRuntime.EventsEmit()`, frontend listens via `EventsOn()`

Key events:
- `task:update` - Task state changes
- `task:progress` - Download progress updates
- `task:log` - yt-dlp output logs
- `video:ai_status` - AI analysis status updates

### Database

Uses GORM with SQLite by default (configurable to MySQL/PostgreSQL). Models are defined in `internal/db/schema/`. Each model has a corresponding DAL in `internal/db/dal/`.

## Key Patterns

### Adding a new feature
1. Define schema in `internal/db/schema/`
2. Create DAL in `internal/db/dal/`
3. Create/update Manager in appropriate `internal/` package
4. Add methods to `app.go` for frontend exposure
5. Run `wails generate module` to update bindings (if using Go methods)
6. Create/update frontend store in `frontend/src/store/`
7. Create/update view in `frontend/src/views/`

### Task execution flow
1. Frontend calls `AddTask` → `task.Manager.AddTask()`
2. Task created with `pending` status, saved to DB
3. `startTask()` spawns goroutine running yt-dlp
4. Progress parsed from yt-dlp stdout, emitted via events
5. On completion, `OnTaskComplete` callback fires (wired in `app.go` startup)

### External binaries
- yt-dlp and ffmpeg are auto-downloaded on first run
- In `shared` build mode, binaries are downloaded at runtime instead of embedded
- Paths resolved via `internal/deps/manager.go`

## Configuration

App settings are persisted to disk via `internal/config/`. Key settings include:
- Download directory and concurrency
- Proxy settings
- Cookie configuration for member content
- AI provider settings (OpenAI, Anthropic, etc.)
- Database configuration