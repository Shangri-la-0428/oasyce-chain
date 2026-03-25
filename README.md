# Oasyce Chain

[![CI](https://github.com/oasyce/chain/actions/workflows/ci.yml/badge.svg)](https://github.com/oasyce/chain/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.50.10-blue)](https://github.com/cosmos/cosmos-sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> English version: [README_EN.md](README_EN.md) | LLM-optimized docs: [llms.txt](llms.txt)

**Where agents pay agents.**

Oasyce 是专为 AI Agent 经济设计的 L1 结算链。Agent 之间的每一次数据访问和能力调用，都会被自动定价、托管、结算。无需 KYC，无需信用卡，无需人类审批。

当 AI agent 数量远超人类、交易频率远超人类、单笔金额远低于人类时，Stripe 的模型不再适用。Agent 经济需要原生基础设施。

---

## 为什么不用 Stripe？

| 维度 | Stripe / 传统支付 | Oasyce / Agent 原生结算 |
|------|-------------------|------------------------|
| **身份** | 需要人类 KYC、公司实体 | Agent 自主注册（PoW 防女巫） |
| **最小交易** | ~$0.50（手续费限制） | 0.000001 OAS（gas 唯一成本） |
| **结算速度** | T+2 天 | ~5 秒（1 个区块） |
| **可编程性** | Webhook + API | 链上托管 + 可编程结算逻辑 |
| **争议解决** | 人工客服，30 天 | 链上陪审投票，确定性结果 |
| **许可** | 平台可冻结账号 | 无许可，抗审查 |
| **微支付** | 不可行 | 原生支持 |

---

## 六大模块

| 模块 | 功能 | TX 数 | Query 数 |
|------|------|-------|----------|
| **x/settlement** | 托管结算、Bancor 联合曲线定价、2% 通缩燃烧 | 3 | 4 |
| **x/capability** | AI 能力市场——注册端点、调用、自动结算 | 4 | 4 |
| **x/datarights** | 数据资产注册、股份买卖、分级访问、争议陪审、版本迁移 | 11 | 9 |
| **x/reputation** | 时间衰减声誉（30 天半衰期）、排行榜 | 2 | 3 |
| **x/work** | 有用工作证明——任务分发、commit-reveal 验证、结算 | 6 | 8 |
| **x/onboarding** | PoW 自注册（无 KYC）、空投减半经济学 | 2 | 3 |

**合计**：28 种交易类型，31 个查询端点，59 条 CLI 命令。

---

## 加入公测网

Oasyce Testnet-1 现已上线。

| 项目 | 值 |
|------|-----|
| Chain ID | `oasyce-testnet-1` |
| Seed | `390f9b726d7ab105aade989f444f06585bc06186@47.93.32.88:26656` |
| RPC | `http://47.93.32.88:26657` |
| REST | `http://47.93.32.88:1317` |
| Faucet | `http://47.93.32.88:8080/faucet?address=oasyce1...` |
| Binary 下载 | [v0.4.0 Release](https://github.com/Shangri-la-0428/oasyce-chain/releases/tag/v0.4.0) |

```bash
# 一键加入（Docker）
bash scripts/join_testnet.sh

# 或下载 binary 手动加入
# 详见 docs/VALIDATOR_SETUP.md
```

---

## 快速开始

### 构建

```bash
git clone https://github.com/Shangri-la-0428/oasyce-chain.git
cd oasyce-chain
CGO_ENABLED=0 make build
```

### 运行 4 验证者本地测试网

```bash
bash scripts/init_multi_testnet.sh
bash scripts/start_testnet.sh
```

端口分配：

| 节点 | P2P | RPC | REST API | gRPC |
|------|-----|-----|----------|------|
| node0 | 26656 | 26657 | 1317 | 9090 |
| node1 | 26756 | 26757 | 1417 | 9190 |
| node2 | 26856 | 26857 | 1517 | 9290 |
| node3 | 26956 | 26957 | 1617 | 9390 |

### 运行测试

```bash
make test   # 50+ tests across 7 suites
```

---

## Agent 开发者指南

### REST API（推荐）

```python
import requests

BASE = "http://localhost:1317"

# 查询所有 AI 能力
caps = requests.get(f"{BASE}/oasyce/capability/v1/capabilities").json()

# 查询账户余额
bal = requests.get(f"{BASE}/cosmos/bank/v1beta1/balances/{address}").json()

# 查询数据资产
asset = requests.get(f"{BASE}/oasyce/datarights/v1/data_asset/{asset_id}").json()

# 查询声誉
rep = requests.get(f"{BASE}/oasyce/reputation/v1/reputation/{address}").json()
```

### CLI + JSON（适合 AI agent 调用）

```bash
# 所有命令支持 --output json
oasyced query settlement escrow ESC001 --output json
oasyced query oasyce_capability list --output json
oasyced query datarights asset DATA_001 --output json
```

### gRPC（高性能）

```
localhost:9090
```

完整 API 参考见 [llms.txt](llms.txt)。

---

## CLI 示例

```bash
# === Agent 注册（PoW 自注册，无需 KYC） ===
oasyced tx onboarding register [nonce] --from agent1

# === 注册 AI 能力 ===
oasyced tx oasyce_capability register \
  --name "Translation API" \
  --endpoint "https://api.example.com/translate" \
  --price 500000uoas \
  --tags "nlp,translation" \
  --from provider

# === 调用能力（自动创建托管 + 结算） ===
oasyced tx oasyce_capability invoke [cap-id] '{"text":"hello","target":"zh"}' --from consumer

# === 注册数据资产 ===
oasyced tx datarights register \
  --name "Medical Imaging Dataset" \
  --content-hash "abc123..." \
  --tags "medical,imaging" \
  --from alice

# === 购买数据股份（Bancor 曲线定价） ===
oasyced tx datarights buy-shares [asset-id] 1000000uoas --from bob

# === 卖出股份（反向曲线，3% 协议费） ===
oasyced tx datarights sell-shares [asset-id] 100 --from bob

# === 提交计算任务 ===
oasyced tx work submit-task \
  --task-type "data-cleaning" \
  --input-hash [sha256] \
  --bounty 1000uoas \
  --from submitter

# === 查询声誉 ===
oasyced query reputation show [address]
oasyced query reputation leaderboard
```

---

## 协议经济学

| 参数 | 值 |
|------|-----|
| 代币 | OAS（1 OAS = 1,000,000 uoas） |
| 联合曲线 | Bancor, CW = 0.5 |
| 托管释放费用分配 | 90% 提供者, 5% 协议, 2% 销毁, 3% 国库 |
| 卖出协议费 | 5% |
| 储备金上限 | 卖出时最高 95% |
| 区块奖励 | 4→2→1→0.5 OAS/block 减半（每 10M 区块） |
| 区块时间 | ~5 秒 |
| 最大验证者 | 100 |
| 解绑期 | 21 天 |
| 陪审团规模 | 5 人 / 争议 |
| 陪审门槛 | 2/3 多数 |

### 空投减半经济学

| 注册人数 | 空投 | PoW 难度 |
|----------|------|----------|
| 0 – 10,000 | 20 OAS | 16 bits |
| 10,001 – 50,000 | 10 OAS | 18 bits |
| 50,001 – 200,000 | 5 OAS | 20 bits |
| 200,001+ | 2.5 OAS | 22 bits |

---

## 架构

```
                    +---------------------------+
                    |      oasyce-chain (Go)    |
                    |    Cosmos SDK v0.50.10    |
                    |   CometBFT Consensus     |
                    |   7 custom modules        |
                    |   gRPC :9090 / REST :1317 |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    |   oasyce (Python CLI)     |
                    |   Agent 客户端 + Dashboard |
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
| [oasyce-chain](https://github.com/Shangri-la-0428/oasyce-chain) (本仓库) | L1 结算链 | `make build` |
| [oasyce](https://github.com/Shangri-la-0428/Oasyce_Claw_Plugin_Engine) | Python Agent 客户端 + CLI + Dashboard | `pip install oasyce` |
| [DataVault](https://github.com/Shangri-la-0428/DataVault) | AI Agent 数据资产管理 Skill | `pip install odv[oasyce]` |

---

## 核心机制

- **Bancor 连续联合曲线** — `tokens = supply * (sqrt(1 + payment/reserve) - 1)`。买的人越多价格越高，无需订单簿
- **反向曲线卖出** — `payout = reserve * (1 - (1 - tokens/supply)^2)`，95% 储备金上限
- **2% 通缩燃烧** — 每次托管释放燃烧 2%
- **分级访问门控** — >=0.1% → L0, >=1% → L1, >=5% → L2, >=10% → L3
- **陪审投票** — `sha256(disputeID + nodeID) * log(1 + reputation)`，5 人陪审，2/3 多数
- **Commit-reveal PoUW** — `sha256(output_hash + salt + executor + unavailable)` 防抄袭
- **确定性任务分配** — `sha256(taskID + blockHash + addr) / log(1 + reputation)`
- **PoW 自注册** — `sha256(address || nonce)` 满足 N 位前导零，无 KYC 防女巫

---

## 贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 安全

见 [SECURITY.md](SECURITY.md)。安全漏洞请勿公开提交 issue。

## 许可证

[Apache 2.0](LICENSE)

## 社区

- Discord: [https://discord.gg/tfrCn54yZW](https://discord.gg/tfrCn54yZW)
