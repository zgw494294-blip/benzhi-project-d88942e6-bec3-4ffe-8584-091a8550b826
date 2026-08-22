# 口译术语包发布工作台

这是面向专业会议口译团队的单页 Web 工作台。术语编制人员、术语编辑、口译演练人员和发布负责人可以在同一条受状态机约束的流程中完成双语词条编制、逐项审定、候选版冻结、演练纠错、修订复核和最终发布。发布时系统生成带 SHA-256 内容摘要的不可变术语快照与批准凭据。

服务由 Go 标准 HTTP 服务器交付浏览器页面和同源 `/api/v1/term-packs` JSON API，业务数据、修订历史、演练发现、审计记录和发布凭据保存在 SQLite 中。默认数据库文件为 `termpacks.db`。

## 环境要求

- Go 1.24 或更高版本
- 无需 Node.js 或前端构建工具

## 构建

```sh
go build ./cmd/termpackd
```

## 运行

```sh
go run ./cmd/termpackd -addr=127.0.0.1:19081
```

浏览器打开 `http://127.0.0.1:19081`。可以使用 `-db` 指定 SQLite 文件路径。未传入 `-addr` 时默认监听 `127.0.0.1:19081`；如果环境变量 `PORT` 是有效的高位端口号，则监听 `127.0.0.1:<PORT>`。

## 测试

```sh
go test ./...
```

真实 HTTP 冒烟检查会启动回环服务，完成包含重大演练发现、关闭处理、第二修订和发布凭据在内的完整流程，然后自动关闭：

```sh
go run ./cmd/termpackd -addr=127.0.0.1:19081 -selfcheck
```

## 业务状态

标准路径为 `Draft`、`Submitted`、`Reviewed`、`Frozen`、`Rehearsal`、`Released`。重大或严重演练发现会使术语包进入 `ChangesRequested`；关闭当前冻结版的全部发现后，系统创建保留旧词条与处理记录的新修订，并从 `Draft` 重新审定。已发布术语包不可修改。

## 扩展能力

工作台同时提供草稿元数据修订、当前修订批量编制与批量审定、提交/审定预检报告、演练发现筛选统计与批量关闭、修订差异对比和发布凭据完整性核验。写入命令均要求 `expectedVersion` 与 `idempotencyKey`，批量命令在单事务中执行；查询接口保持只读。

主要查询入口包括：`GET /api/v1/term-packs/{id}/preflight`、`GET /api/v1/term-packs/{id}/findings`、`GET /api/v1/term-packs/{id}/revisions/{revision}/diff` 和 `GET /api/v1/term-packs/{id}/certificate/verify`。批量写入入口分别为 `POST /api/v1/term-packs/{id}/entries/batch`、`POST /api/v1/term-packs/{id}/entries/review-batch` 与 `POST /api/v1/term-packs/{id}/findings/resolve-batch`。
