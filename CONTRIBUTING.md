# Contributing to Oasyce Chain

Thank you for your interest in contributing to Oasyce Chain!

> 中文版见下方 / Chinese version below

## Getting Started

### Prerequisites

- Go 1.21+
- Protocol Buffers compiler (`protoc`)
- Make

### Setup

```bash
git clone https://github.com/Shangri-la-0428/oasyce-chain.git
cd chain
make build
make test
```

### Running a Local Node

```bash
make build
./scripts/init_testnet.sh
./scripts/start_testnet.sh
```

## Development Workflow

### Branch Naming

- `feat/description` — New features
- `fix/description` — Bug fixes
- `refactor/description` — Code refactoring
- `docs/description` — Documentation

### Code Style

- Run `make lint` before submitting (uses `golangci-lint`)
- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions focused and short
- Add tests for new functionality

### Testing

```bash
make test          # Unit tests
make test-race     # Tests with race detector
./scripts/e2e_test.sh  # End-to-end (requires running node)
```

All PRs must pass CI (build + test + lint).

### Commit Messages

Use conventional format:

```
feat(module): short description

Longer explanation if needed.
```

Prefixes: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

### Pull Request Process

1. Fork the repository and create your branch from `main`
2. Add tests covering your changes
3. Ensure `make test` and `make lint` pass
4. Update documentation if needed
5. Submit PR with clear description of what and why

### Module Structure

Each module follows the standard Cosmos SDK layout:

```
x/<module>/
├── cli/           # CLI tx and query commands
├── keeper/        # Business logic
│   ├── keeper.go
│   ├── msg_server.go
│   └── query_server.go
├── types/         # Messages, queries, keys, errors
├── module.go      # Module registration
```

When adding a new keeper method:
1. Implement in `keeper/`
2. Add CLI command in `cli/tx.go` or `cli/query.go`
3. Add tests in `keeper/*_test.go`
4. Update integration tests if cross-module

### Proto Descriptor Requirements (CRITICAL)

This project uses **hand-written .pb.go files** (not protoc-generated). Every new message type MUST satisfy 3 runtime contracts or the SDK will panic at tx decode time:

1. **`Descriptor()` method** — returns `(fileDescriptor_xxx, []int{N})` where N = message index in the gzipped FileDescriptorProto
2. **`proto.RegisterType`** — called in `init()` with the correct full type URL
3. **`cosmos.msg.v1.signer` option** — stored as extension field 11110000 in the file descriptor's `MessageOptions`

These are **invisible to `go build`** and **invisible to keeper-level tests**. They only surface when a real transaction hits the tx codec.

#### Using the Patcher Tool

After adding new message types, run the patcher to sync file descriptors:

```bash
go run ./tools/patch_descriptors
```

The patcher will:
- Add missing RPC methods and message types to file descriptors
- Inject `cosmos.msg.v1.signer` options for any Msg types missing them
- Validate that every Msg type has a matching `Descriptor()` method in Go source
- Report errors if Go source is out of sync with file descriptors

#### Adding a New Msg Type Checklist

1. Write the Go struct with `Marshal`/`Unmarshal`/`ProtoMessage` methods
2. Add `Descriptor()` method referencing the correct file descriptor variable and index
3. Add `proto.RegisterType()` in `init()`
4. Run `go run ./tools/patch_descriptors` to add RPC + signer options
5. Add the message to `types/codec.go` `RegisterInterfaces()`
6. Add the message to `tests/integration/tx_codec_test.go` `allModuleMessages()`
7. Run `go test ./tests/integration/` to verify the full tx codec path

## Reporting Issues

- **Bugs**: Use the [bug report template](https://github.com/Shangri-la-0428/oasyce-chain/issues/new?template=bug_report.md)
- **Features**: Use the [feature request template](https://github.com/Shangri-la-0428/oasyce-chain/issues/new?template=feature_request.md)
- **Security**: See [SECURITY.md](SECURITY.md) — do NOT open public issues for vulnerabilities

## Community

- Discord: [https://discord.gg/tfrCn54yZW](https://discord.gg/tfrCn54yZW)

---

# 贡献指南

感谢你对 Oasyce Chain 的关注！

## 快速开始

```bash
git clone https://github.com/Shangri-la-0428/oasyce-chain.git
cd chain
make build && make test
```

## 开发规范

- 提交前运行 `make lint`
- 新功能必须有测试
- Commit 格式: `feat(模块): 描述`
- PR 需通过 CI (build + test + lint)

## Proto 描述符要求（重要）

本项目使用**手写 .pb.go 文件**。新增消息类型必须满足 3 个运行时约束：

1. **`Descriptor()` 方法** — 返回 `(fileDescriptor_xxx, []int{N})`
2. **`proto.RegisterType`** — 在 `init()` 中注册完整类型 URL
3. **`cosmos.msg.v1.signer` 选项** — 文件描述符中的扩展字段

新增 Msg 类型后运行：
```bash
go run ./tools/patch_descriptors   # 自动补全 RPC、signer 选项
go test ./tests/integration/       # 验证 tx 编解码
```

## 提交 PR

1. Fork 并从 `main` 创建分支
2. 添加测试
3. 确保 `make test` 和 `make lint` 通过
4. 提交 PR，说明改了什么、为什么改

## 社区

- Discord: [https://discord.gg/tfrCn54yZW](https://discord.gg/tfrCn54yZW)
