
# Live Server 直播推流与播放示例

本项目演示如何使用 [SRS](https://ossrs.net/) 搭建本地 RTMP/FLV 直播服务，并通过前端页面播放直播流。

## 目录结构

```bash
├── js/flv.min.js      # flv.js 播放器库
├── video/test.mp4     # 示例视频
├── README.md          # 使用说明
```

## 依赖环境

- [Docker](https://www.docker.com/)（用于快速启动 SRS 服务）
- [ffmpeg](https://ffmpeg.org/)（用于推流）
- 现代浏览器（建议 Chrome）

## 快速开始

### 1. 启动 SRS 服务（RTMP/FLV）

```bash
docker run --rm -it -p 1935:1935 -p 1985:1985 -p 8080:8080 \
    registry.cn-hangzhou.aliyuncs.com/ossrs/srs:5
```

SRS 默认配置下：

- RTMP 推流地址：`rtmp://localhost:1935/live/livestream`
  
- FLV 拉流地址：http://localhost:8080/live/livestream.flv

### 2. 推送本地视频到 SRS（推流）

```bash
ffmpeg -re -i './video/test.mp4' -vcodec copy -acodec copy -f flv -y rtmp://localhost:1935/live/livestream
```

> 你可以替换 `./video/test.mp4` 为你自己的视频文件。

### 3. 播放直播流

直接用浏览器打开 `index.html`，即可看到直播画面。

> 默认播放地址为 `http://localhost:8080/live/livestream.flv`，如需更改请修改 `index.html` 里的 `url`。

## 前端播放说明

- 本项目使用 [flv.js](https://github.com/bilibili/flv.js) 实现浏览器端 FLV 播放。
- 支持直播低延迟播放，兼容主流浏览器（推荐 Chrome）。
- 如需自定义播放地址，可修改 `camera.vue` 中 flvjs.createPlayer 的 url 字段。

## 常见问题

1. **无法播放/黑屏**
   - 检查 SRS 服务是否已启动，推流命令是否执行成功。
   - 检查防火墙端口（1935/8080）是否开放。
   - 检查浏览器控制台是否有跨域或网络错误。

2. **推流失败**
   - 确认 ffmpeg 已安装，推流地址正确。
   - 视频文件路径是否正确。

3. **前端页面无法访问流**
   - 检查 `camera.vue` 里的 FLV 地址是否和 SRS 配置一致。
   - 检查 SRS 日志，确认有流被推送。

## 拓展

Mac投屏

```bash
ffmpeg -f avfoundation -i "Capture screen 0" \
       -vf "scale=1920:1080" \
       -c:v h264_videotoolbox \
       -b:v 12000k \
       -maxrate 15000k \
       -profile:v high \
       -f flv \
       "rtmp://localhost:1935/live/livestream"
```

## 参考资料

- [SRS 官方文档](https://ossrs.net/docs/v5/doc/getting-started)
- [flv.js 官方文档](https://github.com/bilibili/flv.js)
- [详细教程 (by:yimi233)](https://cloud.tencent.com/developer/article/2136460)

---
