#!/bin/bash
# 开发模式:后端 + 前端热更新
#   - Go 后端: http://127.0.0.1:7789
#   - Vite 前端: http://127.0.0.1:37789 (代理 /api -> 后端)
# Ctrl-C 会同时停掉两者。
set -e
trap 'kill 0' EXIT

echo "==> 启动 Go 后端 (http://127.0.0.1:7789)"
(cd "$(dirname "$0")/.." && go run ./cmd/ani-rss) &
sleep 2

echo "==> 启动 Vite dev server (http://127.0.0.1:37789)"
(cd "$(dirname "$0")/../ui" && pnpm dev)