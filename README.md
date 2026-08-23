# ANI-RSS (Go 二创版)

基于 RSS 自动追番、订阅、下载、刮削、洗版工具 **ani-rss** 的 **Go 重写 + 前端二创版**。
后端用 Go 替代上游 Java(Spring Boot),前端基于上游 `ani-rss-ui` 按网盘下载 + 外置播放器的需求深度定制。

> 这是 **二创项目**:保留上游核心功能(追番/下载/刮削/通知),砍掉不适合我们使用方式的部分,并按"**只走网盘下载 + 全部外置播放器播放**"重新设计。

## 设计思路

- **Go 重写后端**:替代 Java 后端,API 契约兼容,配置/订阅文件格式一致(`config.v2.json`/`ani.v2.json`),可无缝迁移。
- **下载器只保留网盘**:不做本地 BT 客户端对接,专注**云端离线下载**;目前提供 **115**(Cookie 登录,离线下载到 115 网盘后走云盘直链/反代播放)。
- **删除在线播放**:浏览器解不了 MKV/HEVC,在线播放既卡又不实用 → 全部改为**外置播放器跳转**。
- **mpv-handler 唤起 MPV**:MPV 按钮经 `mpv-handler://` 协议拉起本机 mpv 播放,窗口标题与 OSD 均显示文件名。

## 特性

- **下载**:115 云端离线下载(浏览器 Cookie),自动重试、延迟下载、失败重试、同时下载限制、优先保留、自定义标签
- **播放**:播放弹窗仅保留外置播放器按钮(PotPlayer / VLC / IINA / **MPV** / Infuse / 弹弹Play / AnimacX / SenPlayer),无在线播放器
  - 云端文件经 `/api/file` 反代:后端生成 115 直链并**流式透传**(支持 Range/seek、正确 Content-Type),播放器只与本服务通信
  - **MPV**:通过 `mpv-handler://` 协议唤起(见下方安装),OSD/窗口标题显示文件名
- **核心**:RSS 抓取、剧集提取、重命名模板、缺集/摸鱼通知、备用RSS洗版
- **刮削**:TMDB 刮削(NFO/poster/fanart/bangumi.ini)、Bangumi 搜索/评分/OAuth、Mikan / ani-bt / anime-garden 抓取
- **通知**:Telegram / Bark / ServerChan / WebHook / Shell / 邮件 / Emby 刷新
- MKV **内封字幕流式提取**(`matroska`,只读所需字节,不整读大文件)

## 快速开始(前置依赖)

**播放需要外置播放器 + 协议 handler**,MPV 用户请先安装:

1. 安装 [akiirui/mpv-handler](https://github.com/akiirui/mpv-handler)(Linux 手册安装:把 `mpv-handler` 放到 `~/.local/bin`,两个 `.desktop` 放到 `~/.local/share/applications/`,然后):
   ```bash
   chmod +x ~/.local/bin/mpv-handler
   xdg-mime default mpv-handler.desktop x-scheme-handler/mpv-handler
   xdg-mime default mpv-handler-debug.desktop x-scheme-handler/mpv-handler-debug
   update-desktop-database ~/.local/share/applications/
   ```
2. 装好 `mpv-title-wrapper`(让 OSD/窗口标题显示文件名,附本仓库 `scripts/` 或手动创建,并把 handler 指向它):
   ```bash
   # ~/.config/mpv-handler/config.toml
   mpv = "/home/<user>/.local/bin/mpv-title-wrapper"
   ```
3. Firefox:打开 `about:config`,新建布尔值 **`network.protocol-handler.expose.mpv-handler` = false**,使浏览器把 `mpv-handler://` 交给系统。
4. 点 MPV 按钮,Firefox 弹"打开 mpv-handler?"→ 允许并勾选记住。

> PotPlayer/VLC/IINA/Infuse 等直接经各自协议唤起,对应播放器需已安装。

## 前端源码管理

前端源码在本仓库 **`ui/`** 目录,与后端同一 git 仓库。改前端一律在 `ui/src` 下修改后重新构建,产物拷入 `internal/server/webui` 再编译嵌入,前后端一起发布。

### 二创定制点

- `src/play/Artplayer.vue`:移除在线播放器,改为外置播放器按钮(真实 `<a href>`);MPV 用 `mpv-handler://` 协议,URL-safe base64 传地址与文件名。
- `src/play/PlayStart.vue`:移除 `getSubtitles`/loading,不再有任何在线播放动作。
- `src/config/Download.vue`:下载器仅 **115** + 115 Cookie 输入;移除下载地址栏及 qBittorrent/Aria2/OpenList 等死分支。
- `src/home/Config.vue`:移除**捐赠** tab。
- **移除合集(Collection)与捐赠残留**相关组件/接口。
- 前端不再调用 `getSubtitles`(后端 `/api/getSubtitles` 与 `matroska` 包保留,API 兼容)。
- 配置与订阅持久化:`config.v2.json` / `ani.v2.json`(与原版格式一致,可无缝迁移)
- 前端:编译时嵌入 `internal/server/webui`,SPA 回退;`<configDir>/webui` 可覆盖单个文件

## 目录结构

```
cmd/ani-rss/         入口
ui/                  前端源码(Vue,Vite,与后端同一 git 仓库)
scripts/dev.sh       开发模式脚本
internal/
  model/             数据模型(与 Java 实体 JSON 契约一致)
  config/            配置加载/持久化/部分合并
  store/             JSON 原子读写
  cache/             TTL 缓存 + 文件持久缓存(rename-cache)
  auth/              鉴权 token、IP 白名单
  server/            HTTP 路由、处理器、静态资源(嵌入 webui)
  service/           核心服务(下载引擎、订阅管理、备份、刮削调度)
  task/              后台任务循环(rss/rename/bgm)
  download/          qBittorrent / Transmission / Aria2 / OpenList / 115 / PikPak(后端保留,前端仅 115)
  rss/               RSS 抓取与解析
  rename/            剧集提取与重命名模板
  bgm/               Bangumi API 客户端
  tmdb/              TMDB API 客户端
  mikan/             Mikan 网页抓取
  anibt/             ani-bt 抓取
  animegarden/       anime-garden 抓取
  groupregex/        字幕组过滤规则生成
  scrape/            TMDB 刮削(NFO 生成)
  notify/            通知发送器
  matroska/          MKV 内封字幕流式提取
  util/              工具
```

## 构建

前端(`ui/`)与后端(Go)一体构建,统一由 Makefile 管理:

```bash
make all     # 前端 + 后端一次构建(推荐)
make ui      # 仅构建前端并拷入 internal/server/webui
make build   # 仅构建后端(使用已嵌入的 webui)
make run     # 直接运行
make dev     # 开发模式:后端 + Vite 热更新
make clean   # 清理 bin / ui/dist / ui/node_modules
make test    # go vet + go test
```

`make all` 内部流程:
```bash
cd ui && pnpm install && pnpm build        # 构建 Vue 前端
find ui/dist -name '*.gz' -delete          # 去掉 gzip 冗余
cp -r ui/dist/. internal/server/webui/     # 产物拷入嵌入目录
go build -o bin/ani-rss ./cmd/ani-rss      # 编译后端(webui 一并嵌入)
```

### 开发模式

```bash
make dev
```
- Go 后端运行在 `http://127.0.0.1:7789`
- Vite dev server 运行在 `http://127.0.0.1:37789`,`/api` 自动代理到后端,改前端即时热更新
- `Ctrl-C` 同时停掉两者;也可分两个终端分别 `make run` 和 `cd ui && pnpm dev`

## 运行

```bash
./ani-rss                       # 端口 7789,配置目录 ./config(自动创建)
PORT=7789 ./ani-rss             # 自定义端口
CONFIG=/path/to/config ./ani-rss # 自定义配置目录
```

首次启动会自动生成 `config/` 下的 `config.v2.json`、`ani.v2.json`、`logs/`。
若已有原版 ani-rss 的配置目录,直接指向它即可无缝迁移。

默认登录:admin / admin(MD5 存储于配置)。

## 与原版(ani-rss)的差异

- **后端 Go 重写**(替代 Java Spring Boot),API 兼容,可无缝迁移配置
- **下载器只留网盘**:前端仅 115(云端离线下载);qBittorrent/Transmission/Aria2/OpenList/PikPak 后端代码保留但前端不暴露
- **移除在线播放器**(浏览器无法解码 MKV/HEVC),播放全部改为外置播放器跳转
- **移除捐赠、合集功能**;下载设置去掉地址栏等无关注项
- 云端播放:`/api/file` 反代 115 CDN 流,支持 Range/seek;`getSubtitles` 对云盘返回空字幕
- MKV 内封字幕:流式读取,不整读大文件进内存
- MCP server、Swagger UI 暂未实现(默认关闭,不影响使用)

## 免责声明

本工具为中立性技术辅助工具,请遵守当地法律法规,勿用于盗版传播。