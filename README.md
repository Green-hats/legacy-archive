# ani-rss-lite

> **背景**:硬盘越来越贵,追番囤货成本高,于是转向网盘——番剧直接离线下载到 115 网盘,不再占本地硬盘空间。
>
> **定位**:基于 **ani-rss** 的精简版:后端 Go 重写,前端按"网盘下载 + 外置播放器播放"定制,砍掉本地 BT 客户端、在线播放、合集、捐赠等非网盘场景功能。

## 特性

**下载**
- 115 云端离线下载(浏览器 Cookie 登录),自动重试、延迟下载、失败重试、备用 RSS 多源

**播放**
- 全部走外置播放器跳转:PotPlayer / VLC / IINA / MPV / Infuse / 弹弹Play / AnimacX / SenPlayer
- 云端文件经 `/api/file` 流式反代,支持 Range/seek

**追番**
- RSS 抓取、剧集提取、重命名模板、缺集/摸鱼检测与通知、备用 RSS

**刮削**
- TMDB(NFO/poster/fanart)、Bangumi 搜索/评分/OAuth;Mikan / ani-bt / anime-garden 番剧源

**通知**
- Telegram / Bark / ServerChan / WebHook / Shell / 邮件 / Emby 媒体库刷新

**安全**
- 登录鉴权、IP 白名单、反向代理信任列表

## 快速开始

**环境**:Go 1.26+(构建需要)· 外置播放器(MPV 另需 mpv-handler)

```bash
make all    # 前端 + 后端一体构建,产物 bin/ani-rss(或使用预编译二进制,待发布)
./ani-rss                       # 默认端口 7789,配置目录 ./config(自动创建)
PORT=7789 ./ani-rss             # 自定义端口
CONFIG=/path ./ani-rss          # 自定义配置目录
```

首次启动自动生成 `config.v2.json`、`ani.v2.json`、`logs/`,默认登录 **admin / admin**。已有上游 ani-rss 配置可直接指向迁移。

### 配置 115 下载

1. 浏览器登录 [115 网盘](https://115.com),复制 Cookie(`UID=...; CID=...; SEID=...; KID=...`)
2. Web UI → 设置 → 下载,填入 Cookie 并"测试"
3. 通过后开始云端离线下载,文件落盘在 115 网盘

### 外置播放器(MPV)

播放按钮走各播放器协议(`potplayer://`、`vlc://` 等),MPV 需安装 [akiirui/mpv-handler](https://github.com/akiirui/mpv-handler):

```bash
chmod +x ~/.local/bin/mpv-handler
xdg-mime default mpv-handler.desktop x-scheme-handler/mpv-handler
update-desktop-database ~/.local/share/applications/
```

- 用 `scripts/mpv-title-wrapper` 让 OSD/窗口标题显示文件名(在 `~/.config/mpv-handler/config.toml` 指向它)
- Firefox:`about:config` 新建布尔 `network.protocol-handler.expose.mpv-handler = false`,并允许一次协议唤起

## 开发

```bash
make dev    # 后端 :7789 + Vite 热更新 :37789(/api 自动代理)
make ui     # 仅构建前端并嵌入 internal/server/webui
make test   # go vet + go test
make clean  # 清理构建产物
```

## 与原版 ani-rss 的差异

| 保留 | 移除 |
|---|---|
| RSS 追番 / 重命名 / 缺集摸鱼通知 | 在线播放器(改外置播放器跳转) |
| 115 云端离线下载 | 本地 BT 客户端(qBittorrent、trackers、标签、做种等待) |
| TMDB / Bangumi 刮削、番剧源 | 合集、捐赠、QQ 机器人通知、自动更新 |
| Telegram / 邮件等通知 | 本地文件迁移/上传(OpenList、FileMove) |

后端 Go 重写、API 契约兼容,配置与订阅文件(`config.v2.json` / `ani.v2.json`)格式一致,可无缝迁移。

<details>
<summary>项目结构</summary>

```
cmd/ani-rss/        入口
ui/                 前端源码(Vue 3 + Vite,与后端同一仓库)
scripts/            开发脚本、mpv-title-wrapper
internal/
  model/            数据模型(JSON 契约与上游一致)
  config/           配置加载/持久化/部分合并
  server/           HTTP 路由、处理器、静态资源(嵌入 webui)
  service/          核心服务(下载引擎、订阅管理、刮削调度)
  task/             后台任务循环(rss/bgm)
  download/         下载客户端(115 正式,PikPak 开发中)
  rss/ rename/       RSS 解析 · 剧集提取与重命名
  bgm/ tmdb/         刮削 API 客户端
  mikan/ anibt/ animegarden/   番剧源抓取
  notify/ scrape/   通知发送器 · NFO 生成
  auth/ cache/ store/ util/    基础设施
```

</details>

<details>
<summary>常见问题</summary>

- **刷新后"共 0 个条目"**:多为排除规则 `\d-\d` 误伤 `H265-10bit` 等编码格式,删掉该条;匹配规则建议 `H265|HEVC|AV1`
- **AnimeGarden 订阅 400**:字幕组名含中文已在新版本自动编码处理
- **预览灰屏**:空 RSS 后端已保证返回空数组,更新到最新构建即可

</details>

## 免责声明

本工具为中立性技术辅助工具,请遵守当地法律法规,勿用于盗版传播。

## License

[ani-rss](https://github.com/wushuo894/ani-rss) 的精简版本,相关版权与开源许可遵循上游项目。