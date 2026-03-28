# Oasyce Testnet Operations Runbook

公测期间运维手册。每日巡检 + 应急响应。

---

## 每日巡检（5 分钟）

```bash
# 一键巡检（本地执行）
ssh -p 29222 root@47.93.32.88 '
echo "=== Node ===" && systemctl is-active oasyced && systemctl is-active oasyce-faucet
echo "=== Block ===" && curl -s localhost:26657/status | python3 -c "import sys,json;d=json.load(sys.stdin)[\"result\"][\"sync_info\"];print(\"height:\",d[\"latest_block_height\"],\" catching_up:\",d[\"catching_up\"])"
echo "=== Disk ===" && df -h / | tail -1
echo "=== Mem ===" && free -m | grep Mem
'
```

```bash
# Faucet 余额（本地 REST，无需 SSH）
curl -s http://47.93.32.88:1317/cosmos/bank/v1beta1/balances/oasyce1msmqqjw64k8m827w3apda97umxt9lgfxszr25d
```

## 发布后必须确认

每次 `main` 推送后，先确认 GitHub Actions，不要只看 VPS 还活着：

```bash
gh run list -R Shangri-la-0428/oasyce-chain --limit 3
gh run view <run-id> -R Shangri-la-0428/oasyce-chain --json conclusion,status,jobs,url
```

必须满足：

- `build` 绿
- `test` 绿
- `lint` 绿
- `docker` 绿

如果有更新的 push，旧 run 直接取消，不要盯旧结果。

结构约束：

- `main` 只做单架构 docker 构建，优先给出快速信号
- `tag` 发布再做多架构镜像

### 基线值（2026-03-26 部署时）

| 指标 | 基线 | 告警阈值 |
|------|------|----------|
| 磁盘 | 6.1G / 40G (17%) | > 80% |
| 内存 | 319MB / 1608MB | > 1400MB |
| Swap | 0 / 2048MB | > 1024MB |
| Faucet 余额 | 49,999,900 OAS | < 10,000,000 OAS |
| 出块 | ~1 block/5s | 停滞 > 60s |

---

## 应急响应

### 节点停止出块

```bash
# 1. 检查服务状态
ssh -p 29222 root@47.93.32.88 'systemctl status oasyced'

# 2. 查看最近日志
ssh -p 29222 root@47.93.32.88 'journalctl -u oasyced --since "5 min ago" --no-pager | tail -50'

# 3. 重启
ssh -p 29222 root@47.93.32.88 'systemctl restart oasyced && sleep 5 && systemctl is-active oasyced'
```

### Faucet 无响应

```bash
ssh -p 29222 root@47.93.32.88 'systemctl restart oasyce-faucet && sleep 2 && curl -s localhost:8080/faucet?address=oasyce1a57fdrtq2wu65tjeyx9jyg4cku4evr8en4gyv5'
```

### 磁盘空间不足

```bash
# 清理旧日志
ssh -p 29222 root@47.93.32.88 'journalctl --vacuum-size=100M'
```

### 部署新二进制（热更新，不丢数据）

```bash
# 本地
cd /Users/wutongcheng/Desktop/Net/oasyce-chain
make build-linux
gzip -c build/oasyced-linux | ssh -p 29222 root@47.93.32.88 'cat > /tmp/oasyced.gz && gunzip -f /tmp/oasyced.gz && chmod +x /tmp/oasyced && systemctl stop oasyced && cp /tmp/oasyced /usr/local/bin/oasyced && systemctl start oasyced && systemctl restart oasyce-faucet && sleep 3 && oasyced version'
```

### 全链重置（清除所有数据，重新开始）

```bash
# 上传并执行 reset 脚本
scp -P 29222 scripts/reset_testnet.sh root@47.93.32.88:/tmp/
ssh -p 29222 root@47.93.32.88 'bash /tmp/reset_testnet.sh'
```

---

## 备份

```bash
# 导出链状态（可用于迁移或恢复）
ssh -p 29222 root@47.93.32.88 'su - oasyce -c "oasyced export 2>/dev/null" > /tmp/state-export.json'
scp -P 29222 root@47.93.32.88:/tmp/state-export.json ./backups/

# 关键文件备份
ssh -p 29222 root@47.93.32.88 'tar czf /tmp/oasyce-secrets.tar.gz /home/oasyce/secrets/'
scp -P 29222 root@47.93.32.88:/tmp/oasyce-secrets.tar.gz ./backups/
```

---

## 关键地址

| 角色 | 地址 |
|------|------|
| Validator | `oasyce1a57fdrtq2wu65tjeyx9jyg4cku4evr8en4gyv5` |
| Faucet | `oasyce1msmqqjw64k8m827w3apda97umxt9lgfxszr25d` |
| Mnemonics | VPS `/home/oasyce/secrets/` |

## SSH

```bash
ssh -p 29222 root@47.93.32.88
```
