# Oasyce 工作记录 & 开源准备差距分析

> 最后更新: 2026-03-20

---

## 一、当前工作完成情况

### oasyce-chain (Go L1 链)

| 阶段 | 状态 | 说明 |
|------|------|------|
| 初始化 Cosmos SDK v0.50.10 | ✅ 完成 | app.go, go.mod, 4 custom modules |
| 4 模块实现 (settlement/capability/reputation/datarights) | ✅ 完成 | keeper + msg_server + query_server |
| Protobuf 迁移 | ✅ 完成 | 16 proto files, gRPC + REST |
| CLI 命令 | ✅ 完成 | tx + query for all modules |
| E2E 验证 | ✅ 完成 | 真实交易验证全部通过 |
| CI/CD | ✅ 完成 | GitHub Actions (build, test, lint, Docker) |
| Docker | ✅ 完成 | 多阶段构建 + 4 节点 docker-compose |
| Makefile | ✅ 完成 | build, test, lint, proto-gen, docker-* |
| Bancor 连续曲线 | ✅ 完成 | 替换阶梯定价, CW=0.5 |
| 2% Token Burn | ✅ 完成 | 93% provider + 5% fee + 2% burn |
| Sell 机制 | ✅ 完成 | 反向 Bancor, 95% 储备金上限 |
| Access Level Gating | ✅ 完成 | L0-L3, 股权阈值 + 声誉上限 |
| Jury Voting | ✅ 完成 | 确定性选举, 2/3 多数, 押金退还 |
| 安全修复 | ✅ 完成 | 确定性ID, Int64溢出防护, 最小费用 |
| Owner 主动下架 (DelistAsset) | ✅ 完成 | keeper + msg_server + test |
| 前端运行保护 (Slippage) | ✅ 完成 | MinSharesOut/MinPayoutOut 参数 |
| Jury 多种 remedy | ✅ 完成 | delist/rights_correction, fallback |
| CLI 同步 | ✅ 完成 | --remedy, --min-shares-out flags |
| 开源基础文件 | ✅ 完成 | README/LICENSE/CONTRIBUTING/CHANGELOG/SECURITY/CoC |
| GitHub 模板 | ✅ 完成 | issue templates, PR template |
| **测试** | **✅ 30+ tests, 5 suites** | `go test ./...` 全部通过 |
| **构建** | **✅ 干净** | `go build ./...` 无错误 |

### Oasyce Plugin Engine (Python 薄客户端)

| 阶段 | 状态 | 说明 |
|------|------|------|
| v2.1.0 发布 | ✅ 完成 | P3-P13 全部阶段, 1063 tests |
| 分层架构 | ✅ 完成 | L0 纯函数 → L4 界面, 零违规 |
| 协议规格 | ✅ 完成 | formulas.py 作为权威规格 |
| 开源基础设施 | ✅ 完成 | README/LICENSE/CONTRIBUTING/CHANGELOG/CI/CD |
| PyPI 发布 | ✅ 完成 | 自动化 release workflow |
| GitHub 同步 | ✅ 完成 | origin/main 同步 |

---

## 二、开源准备差距分析

### 评分总览

| 仓库 | 开源就绪度 | 阻塞项数 |
|------|-----------|---------|
| Plugin Engine | **95%** | 0 阻塞 |
| oasyce-chain | **90%** | 1 阻塞 (需配置 git remote + push) |
| DataVault | **85%** | 0 阻塞 |

### oasyce-chain — 阻塞项 (必须完成)

| # | 缺失项 | 优先级 | 工作量 | 说明 |
|---|--------|--------|--------|------|
| B1 | **README.md** | 🔴 阻塞 | 2h | 项目介绍、架构、快速启动、功能列表 |
| B2 | **LICENSE** | 🔴 阻塞 | 5min | Apache 2.0 (Cosmos SDK 生态标准) |
| B3 | **CONTRIBUTING.md** | 🔴 阻塞 | 1h | 开发规范、PR流程、测试要求 |
| B4 | **CHANGELOG.md** | 🔴 阻塞 | 30min | 版本历史 (v0.1.0) |
| B5 | **Git Remote** | 🔴 阻塞 | 5min | 未配置 remote, 无法推送 |
| B6 | **提交未提交代码** | 🔴 阻塞 | 30min | 55+ 文件未提交 (含所有链升级) |

### oasyce-chain — 高优先级 (强烈建议)

| # | 缺失项 | 优先级 | 工作量 | 说明 |
|---|--------|--------|--------|------|
| H1 | `.github/ISSUE_TEMPLATE/` | 🟡 高 | 30min | bug report + feature request 模板 |
| H2 | `.github/pull_request_template.md` | 🟡 高 | 15min | PR 检查清单 |
| H3 | `SECURITY.md` | 🟡 高 | 30min | 漏洞报告流程 |
| H4 | `CODE_OF_CONDUCT.md` | 🟡 高 | 5min | Contributor Covenant |
| H5 | `docs/ARCHITECTURE.md` | 🟡 高 | 1h | 模块关系图、数据流 |
| H6 | `docs/VALIDATOR_SETUP.md` | 🟡 高 | 1h | 验证者节点搭建指南 |

### oasyce-chain — 中优先级 (推荐)

| # | 缺失项 | 优先级 | 工作量 |
|---|--------|--------|--------|
| M1 | `docs/ECONOMICS.md` | 🟢 中 | 1h |
| M2 | `docs/API_REFERENCE.md` | 🟢 中 | 2h |
| M3 | README_CN.md (中文版) | 🟢 中 | 1h |
| M4 | Swagger/OpenAPI 文档 | 🟢 中 | 2h |
| M5 | 测试覆盖率报告 | 🟢 中 | 30min |

### Plugin Engine — 建议项 (非阻塞)

| # | 缺失项 | 优先级 | 工作量 |
|---|--------|--------|--------|
| P1 | `ROADMAP.md` | 🟢 中 | 30min |
| P2 | 提交 CLAUDE.md 变更 | 🟢 中 | 5min |

---

## 三、开源发布行动计划

### Phase 1: 基础设施 (当天可完成)

```
1. [B5] 创建 GitHub 仓库, 添加 remote
2. [B2] 添加 LICENSE (Apache 2.0)
3. [H4] 添加 CODE_OF_CONDUCT.md
4. [B4] 创建 CHANGELOG.md
5. [B3] 创建 CONTRIBUTING.md
6. [B6] 提交所有代码 → 推送到 GitHub
```

### Phase 2: README & 文档 (1-2天)

```
7. [B1] 写 README.md — 这是第一印象, 最重要
   - 项目愿景 (AI 能力自由市场)
   - 4 模块功能介绍
   - 30 秒快速启动
   - 架构图
   - 与 Plugin Engine 的关系
8. [H3] SECURITY.md
9. [H1] Issue templates
10. [H2] PR template
11. [H5] docs/ARCHITECTURE.md
```

### Phase 3: 宣传准备 (3-5天)

```
12. [M1] docs/ECONOMICS.md — 代币经济学白皮书
13. [H6] docs/VALIDATOR_SETUP.md
14. [M3] README_CN.md
15. [P1] ROADMAP.md — 展示未来方向
16. 准备宣传材料:
    - 项目介绍一页纸
    - Twitter/X 发布推文
    - Discord 公告
    - 技术博客文章
```

---

## 四、技术债务 (开源后可迭代)

| 问题 | 严重度 | 说明 |
|------|--------|------|
| E-7: 前端运行保护 | Medium | BuyShares 需要 minSharesOut 参数防止三明治攻击 |
| E-9: Jury 仅支持 delist | Low | 应支持 transfer/rights_correction 等补救措施 |
| A-4: ReserveRatio 常量未统一引用 | Low | settlement 和 datarights 各自计算 |
| MsgSellShares 非 proto | Low | 手写 Go struct, 无法用 CLI tx 广播 |
| x/work 模块 (PoUW) | 规划中 | Proof of Useful Work, Phase A+B 后实现 |

---

## 五、竞品对比 — 我们的差异化优势

| 特性 | Oasyce | Ocean Protocol | Fetch.ai |
|------|--------|---------------|----------|
| 数据确权 + 定价 | ✅ Bancor 自动定价 | ✅ 固定定价 | ❌ |
| AI 能力市场 | ✅ 托管 + 结算 | ❌ | ✅ Agent 框架 |
| 争议解决 | ✅ 陪审团投票 | ❌ | ❌ |
| 分级访问 | ✅ L0-L3 股权门控 | ❌ | ❌ |
| 连续定价曲线 | ✅ Bancor CW=0.5 | ❌ | ❌ |
| Token Burn | ✅ 2% 通缩 | ❌ | ❌ |
| 主权 L1 | ✅ Cosmos SDK | ❌ (EVM L2) | ✅ Cosmos SDK |
