# AniVault

本地动画收藏库。手动关联视频文件夹与 [Bangumi](https://bgm.tv) 条目，自动匹配剧集并一键调起 mpv 播放。不依赖 Emby/Jellyfin/Plex。

## 功能

- **海报墙** — 首页网格展示所有番剧，支持过滤搜索
- **自动匹配** — 文件名提取集号，自动对应 Bangumi 剧集，支持自定义正则
- **元数据缓存** — Bangumi 剧集列表首次拉取后缓存，后续秒开
- **mpv 播放** — 点击本地文件服务端直接调起 mpv
- **管理页** — 搜索 Bangumi、浏览目录、增删关联、设置匹配规则
- **导入/导出** — JSON 格式备份恢复全部数据
- **白色极简风格**

## 安装

```bash
cd AniVault
python3 -m venv venv
venv/bin/pip install flask requests pytest
```

## 启动

```bash
./start.sh
```

首次运行会自动创建 venv 并安装依赖。访问 `http://127.0.0.1:5000`。

### 开机自启

`~/.config/autostart/anivault.desktop`:

```ini
[Desktop Entry]
Type=Application
Name=AniVault
Exec=/path/to/AniVault/start.sh
Hidden=false
X-GNOME-Autostart-enabled=true
```

## 使用流程

1. 打开 `/manage`，搜索番剧名，选择条目，浏览/输入文件夹，确认
2. 首页出海报，点击进详情
3. 详情页自动匹配剧集 → 点 🎬 调 mpv
4. 管理页可导出/导入 JSON 备份数据

### 匹配规则

默认支持：`S01E01` / `E01` / `Ep01` / `第01话` / `[01]` / `- 01 -` 等格式。匹配到超过 200 的数字（如年份）自动忽略。

**自定义正则**：在管理页每条映射下方点"修改"设置，或添加时填。详见 [REGEX.md](REGEX.md)。

### 测试

```bash
venv/bin/python -m pytest test_app.py -v
```

## 项目结构

```
├── app.py              # Flask 后端
├── test_app.py         # 测试
├── start.sh            # 启动脚本
├── requirements.txt
├── README.md
├── REGEX.md            # 正则说明
├── templates/
│   ├── base.html       # 公共布局
│   ├── index.html      # 首页海报墙
│   ├── show.html       # 详情 + 剧集匹配
│   └── manage.html     # 管理页
└── data/               # SQLite 数据库
```

## 依赖

- Python 3.8+
- Flask、requests
- mpv（播放视频）
- FFmpeg（可选，字幕提取）
