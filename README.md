# Oasyce Chain

[![CI](https://github.com/oasyce/chain/actions/workflows/ci.yml/badge.svg)](https://github.com/oasyce/chain/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.50.10-blue)](https://github.com/cosmos/cosmos-sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> English version: [README_EN.md](README_EN.md)

**AI Agent 之间的权益结算层。**

Oasyce Chain 是一条基于 Cosmos SDK v0.50.10 的应用链——AI Agent 之间的每一次数据访问和能力调用，都会被自动定价、托管、结算。数据有主权，能力有价格。

可以理解为 **AI 经济的 Stripe**——专为机器对机器交易设计的支付结算层。

---

## 为什么需要 Oasyce？

现在的 AI 免费使用你的数据。Oasyce 改变这一点：

- 你拍了一张照片，AI 想用来训练——它必须付费，你自动赚钱
- 你做了一个翻译 API——其他 Agent 调用它，你按次收费，质量由质押保证
- 定价自动（Bancor 联合曲线），结算去信任（托管），争议去中心化（陪审投票）

---

## 模块

| 模块 | 功能 |
|------|------|
| **x/datarights** | 数据资产注册、股份买卖（Bancor 曲线）、争议提交、陪审投票、分级访问 |
| **x/settlement** | 托管生命周期、联合曲线定价、2% 通缩燃烧、费用分配 |
| **x/capability** | AI 能力注册（端点）、通过托管结算调用 |
| **x/reputation** | 基于反馈的声誉评分、时间衰减、排行榜 |

### 核心特性

- **Bancor 连续联合曲线** — 自动定价：`tokens = supply * (sqrt(1 + payment/reserve) - 1)`。买的人越多价格越高，无需订单簿。
- **卖出机制** — 通过反向曲线卖回：`payout = reserve * (1 - (1 - tokens/supply)^2)`，95% 储备金上限。
- **2% 通缩燃烧** — 每次托管释放燃烧 2%（93% 给提供者，5% 协议费，2% 燃烧），通缩设计。
- **分级访问门控** — 持有股权解锁访问级别：>=0.1% -> L0, >=1% -> L1, >=5% -> L2, >=10% -> L3，受声誉上限约束。
- **陪审投票** — 争议由确定性选举的陪审团解决（`sha256(disputeID + nodeID) * log(1 + reputation)`），需 2/3 多数。
- **托管结算** — 执行前锁定资金，质量验证后释放。自动过期退款。

---

## 快速开始

### 构建

```bash
git clone https://github.com/Shangri-la-0428/oasyce-chain.git
cd oasyce-chain
make build
```

### 运行本地节点

```bash
./scripts/init_testnet.sh
./scripts/start_testnet.sh
```

节点端口：
- **RPC**: `localhost:26657`
- **REST API**: `localhost:1317`
- **gRPC**: `localhost:9090`

### 运行测试

```bash
make test
```

### Docker（4 节点测试网）

```bash
make docker-build
docker-compose up
```

---

## CLI 示例

```bash
# 注册数据资产
oasyced tx datarights register \
  --name "Medical Imaging Dataset" \
  --content-hash "abc123..." \
  --rights-type 1 \
  --tags "medical,imaging" \
  --from alice

# 购买数据资产股份
oasyced tx datarights buy-shares \
  --asset-id DATA_xxxx \
  --amount 1000000uoas \
  --from bob

# 卖出股份（反向 Bancor 曲线）
oasyced tx datarights sell-shares \
  --asset-id DATA_xxxx \
  --shares 100 \
  --from bob

# 创建托管
oasyced tx settlement create-escrow \
  --provider cosmos1xxx \
  --amount 1000000uoas \
  --from alice

# 注册 AI 能力
oasyced tx oasyce_capability register \
  --name "Translation API" \
  --endpoint "https://api.example.com/translate" \
  --price 500000uoas \
  --from provider

# 查询声誉
oasyced query reputation show cosmos1xxx
```

---

## 架构

```
                    +---------------------------+
                    |      oasyce-chain (Go)    |
                    |    Cosmos SDK v0.50.10    |
                    |   CometBFT Consensus     |
                    |   -----------------------|
                    |   x/datarights            |
                    |   x/settlement            |
                    |   x/capability            |
                    |   x/reputation            |
                    |   gRPC :9090 / REST :1317 |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    |   oasyce (Python CLI)     |
                    |   薄客户端 + Dashboard     |
                    |   pip install oasyce      |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    |   DataVault (AI Skill)    |
                    |   本地数据资产管理          |
                    |   scan/classify/privacy   |
                    |   pip install odv[oasyce] |
                    +---------------------------+
```

### 生态

| 组件 | 定位 | 安装 |
|------|------|------|
| [oasyce-chain](https://github.com/Shangri-la-0428/oasyce-chain) (本仓库) | L1 共识、状态、结算 | `make build` |
| [oasyce](https://github.com/Shangri-la-0428/Oasyce_Claw_Plugin_Engine) | Python 薄客户端、CLI、Dashboard | `pip install oasyce` |
| [DataVault](https://github.com/Shangri-la-0428/DataVault) | AI Agent 数据资产管理 Skill | `pip install odv[oasyce]` |

---

## 协议经济学

| 参数 | 值 |
|------|-----|
| 代币 | OAS (uoas = 10^-6 OAS) |
| 联合曲线 | Bancor, CW = 0.5 |
| 引导价格 | 1 uoas/token |
| 协议费 | 托管释放时 5% |
| 燃烧率 | 托管释放时 2% |
| 储备金上限 | 卖出时最高 95% |
| 陪审团规模 | 每争议 5 名陪审员 |
| 陪审门槛 | 2/3 多数通过 |

---

## 当前进度

### Phase A: 核心链 — 已完成

- 4 个自定义模块完整实现（datarights, settlement, capability, reputation）
- 16 个 Protobuf 文件迁移完毕，gRPC + REST 全通
- Bancor 联合曲线 + 2% 燃烧 + 卖出机制 + 分级访问 + 陪审投票
- 全部 CLI 命令（tx + query）
- E2E 验证通过
- CI/CD、Docker 4 节点测试网、GitHub 基础设施

### Phase B: 生产就绪（进行中）

- IBC 跨链集成
- 治理模块
- 主网创世配置
- 安全审计
- Swagger API 文档
- 验证者激励计划
- 公共测试网上线

### Phase C: 有用工作证明 ✅

- x/work 模块：AI 计算任务提交 + 冗余执行 + 多数共识结算
- Commit-reveal 防抄袭，确定性执行者分配，信誉加权
- 经济模型：90% 执行者 / 5% 协议 / 2% 销毁 / 3% 提交者返利
- 6 条交易命令 + 8 条查询命令，13 个单元测试

### Phase D: 生态扩展（规划中）

- 跨链数据权益、隐私计算、移动端钱包、多语言 SDK

### Phase E: 去中心化 AI 市场（远期）

- Agent 自动发现、联邦学习、数据 DAO、收入共享

---

## 贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 安全

见 [SECURITY.md](SECURITY.md)。安全漏洞请勿公开提交 issue。

## 许可证

[Apache 2.0](LICENSE)

## 社区

- Discord: [https://discord.gg/tfrCn54yZW](https://discord.gg/tfrCn54yZW)
