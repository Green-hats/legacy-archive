# Anime RSS Automation / RSS 内容自动化采集工具

一个个人练习项目，用来把 RSS 订阅里的番剧条目整理成结构化数据，并通过 FastAPI 和前端页面进行浏览。项目也保留了可选的 PikPak 云盘扫描、直链补全和离线下载自动化逻辑。

> 说明：公开仓库不包含账号、密码、私有配置或运行时生成数据。需要本地运行时，请通过环境变量提供自己的 RSS 源和 PikPak 配置。

## 功能概览

- **RSS 采集与整理**：解析订阅源中的番剧名称、剧集、发布时间、磁力链接等信息。
- **数据增强**：尝试抓取条目封面，并输出 JSON 数据供前端展示。
- **FastAPI 接口**：提供 JSON 读取接口和调度器状态/手动触发接口。
- **前端展示**：使用原生 HTML/CSS/JavaScript 展示番剧列表和详情页。
- **可选 PikPak 自动化**：支持扫描云盘目录、匹配剧集、补全播放直链，并可按新增条目触发离线下载。
- **定时与去重**：通过状态文件记录已处理条目，减少重复下载或重复处理。

## 项目结构

```text
Auto_Anime_DL/
├── auto_anime_dl/
│   ├── config.py        # 环境变量与数据路径配置，不包含私密信息
│   ├── pikpak.py        # PikPak 扫描、匹配、离线下载封装
│   ├── rss.py           # RSS 解析与封面增强
│   ├── scheduler.py     # 定时抓取、去重、可选下载
│   ├── state.py         # 已处理条目状态读写
│   └── utils.py         # JSON / 文件 / 环境变量工具函数
├── frontend/            # 前端页面与样式
├── main.py              # FastAPI 服务入口
├── rssworker.py         # RSS 采集与封面增强入口
├── pikpak_dl.py         # 云盘扫描与播放直链补全入口
├── pikpak_automation.py # 可选离线下载自动化入口
├── pyproject.toml
└── test_main.http
```

## 数据流

```text
RSS 源
  -> rssworker.py
  -> bangumi.json
  -> bangumi_with_cover.json
  -> pikpak_dl.py 可选补全 play_url
  -> frontend/bangumi_with_play_url.json
  -> FastAPI + 前端页面展示
```

## 快速开始

### 1. 安装依赖

```bash
pip install -e .
```

也可以使用 `uv`：

```bash
uv sync
```

### 2. 配置环境变量

至少需要配置自己的 RSS 源。PikPak 相关配置只在使用云盘扫描或离线下载功能时需要。

```bash
export RSS_URL_ANIME_COLLECTION="https://api.animes.garden/collection/your-id/feed.xml"

# 可选：使用 PikPak 功能时配置
export PIKPAK_USERNAME="your_email@example.com"
export PIKPAK_PASSWORD="your_password"
export PIKPAK_ROOT_FOLDER_ID="your_root_folder_id"

# 可选：调度周期，默认 21600 秒，即 6 小时
export SCHEDULE_INTERVAL_SECONDS=21600
```

### 3. 运行采集脚本

```bash
python rssworker.py
```

如需扫描云盘并补全播放直链：

```bash
python pikpak_dl.py
```

### 4. 启动后端与前端

```bash
uvicorn main:app --reload
```

浏览器访问：

```text
http://127.0.0.1:8000/
```

## API

- `GET /api/bangumi`：读取 `bangumi.json`
- `GET /api/bangumi_with_cover`：读取 `bangumi_with_cover.json`
- `GET /api/bangumi_with_play_url`：读取 `frontend/bangumi_with_play_url.json`
- `GET /api/scheduler/status`：查看调度周期和已处理条目数量
- `GET|POST /api/scheduler/run_once`：手动执行一轮抓取与去重处理

## 运行时文件

以下文件会在本地运行时生成，默认不提交到公开仓库：

- `bangumi.json`
- `bangumi_with_cover.json`
- `frontend/bangumi_with_play_url.json`
- `pikpak_folder_structure.json`
- `.processed_state.json`
- `pikpak_automation.log`

## 安全说明

- 不要把账号、密码、Cookie、Token 或个人 RSS 源直接写入代码。
- 公开仓库只保留示例配置和项目代码，运行配置请通过环境变量传入。
- PikPak 自动化相关功能仅作为个人学习和流程练习使用。
