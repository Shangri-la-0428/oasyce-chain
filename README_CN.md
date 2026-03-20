# Oasyce Chain

[![CI](https://github.com/oasyce/chain/actions/workflows/ci.yml/badge.svg)](https://github.com/oasyce/chain/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.50.10-blue)](https://github.com/cosmos/cosmos-sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> English version: [README.md](README.md)

**AI 代理之间的权益结算层。**

Oasyce Chain 是一条 Cosmos SDK 应用链——AI 代理之间的每一次数据访问和能力调用，都会被自动定价、托管、结算。数据有主权，能力有价格。

---

## 为什么需要 Oasyce？

现在的 AI 免费使用你的数据。Oasyce 改变这一点：

- 你拍了一张照片，AI 想用来训练——它必须付费，你自动赚钱
- 你做了一个翻译 API——其他代理调用它，你按次收费，质量由质押保证
- 定价自动（联合曲线），结算去信任（托管），争议去中心化（陪审投票）

可以理解为 **AI 经济的 Stripe**——专为机器对机器交易设计的支付结算层。

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
git clone https://github.com/oasyce/chain.git
cd chain
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

## 生态

```
oasyce-chain  — L1 共识层（本仓库）
oasyce CLI    — Python 薄客户端 + Dashboard
DataVault     — AI agent 数据管理 skill
```

| 组件 | 定位 | 安装 |
|------|------|------|
| [oasyce-chain](https://github.com/oasyce/chain) | L1 共识、状态、结算 | `make build` |
| [oasyce](https://github.com/Shangri-la-0428/Oasyce_Claw_Plugin_Engine) | Python 薄客户端、CLI、Dashboard | `pip install oasyce` |
| [DataVault](https://github.com/Shangri-la-0428/DataVault) | AI agent 数据资产管理 skill | `pip install odv[oasyce]` |

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

## 贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 安全

见 [SECURITY.md](SECURITY.md)。安全漏洞请勿公开提交 issue。

## 许可证

[Apache 2.0](LICENSE)

## 社区

- Discord: [https://discord.gg/tfrCn54yZW](https://discord.gg/tfrCn54yZW)
