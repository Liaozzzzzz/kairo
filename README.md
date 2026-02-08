# Kairo

[中文](README_zh-CN.md)

Kairo is a modern cross-platform video downloader built with [Wails](https://wails.io/). It combines the high performance of Go with the flexibility of React, leveraging the powerful [yt-dlp](https://github.com/yt-dlp/yt-dlp) under the hood to support video downloads from platforms like YouTube, Bilibili, and many others.

## ✨ Features

- **Cross-Platform Support**: Supports Windows, macOS, and Linux.
- **Modern UI**: Clean interface built with React and Tailwind CSS, providing a smooth user experience.
- **Powerful Downloading Capabilities**: Supports hundreds of video sites including YouTube and Bilibili (powered by yt-dlp).
- **Task Management**: Clear task list supporting download progress viewing, logs, and history.
- **Advanced Configuration**:
  - **Cookie Support**: Import Cookies from browsers or local files to easily download member-only or age-restricted content.
  - **Site-Specific Configuration**: Set authentication methods separately for different sites (e.g., Bilibili, YouTube).
- **Internationalization**: Built-in support for English and Chinese.

## 🧩 Function Details

### 🎥 Video Download

- **Smart Parsing**: Automatically parses video links to retrieve titles, cover previews, and available quality options (e.g., 4K, 1080P, 720P).
- **Format Selection**: Supports selecting "Best Quality", specific resolutions (e.g., 4K, 1080P), or audio-only extraction.
- **Format Conversion**: Supports merging/converting videos to common formats like MP4, MKV, AVI, WEBM, FLV, MOV, etc.
- **Custom Path**: Supports setting a global default download path, or manually specifying the save location for each download.

### 📋 Task Management

- **Real-time Progress**: Visually displays download progress bars, current download speed, and estimated remaining time.
- **Status Filtering**: Filter the task list by statuses like "Downloading", "Completed", or "All".
- **Log Diagnostics**: Built-in log viewer to view underlying `yt-dlp` output logs in real-time, facilitating troubleshooting of download failures.

### ⚙️ Advanced Settings

- **Network Configuration**:
  - **HTTP Proxy**: Supports setting a global HTTP proxy to easily access overseas video sites like YouTube.
  - **Speed Limit**: Supports setting maximum download speed to avoid using too much bandwidth.
  - **Concurrency Control**: Adjust the number of simultaneous download tasks.
- **Cookie Authentication (Member/Restricted Content)**:
  - **Browser Auto-Read**: Directly read logged-in Cookies from browsers like Chrome, Edge, Firefox, Opera, Safari, Brave, Vivaldi, and Chromium (no manual extraction needed).
  - **Netscape Format File**: Import standard Netscape format Cookies files (e.g., exported via EditThisCookie plugin).
  - **Site Isolation**: Authentication configurations for Bilibili and YouTube are independent and do not interfere with each other.

## 🛠 Tech Stack

- **Core**: [Wails](https://wails.io/) (Go + Webview)
- **Frontend**: React, TypeScript, Tailwind CSS, Zustand, Headless UI, i18next
- **Backend**: Go Standard Library
- **Downloader**: yt-dlp, ffmpeg

## 🚀 Quick Start

### Prerequisites

- [Go](https://go.dev/) 1.18+
- [Node.js](https://nodejs.org/) (Recommended to use [pnpm](https://pnpm.io/))
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

Install Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Development

1. Clone the project locally.

2. Install frontend dependencies:

```bash
cd frontend
pnpm install
# Or npm install
cd ..
```

3. Start development mode:

```bash
wails dev
```

On the first run, the project will automatically download the required `yt-dlp` and `ffmpeg` binaries via `scripts/init_binaries.go`.

**Shared Mode (Recommended for reducing binary size):**

If you want the built application to not include embedded binaries (yt-dlp/ffmpeg) and instead download them automatically on the first run, you can use the `shared` tag:

```bash
# Development mode
wails dev -tags shared

# Build production version
wails build -tags shared
```

In `shared` mode:
1. The `wails build` phase will skip the download process in `scripts/init_binaries.go`, significantly speeding up the build.
2. When the application starts, if it finds missing binaries in the configuration directory, it will automatically download them from the network.

### Build

Build production version:

```bash
wails build
```

The built executable will be located in the `build/bin` directory.

## ⚙️ Project Structure

```
.
├── build/              # Build artifacts and resources
├── frontend/           # React frontend code
│   ├── src/
│   │   ├── components/ # Common components
│   │   ├── views/      # Page views (Downloads, Tasks, Settings)
│   │   ├── store/      # Zustand state management
│   │   └── ...
├── internal/           # Go backend logic
│   ├── downloader/     # Core download logic
│   ├── task/           # Task management
│   └── ...
├── scripts/            # Helper scripts (e.g., binary download)
└── app.go              # Wails application entry
```

## 📝 License

[MIT](LICENSE)
