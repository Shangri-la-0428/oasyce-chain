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

## 提交 PR

1. Fork 并从 `main` 创建分支
2. 添加测试
3. 确保 `make test` 和 `make lint` 通过
4. 提交 PR，说明改了什么、为什么改

## 社区

- Discord: [https://discord.gg/tfrCn54yZW](https://discord.gg/tfrCn54yZW)
