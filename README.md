# Oasyce Chain

[![CI](https://github.com/Shangri-la-0428/oasyce-chain/actions/workflows/ci.yml/badge.svg)](https://github.com/Shangri-la-0428/oasyce-chain/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.50.10-blue)](https://github.com/cosmos/cosmos-sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> English version: [README_EN.md](README_EN.md) | LLM-optimized docs: [docs/llms.txt](docs/llms.txt)
>
> 官网: [chain.oasyce.com](https://chain.oasyce.com)

**Sigil 栈的公共生命周期账本、授权真相层与结算终局层。**

当 AI loops 开始跨设备、跨 delegate、跨市场协作，问题不再只是"怎么调用 API"或"怎么转账"，而是：**谁在持续存在？谁被授权执行？哪些承诺需要公开最终性？哪些交换需要清算？**

Stripe / x402 / Tempo 解决了"怎么付钱"。Oasyce 解决的是"为什么付钱是合理的"。

---

## 在 Sigil 栈中的角色

- `Sigil`：定义连续性与生命周期语法
- `oasyce-sdk`：在本地实例化 delegate body，并解析 binding / signer
- `Thronglets`：承载共享环境、trace、signal、presence
- `Psyche`：承载主观连续性和自我状态
- `Oasyce Chain`：记录生命周期事件、授权真相、承诺、结算与公共终局

因此，Chain 不是高频 runtime，也不是全部产品的前门。它只负责那些必须公开、持久、可审计、可最终裁决的事实。

## 独立采用 / Independent Adoption

`Oasyce Chain` 必须能被独立使用。

- 你可以只把它当成一个公共授权 / 结算 / 生命周期账本
- 你可以直接走 `CLI / REST / gRPC`
- 你不需要先安装 `Psyche`
- 你不需要先安装 `Thronglets`
- 你也不需要先安装 `oasyce-sdk`

`oasyce-sdk` 只是对“本地 delegate runtime + 链”这条路径的 bridge，不是 Chain 的存在前提。

---

## 不只是支付，也不只是市场

| 问题 | 支付方案 (Stripe, x402, Tempo) | Oasyce |
|------|-------------------------------|--------|
| **主体生命周期** | 不涉及 | `x/sigil` 记录 GENESIS / BOND / FORK / MERGE / DISSOLVE |
| **授权真相** | 平台侧 ACL / 私有配置 | `x/delegate` + 链上状态给出可验证授权边界 |
| **数据归属** | 不涉及 | 数据证券化——联合曲线定价、股份交易、版本迁移 |
| **公平定价** | 固定价格 / 线下协商 | Bancor 连续曲线——需求越多价格越高 |
| **服务交付** | 付款后祈祷 | 链上托管 + 挑战窗口 + 争议机制 |
| **信任** | 无 / 平台背书 | 链上信用评分（时间衰减 + 可验证反馈） |
| **纠纷** | 退单或无门 | 链上陪审投票，确定性裁决 |
| **准入** | KYC + 公司实体 | PoW 自注册，无许可 |

---

## 模块分层

### Tier 1: 公共原语

| 模块 | 角色 | TX | Query |
|------|------|-----|-------|
| **x/sigil** | 生命周期账本——GENESIS / BOND / FORK / MERGE / DISSOLVE | 7 | 6 |
| **x/anchor** | 证据桥——把稀疏 durable trace 锚定为公共证明 | 2 | 4 |
| **x/onboarding** | 无许可 GENESIS 路径——PoW 防女巫 + 空投减半经济学 | 2 | 3 |

### Tier 2: 授权与经济基础设施

| 模块 | 角色 | TX | Query |
|------|------|-----|-------|
| **x/delegate** | 授权执行面——principal 为 delegate 设预算与消息边界 | 4 | 4 |
| **x/settlement** | 结算骨架——原子托管、联合曲线、费用路由、销毁 | 3 | 4 |
| **x/datarights** | 资产/股份/访问/争议/版本迁移的经济层 | 11 | 10 |
| **x/halving** | 稀缺性调度——区块奖励与减半节律 | 0 | 2 |

### Tier 3: 组合出来的高层表面

| 模块 | 角色 | TX | Query |
|------|------|-----|-------|
| **x/capability** | 服务调用表面——注册/调用/挑战窗口/自动结算 | 8 | 5 |
| **x/reputation** | 反馈残留——时间衰减信誉、仲裁和定价参考 | 2 | 3 |
| **x/work** | 可验证工作表面——commit-reveal、多执行者共识 | 6 | 8 |

完整接口与工作流见 [docs/llms.txt](docs/llms.txt)。

---

<!-- BEGIN GENERATED:PUBLIC_BETA_ZH -->
## 加入公测网

Oasyce Testnet-1 现已上线。

公开测试的**唯一链侧接入文档**是 [docs/PUBLIC_BETA_CN.md](/Users/wutongcheng/Desktop/Net/oasyce-chain/docs/PUBLIC_BETA_CN.md)。先完成链上接入；只有在你需要 Dashboard、本地扫描或 Python 自动化时，再按需接上 `oas`、`oasyce-agent` 或 `oasyce-sdk`。

| 项目 | 值 |
|------|-----|
| Chain ID | `oasyce-testnet-1` |
| Seed | `3e5a914ab7e7400091ddf461fb14992de785b0cb@47.93.32.88:26656` |
| RPC | `http://47.93.32.88:26657` |
| REST | `http://47.93.32.88:1317` |
| Faucet | `http://47.93.32.88:8080/faucet?address=oasyce1...` |
| 公测指南 | [docs/PUBLIC_BETA_CN.md](https://github.com/Shangri-la-0428/oasyce-chain/blob/main/docs/PUBLIC_BETA_CN.md) |
| 安装 CLI | `bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/install_oasyced.sh)` |
| Windows CLI | `Invoke-WebRequest .../install_oasyced.ps1 -OutFile install_oasyced.ps1` |
| 准备账户 | `bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/bootstrap_public_beta_account.sh)` |
| Windows 账户 | `Invoke-WebRequest .../bootstrap_public_beta_account.ps1 -OutFile bootstrap_public_beta_account.ps1` |
| 准备节点 | `bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/bootstrap_public_beta_node.sh)` |
| 启动节点 | `bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/run_public_beta_node.sh)` |
| 产品侧指南 | [oasyce-net 公测指南](https://github.com/Shangri-la-0428/oasyce-net/blob/main/docs/public-testnet-guide.md) |
| Dashboard | `pip install oasyce && oas bootstrap && oas start` |
| oasyce-agent | [oasyce-sdk README](https://github.com/Shangri-la-0428/oasyce-sdk/blob/main/README.md) |
| API Reference | [chain.oasyce.com/docs.html](https://chain.oasyce.com/docs.html) |
| Validator Guide | [docs/VALIDATOR_SETUP.md](https://github.com/Shangri-la-0428/oasyce-chain/blob/main/docs/VALIDATOR_SETUP.md) |
| Releases | [latest](https://github.com/Shangri-la-0428/oasyce-chain/releases/latest) |
| Python SDK | `pip install -U "oasyce-sdk>=0.5.0"` ([GitHub](https://github.com/Shangri-la-0428/oasyce-sdk)) |
<!-- END GENERATED:PUBLIC_BETA_ZH -->

**最快路径（使用经济系统）：**
```bash
pip install oasyce-sdk
oasyce-agent start
```
```python
from oasyce_sdk.crypto import Wallet, NativeSigner
from oasyce_sdk import OasyceClient

wallet = Wallet.auto()  # 复用本机 binding；首次设备可先运行 oasyce-agent start
client = OasyceClient("http://47.93.32.88:1317")
signer = NativeSigner(wallet, client, chain_id="oasyce-testnet-1")
# 注册、调用、买卖、评价 — 纯 Python，零 Go 依赖
```

**运行节点/验证者（需要 VPS）：**
```bash
# Docker helper
bash scripts/join_testnet.sh

# 规范入口 → docs/PUBLIC_BETA.md / docs/VALIDATOR_SETUP.md
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
make test   # 278 tests
```

---

## Agent 开发者指南

### REST API（推荐）

```python
import requests

BASE = "http://<node>:1317"

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

完整 API 参考见 [docs/llms.txt](docs/llms.txt)。

---

## CLI 示例

```bash
# === Agent 注册（PoW 自注册，无需 KYC） ===
oasyced util solve-pow [address] --difficulty 16 --output json  # 先解 PoW
oasyced tx onboarding register [nonce] --from agent1 --output json --yes

# === 注册 AI 能力: register [name] [endpoint-url] [price] ===
oasyced tx oasyce_capability register "Translation API" \
  "https://api.example.com/translate" 500000uoas \
  --tags "nlp,translation" \
  --from provider --output json --yes

# === 调用能力（自动创建托管 + 结算） ===
oasyced tx oasyce_capability invoke [cap-id] --input '{"text":"hello","target":"zh"}' --from consumer --output json --yes

# === 完成调用（提交输出哈希，开始 100 区块挑战窗口） ===
oasyced tx oasyce_capability complete-invocation [inv-id] [sha256-output-hash] \
  --usage-report '{"prompt_tokens":150,"completion_tokens":80}' \
  --from provider --output json --yes

# === 认领付款（挑战窗口结束后） ===
oasyced tx oasyce_capability claim-invocation [inv-id] --from provider --output json --yes

# === 争议（挑战窗口内，消费者发起） ===
oasyced tx oasyce_capability dispute-invocation [inv-id] "reason" --from consumer --output json --yes

# === 注册数据资产: register [name] [content-hash] ===
oasyced tx datarights register "Medical Imaging Dataset" "abc123..." \
  --tags "medical,imaging" \
  --from alice --output json --yes

# === 购买数据股份（Bancor 曲线定价） ===
oasyced tx datarights buy-shares [asset-id] 1000000uoas --from bob --output json --yes

# === 卖出股份（反向曲线，5% 协议费） ===
oasyced tx datarights sell-shares [asset-id] 100 --from bob --output json --yes

# === 提交计算任务: submit-task [type] [input-hash] [input-uri] [max-cu] [bounty] ===
oasyced tx work submit-task data-cleaning [sha256] \
  "https://storage.example.com/input" 1000 1000uoas \
  --from submitter --output json --yes

# === 查询声誉 ===
oasyced query reputation show [address] --output json
oasyced query reputation leaderboard --output json
```

---

## 协议经济学

| 参数 | 值 |
|------|-----|
| 代币 | OAS（1 OAS = 1,000,000 uoas） |
| 联合曲线 | Bancor, CW = 0.5 |
| 托管释放费用分配 | 90% 提供者, 5% 协议, 2% 销毁, 3% 国库 |
| 卖出协议费 | 5% (从联合曲线卖出收益中扣除) |
| 储备金上限 | 卖出时最高 95% |
| 区块奖励 | 4→2→1→0.5 OAS/block 减半（每 10M 区块） |
| 区块时间 | ~5 秒 |
| 最大验证者 | 100 |
| 解绑期 | 21 天 |
| 挑战窗口 | 100 区块（~8 分钟） |
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
                    |   8 custom modules        |
                    |   gRPC :9090 / REST :1317 |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    |   oasyce-sdk (Agent SDK)  |
                    |   binding / signer / 数据入口 |
                    |   MCP Server + LangChain  |
                    |   pip install oasyce-sdk  |
                    +---------------------------+
```

### 生态

| 组件 | 定位 | 安装 |
|------|------|------|
| [oasyce-chain](https://github.com/Shangri-la-0428/oasyce-chain) (本仓库) | L1 结算链 | `make build` |
| [oasyce-sdk](https://github.com/Shangri-la-0428/oasyce-sdk) | Agent Runtime + binding/signer/MCP/LangChain | `pip install oasyce-sdk` |

---

## 核心机制

- **Bancor 连续联合曲线** — `tokens = supply * (sqrt(1 + payment/reserve) - 1)`。买的人越多价格越高，无需订单簿
- **反向曲线卖出** — `payout = reserve * (1 - (1 - tokens/supply)^2)`，95% 储备金上限
- **2% 通缩燃烧** — 每次托管释放燃烧 2%
- **分级访问门控** — >=0.1% → L0, >=1% → L1, >=5% → L2, >=10% → L3
- **陪审投票** — `sha256(disputeID + nodeID) * log(1 + reputation)`，5 人陪审，2/3 多数
- **挑战窗口** — 完成调用后 100 区块（~8 分钟）消费者可争议；无争议则提供者认领付款
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
