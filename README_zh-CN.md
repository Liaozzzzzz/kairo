# Kairo

[English](README.md)

Kairo 是一个基于 [Wails](https://wails.io/) 构建的现代化跨平台视频下载器。它结合了 Go 的高性能后端与 React 的灵活性，底层利用强大的 [yt-dlp](https://github.com/yt-dlp/yt-dlp) 来支持 YouTube、Bilibili 等多种平台的视频下载。

## ✨ 特性

- **跨平台支持**：支持 Windows、macOS 和 Linux。
- **现代化 UI**：基于 React 和 Tailwind CSS 构建的清爽界面，提供流畅的用户体验。
- **强大的下载能力**：支持 YouTube、Bilibili 等数百个视频网站（基于 yt-dlp）。
- **任务管理**：清晰的任务列表，支持查看下载进度、日志和历史记录。
- **高级配置**：
  - **Cookie 支持**：支持从浏览器或本地文件导入 Cookie，轻松下载会员或年龄限制内容。
  - **站点独立配置**：可针对不同站点（如 Bilibili、YouTube）单独设置认证方式。
- **国际化**：内置中英文多语言支持。

## 🧩 功能详情

### 🎥 视频下载

- **智能解析**：自动解析视频链接，获取标题、封面预览以及可用的画质选项（如 4K, 1080P, 720P）。
- **格式选择**：支持选择“最佳画质”、指定特定分辨率（如 4K, 1080P）或仅提取音频。
- **格式转换**：支持将视频合并/转换为 MP4, MKV, AVI, WEBM, FLV, MOV 等常见格式。
- **自定义路径**：支持设置全局默认下载路径，也可在每次下载时手动指定保存位置。

### 📋 任务管理

- **实时进度**：直观展示下载进度条、当前下载速度和预计剩余时间。
- **状态过滤**：支持按“下载中”、“已完成”或“全部”筛选任务列表。
- **日志诊断**：内置日志查看器，可实时查看底层 `yt-dlp` 的输出日志，方便排查下载失败原因。

### ⚙️ 高级设置

- **网络配置**：
  - **HTTP 代理**：支持设置全局 HTTP 代理，轻松访问 YouTube 等海外视频站点。
  - **速度限制**：支持设置最大下载速度，避免占用过多带宽。
  - **并发控制**：可调整同时下载的任务数量。
- **Cookie 认证（会员/限制内容）**：
  - **浏览器自动读取**：支持直接从 Chrome、Edge、Firefox、Opera、Safari、Brave、Vivaldi、Chromium 等浏览器中读取已登录的 Cookie（无需手动提取）。
  - **Netscape 格式文件**：支持导入标准的 Netscape 格式 Cookies 文件（如通过 EditThisCookie 插件导出）。
  - **站点隔离**：Bilibili 和 YouTube 的认证配置相互独立，互不干扰。

## 🛠 技术栈

- **Core**: [Wails](https://wails.io/) (Go + Webview)
- **Frontend**: React, TypeScript, Tailwind CSS, Zustand, Headless UI, i18next
- **Backend**: Go Standard Library
- **Downloader**: yt-dlp, ffmpeg

## 🚀 快速开始

### 前置要求

- [Go](https://go.dev/) 1.18+
- [Node.js](https://nodejs.org/) (推荐使用 [pnpm](https://pnpm.io/))
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

安装 Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 开发

1. 克隆项目到本地。

2. 安装前端依赖：

```bash
cd frontend
pnpm install
# 或者 npm install
cd ..
```

3. 启动开发模式：

```bash
wails dev
```

首次运行时，项目会自动通过 `scripts/init_binaries.go` 下载所需的 `yt-dlp` 和 `ffmpeg` 二进制文件。

**Shared 模式（推荐用于减小二进制体积）：**

如果您希望构建出的应用不包含内嵌的二进制文件（yt-dlp/ffmpeg），而是让应用在首次运行时自动下载，可以使用 `shared` 标签：

```bash
# 开发模式
wails dev -tags shared

# 构建生产版本
wails build -tags shared
```

在 `shared` 模式下：
1. `wails build` 阶段会跳过 `scripts/init_binaries.go` 的下载过程，显著加快构建速度。
2. 应用启动时，如果发现配置目录下缺少二进制文件，会自动从网络下载。

### 构建

构建生产环境版本：

```bash
wails build
```

构建后的可执行文件将位于 `build/bin` 目录中。

## ⚙️ 项目结构

```
.
├── build/              # 构建产物和资源
├── frontend/           # React 前端代码
│   ├── src/
│   │   ├── components/ # 通用组件
│   │   ├── views/      # 页面视图 (Downloads, Tasks, Settings)
│   │   ├── store/      # Zustand 状态管理
│   │   └── ...
├── internal/           # Go 后端逻辑
│   ├── downloader/     # 下载核心逻辑
│   ├── task/           # 任务管理
│   └── ...
├── scripts/            # 辅助脚本 (如二进制下载)
└── app.go              # Wails 应用入口
```

## 📝 License

[MIT](LICENSE)
