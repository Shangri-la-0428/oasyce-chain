# Oasyce Chain IBC 跨链通信指南

## 1. IBC 概述

Oasyce Chain 集成了 ibc-go v8.8.0，支持 IBC (Inter-Blockchain Communication) 跨链通信协议。
通过 IBC，Oasyce Chain 可以与 Cosmos 生态中的其他链（如 Cosmos Hub、Osmosis 等）进行安全的跨链交互。

## 2. 支持的功能

| 功能 | 说明 |
|------|------|
| IBC Transfer | 跨链转账 OAS 代币（ics-20） |
| Tendermint 光客户端 | 07-tendermint 客户端，用于验证对端链的状态 |
| IBC Channel/Connection | 完整的通道与连接生命周期管理 |

## 3. 创建 IBC 通道

使用 Hermes 或 rly 中继器在 Oasyce Chain 与目标链之间建立通道。

### 3.1 使用 Hermes

```bash
# 1. 安装 hermes
cargo install ibc-relayer-cli --bin hermes

# 2. 配置 config.toml，添加两条链的 RPC/gRPC 端点和密钥
#    参考: https://hermes.informal.systems/documentation/configuration

# 3. 创建客户端、连接和通道（一条命令完成）
hermes create channel \
  --a-chain oasyce-1 \
  --b-chain cosmoshub-4 \
  --a-port transfer \
  --b-port transfer \
  --new-client-connection --yes

# 4. 启动中继
hermes start
```

### 3.2 使用 rly (Go Relayer)

```bash
# 1. 初始化并添加链配置
rly chains add oasyce-1 --file oasyce.json
rly chains add cosmoshub-4 --file cosmoshub.json

# 2. 添加密钥
rly keys restore oasyce-1 default "your mnemonic..."
rly keys restore cosmoshub-4 default "your mnemonic..."

# 3. 创建路径并建立链接
rly paths new oasyce-1 cosmoshub-4 oasyce-hub
rly tx link oasyce-hub

# 4. 启动中继
rly start oasyce-hub
```

## 4. 跨链转账

### 发送 OAS 到其他链

```bash
oasyced tx ibc-transfer transfer transfer channel-0 \
  cosmos1xxx 1000000uoas \
  --from alice \
  --fees 10000uoas \
  --chain-id oasyce-1
```

### 从其他链接收代币后查看余额

```bash
oasyced query bank balances $(oasyced keys show alice -a)
```

IBC 接收的代币以 `ibc/HASH` 格式的 denom 显示。

## 5. 查询 IBC 状态

```bash
# 查看所有 IBC 通道
oasyced query ibc channel channels

# 查看指定通道
oasyced query ibc channel end transfer channel-0

# 查看 IBC 连接
oasyced query ibc connection connections

# 查看跨链代币来源
oasyced query ibc-transfer denom-traces

# 查看特定 denom 的来源路径
oasyced query ibc-transfer denom-trace <hash>

# 查看转账托管地址余额
oasyced query ibc-transfer escrow-address transfer channel-0
```

## 6. 安全注意事项

- **通道授权**: 仅与经过验证的链建立 IBC 通道，建议通过治理提案审核新通道。
- **中继器密钥管理**: 中继器密钥仅需少量 gas 费用的余额，不要存放大量资金。
- **客户端过期**: 监控 Tendermint 客户端的 trusting period，确保中继器持续运行以防止客户端过期。
- **Packet 超时**: 跨链交易设置合理的超时时间（默认 10 分钟），超时后资金会自动退回。
- **版本兼容**: 确保对端链的 ibc-go 版本与 Oasyce Chain (v8.8.0) 兼容。
