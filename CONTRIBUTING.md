# 参与开发

欢迎提 Issue 和 PR。动手之前请先读这三份：

1. [原则](docs/principles.md)：违背其中任何一条的改动不会合并，特别是开头两条：只有主机和模型通信；主机和设备之间只经连接服务、端到端加密。
2. [设计文档](docs/design.md)：改了行为，在同一个 PR 里更新它。
3. [安全说明](docs/security.md)：面向用户的承诺，改动不能让它变成假话。

安全问题请按 [SECURITY.md](SECURITY.md) 私下报告，不要开公开 Issue。

## 本地开发

需要 Go（版本见 `go.mod`）。

```bash
go build ./...
go test -race ./...
go vet ./...
scripts/build-web.sh          # 编译网页客户端（WebAssembly）到 dist/web
```

不要用自己的正式主机跑自动化测试：配对、移除设备会影响你已经在用的设备。另起一个数据目录：

```bash
OMNILLM_HOME=/tmp/omnillm-dev OMNILLM_RELAY=off go run ./cmd/omnillm hub run
```

## 提交约定

- 提交信息用 `feat:`、`fix:`、`docs:`、`test:`、`refactor:`、`chore:` 开头。
- 修 bug 先写一条能复现它的测试。
- 给人看的文字（错误提示、界面）中英双语，说清发生了什么、下一步怎么做，不出现协议字段、错误码和内部名词（`relay`、`Noise`、`mailbox` 这些对用户说成“连接服务”“加密”等）。
- 安全规则写在代码里并有测试守住；只写在文档或提示词里不算。
- 新增依赖要说明理由；OmniLLM 首先是一个 Go 库，要能嵌入别的程序，库本身不读配置、不开端口、不写标准输出、没有全局状态。
- 发版时在 [docs/changelog.md](docs/changelog.md) 写一条。

## License

提交的代码按 [Apache-2.0](LICENSE) 授权。
