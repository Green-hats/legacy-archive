# ani-rss-lite

基于 RSS 自动追番、订阅、下载、刮削、通知的开源工具 **ani-rss** 的**精简版**。
后端以 Go 重写(替代上游 Java),前端按"**网盘下载 + 外置播放器播放**"重新定制,只保留真正用到的功能。

> **背景**:硬盘越来越贵,追番囤货的成本越来越高,于是转向网盘——把番剧直接离线下载到 115 网盘,不再占本地硬盘空间,播放走外置播放器直连网盘。
>
> **定位**:去掉不适合网盘追番场景的部分(本地 BT 客户端、在线播放、合集、捐赠等),专注 **115 云端离线下载 + 外置播放器直连播放**。

## 特性

- **下载**:115 云端离线下载(浏览器 Cookie 登录),自动重试、延迟下载、失败重试、备用 RSS 多源
- **播放**:全部走外置播放器跳转(PotPlayer / VLC / IINA / MPV / Infuse / 弹弹Play / AnimacX / SenPlayer),无在线播放器;云端文件经后端流式反代,支持 Range/seek
- **核心**:RSS 抓取、剧集提取、重命名模板、缺集/摸鱼检测与通知、备用 RSS
- **刮削**:TMDB(NFO/poster/fanart)、Bangumi 搜索/评分/OAuth、Mikan / ani-bt / anime-garden 番剧源
- **通知**:Telegram / Bark / ServerChan / WebHook / Shell / 邮件 / Emby 媒体库刷新
- **安全**:登录鉴权、IP 白名单、反向代理信任列表

## 快速开始

### 环境要求

- Go 1.26+
- Node.js + pnpm(仅从源码构建前端时需要)
- 外置播放器(MPV 用户另需 `mpv-handler`)

### 构建

```bash
make all   # 前端 + 后端一体构建,产物在 bin/ani-rss
```

或直接下载预编译二进制(待发布)。

### 运行

```bash
./ani-rss                        # 端口 7789,配置目录 ./config(自动创建)
PORT=7789 ./ani-rss              # 自定义端口
CONFIG=/path/to/config ./ani-rss # 自定义配置目录
```

首次启动自动生成 `config/` 下的 `config.v2.json`、`ani.v2.json`、`logs/`。
默认登录:**admin / admin**(密码以 MD5 存储)。

> 已有上游 ani-rss 的配置目录,直接指向即可无缝迁移(配置与订阅文件格式一致)。

### 配置 115 下载

1. 浏览器登录 [115 网盘](https://115.com),复制 Cookie(`UID=...; CID=...; SEID=...; KID=...`)
2. 打开 Web UI → 设置 → 下载,填入 Cookie 并点击"测试"
3. 测试通过后即开始云端离线下载,文件落盘在 115 网盘

### 外置播放器

播放按钮通过各播放器自定义协议唤起(如 `potplayer://`、`vlc://`、`iina://`)。MPV 用户需安装 [akiirui/mpv-handler](https://github.com/akiirui/mpv-handler):

```bash
chmod +x ~/.local/bin/mpv-handler
xdg-mime default mpv-handler.desktop x-scheme-handler/mpv-handler
update-desktop-database ~/.local/share/applications/
```

- 将 `scripts/mpv-title-wrapper` 放入 `~/.local/bin`,并让 handler 指向它(`~/.config/mpv-handler/config.toml` 中 `mpv = ".../mpv-title-wrapper"`),使 OSD/窗口标题显示文件名
- Firefox 需在 `about:config` 新建布尔 `network.protocol-handler.expose.mpv-handler = false`,再允许一次协议唤起

## 开发

```bash
make dev   # 后端 + Vite 热更新(前端 /api 自动代理到后端)
```

- Go 后端:`http://127.0.0.1:7789`
- Vite dev server:`http://127.0.0.1:37789`
- 改前端后重新构建嵌入:`make ui && make build`

## 构建命令

```bash
make all     # 前端 + 后端一体构建(推荐)
make ui      # 仅构建前端并拷入 internal/server/webui
make build   # 仅构建后端(使用已嵌入的 webui)
make run     # 直接运行
make dev     # 开发模式
make test    # go vet + go test
make clean   # 清理 bin / ui/dist / ui/node_modules
```

## 项目结构

```
cmd/ani-rss/        入口
ui/                 前端源码(Vue 3 + Vite,与后端同一仓库)
scripts/            开发脚本、mpv-title-wrapper
internal/
  model/            数据模型(JSON 契约与上游一致)
  config/           配置加载/持久化/部分合并
  store/            JSON 原子读写
  cache/            TTL 内存缓存
  auth/             鉴权 token、IP 白名单
  server/           HTTP 路由、处理器、静态资源(嵌入 webui)
  service/          核心服务(下载引擎、订阅管理、刮削调度)
  task/             后台任务循环(rss/bgm)
  download/         下载客户端(115 正式,PikPak 开发中)
  rss/               RSS 抓取与解析
  rename/           剧集提取与重命名模板
  bgm/               Bangumi API 客户端
  tmdb/              TMDB API 客户端
  mikan/             Mikan 网页抓取
  anibt/             ani-bt 抓取
  animegarden/       anime-garden 抓取
  groupregex/        字幕组过滤规则生成
  scrape/            TMDB 刮削(NFO 生成)
  notify/            通知发送器
  util/              工具
```

## 与原版 ani-rss 的差异

- **后端 Go 重写**,替代 Java Spring Boot,API 契约兼容、配置格式一致,可无缝迁移
- **下载器只保留网盘**:前端仅暴露 115(云端离线下载);后端另有 PikPak(开发中)
- **移除在线播放器**(浏览器无法解码 MKV/HEVC),播放全部改为外置播放器跳转
- **移除合集 / 捐赠 / QQ 机器人通知 / 自动更新**
- **移除本地 BT 客户端概念**:qBittorrent 下载器、trackers 更新、标签、做种等待、下载任务列表、本地文件迁移/上传(OpenList/FileMove)等
- **云端播放**:`/api/file` 反代 115 CDN 流,支持 Range/seek

## 常见问题

- **刷新 RSS 后"共 0 个条目"**:多为排除规则 `\d-\d` 误伤标题中的编码格式(如 `H265-10bit` 里的 `5-1`),请在订阅的排除规则中删除该条;匹配规则中 `h265` 建议写成 `H265|HEVC|AV1`
- **AnimeGarden 订阅拉取 400**:字幕组名含中文时需正确编码,新版本已自动处理
- **预览灰屏**:空 RSS 时后端已保证返回空数组,若仍异常请确认使用最新构建

## 免责声明

本工具为中立性技术辅助工具,请遵守当地法律法规,勿用于盗版传播。

## License

本项目为 [ani-rss](https://github.com/wushuo894/ani-rss) 的精简版本,相关版权与开源许可遵循上游项目。