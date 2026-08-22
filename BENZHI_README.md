# BENZHI_README

## 项目说明
- 项目：benzhi-project-d88942e6-bec3-4ffe-8584-091a8550b826
- 项目用途：口译术语包发布工作台已完整实现。两条标准验收命令、go vet、go build 和前端 JavaScript 语法检查均通过；桌面与移动视口已实际截图检查。普通服务当前运行于 http://127.0.0.1:19081，使用内存数据库供试用。
- Go 工具链：`golang:1.24.0`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/termpackd -addr=127.0.0.1:19081 -selfcheck
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-d88942e6-bec3-4ffe-8584-091a8550b826-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-d88942e6-bec3-4ffe-8584-091a8550b826-arm64 linux/arm64
docker run -it benzhi-project-d88942e6-bec3-4ffe-8584-091a8550b826-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/termpackd -addr=127.0.0.1:19081 -selfcheck`
