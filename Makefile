.PHONY: build run clean test ui all dev

# 构建前端:pnpm build,产物拷贝到 internal/server/webui(后端编译时嵌入)
ui:
	cd ui && pnpm install && pnpm build
	find ui/dist -name '*.gz' -delete
	rm -rf internal/server/webui
	mkdir -p internal/server/webui
	cp -r ui/dist/. internal/server/webui/

# 构建后端(使用已嵌入的 webui)
build:
	go build -o bin/ani-rss ./cmd/ani-rss

# 前端 + 后端一次性构建
all: ui build

# 运行
run:
	go run ./cmd/ani-rss

# 开发:启动后端 + 前端 Vite dev server(热更新,自动代理 /api 到后端)
dev:
	bash scripts/dev.sh

clean:
	rm -rf bin ui/dist ui/node_modules

test:
	go vet ./...
	go test ./...