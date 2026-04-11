# Oasyce Testnet Operations Runbook

公测期间运维手册。每日巡检 + 应急响应。

---

## 控制面原则

VPS 运维现在有三条控制面，优先级固定如下：

1. **Alibaba Cloud CLI + Cloud Assistant**
2. **GitHub Actions 手动运维 workflow**
3. **直接 SSH**

原因：

- 直接 SSH 是一个入口，不应成为唯一入口
- Cloud Assistant 已在线，能稳定执行命令
- GitHub Actions runner 也能稳定通过 SSH 进入机器
- 所以即使本机到 `47.93.32.88:29222` 的 SSH 路径抽风，VPS 仍然可维护

本地推荐先验证控制面：

```bash
aliyun ecs DescribeInstances --RegionId cn-beijing --InstanceIds '["i-2ze3c737ux27bp7j38rq"]'
aliyun ecs DescribeCloudAssistantStatus --RegionId cn-beijing --InstanceId '["i-2ze3c737ux27bp7j38rq"]'
```

Cloud Assistant 标准执行入口：

```bash
scripts/ecs_cloud_run.sh 'hostname && whoami && systemctl is-active ssh'
```

固定脚本入口：

```bash
scripts/ecs_chain_status.sh
scripts/ecs_alert_status.sh
scripts/ecs_tail_unit.sh oasyced
scripts/ecs_service_restart.sh oasyced
```

---

## 每日巡检（5 分钟）

```bash
# 一键巡检（优先走 Cloud Assistant）
scripts/ecs_cloud_run.sh '
echo "=== Node ===" && systemctl is-active oasyced && systemctl is-active oasyce-faucet
echo "=== Block ===" && curl -s localhost:26657/status | python3 -c "import sys,json;d=json.load(sys.stdin)[\"result\"][\"sync_info\"];print(\"height:\",d[\"latest_block_height\"],\" catching_up:\",d[\"catching_up\"])"
echo "=== Disk ===" && df -h / | tail -1
echo "=== Mem ===" && free -m | grep Mem
'
```

```bash
# 如果 Cloud Assistant 临时不可用，再退回 SSH
ssh -p 29222 root@47.93.32.88 'hostname'
```

```bash
# Faucet 余额（本地 REST，无需 SSH）
curl -s http://47.93.32.88:1317/cosmos/bank/v1beta1/balances/oasyce1msmqqjw64k8m827w3apda97umxt9lgfxszr25d
```

### Healthcheck 运行边界

当前健康巡检只有两条真实入口：

- `root` crontab：`*/5 * * * * /opt/oasyce/src/scripts/healthcheck.sh`
- `oasyce` crontab：`*/30 * * * * /usr/bin/python3 /opt/oasyce/src/scripts/consumer_agent.py >> /var/log/oasyce-consumer.log 2>&1`

状态文件不再放在 `/tmp`，而是：

- healthcheck state: `/var/lib/oasyce-healthcheck/`
- consumer state: `/var/lib/oasyce-consumer/state.json`

这意味着：

- `/tmp` 清理不会再重置告警去重状态
- healthcheck 会自动探测 consumer 是否真的部署，未部署时不再误报 `Consumer agent STALE`
- healthcheck 有执行锁，同一时间只允许一份实例运行
- 每个告警 key 都有冷却窗口，避免重复邮件风暴

## 本地自治验收

链仓库现在有三条不同的验收面，职责固定如下：

- `./scripts/e2e_test.sh`：CLI / module E2E，验证链模块和传统 `oasyced` 交易面
- `python3 scripts/e2e_autonomy.py --sdk-mode source`：AI autonomy acceptance，只验证 wrapper 经济闭环
- `python3 scripts/live_gate_local.py`：推荐默认入口，自举 tempnet，串起 fresh build、Pulse hard-pass、wrapper 自治闭环

### 推荐默认入口

主 gate 现在固定为：

```bash
python3 scripts/live_gate_local.py
```

它会自己完成这些步骤：

- fresh build 当前源码到 `build/oasyced`
- 在临时 home 中 init 一个 tempnet，不复用 `~/.oasyced`
- 启动本地节点并等待 REST / RPC ready
- 跑 `check_sdk_surface.py --mode source`
- 跑 `check_pulse_compat.py --sdk-mode source`
- 跑 `e2e_autonomy.py --sdk-mode source`

通过标准固定为：

- 当前 fresh build 的 `oasyced` 包含 `tx sigil pulse`
- tempnet 可以正常出块
- Pulse 兼容检查返回 `chain-ready` + `thronglets-ready` + `sdk-ready`
- `cli_live_tx` 和 `sdk_live_tx` 都成功
- wrapper 闭环完成 provider register、consumer invoke / feedback、data asset register、buy-shares

### Source Truth 与 preflight

链侧验收现在默认只认相邻 `oasyce-sdk` 源码 checkout：

```bash
python3 scripts/check_sdk_surface.py --mode source
```

规则：

- `source` 模式下，`oasyce_sdk.__version__` 必须和相邻 checkout 的 `pyproject.toml` 一致
- `resolve_identity` / `Identity`、scanner、`pulse_sigil`、`MsgPulse.dimensions` schema support 都是 hard requirement
- `pip` 安装出来的 distribution metadata 漂移只算 preflight warning，不计入 live gate 失败

如果你只是想清理本机 editable / dist 元数据，再单独执行：

```bash
pip install -e /Users/wutongcheng/Desktop/oasyce-sdk
```

这不是 live gate 的阻断步骤，只是环境同步动作。

### 什么时候单独跑 probe

以下场景先单独跑 probe，能更快定位问题：

- 修改了 `scripts/_sdk_compat.py`
- 修改了 `scripts/check_sdk_surface.py`
- 修改了 `scripts/check_pulse_compat.py`
- 想区分是 SDK surface 漂移、Pulse live path 回归，还是 wrapper 闭环问题

常用命令：

```bash
python3 scripts/check_sdk_surface.py --mode source
python3 scripts/check_pulse_compat.py --sdk-mode source
python3 scripts/e2e_autonomy.py --sdk-mode source
```

### `e2e_autonomy.py` 的定位

`e2e_autonomy.py` 现在继续保持单一职责，不承担 Pulse broadcaster：

- provider wrapper 可以 `--register` 并作为本地 HTTP provider 启动
- consumer wrapper 能发现 capability、invoke、拿到 provider 响应并提交 feedback
- data wrapper 能扫描本地目录并注册一个 safe asset
- consumer wrapper 能完成一次 data shares 购买
- 整条写链路径来自 `oasyce-sdk` native signer，不允许依赖 `oasyced tx` subprocess

### 失败时先看哪里

- `build step failed`：优先看 fresh build 输出，确认当前源码能产出带 `tx sigil pulse` 的二进制
- `sdk surface check failed`：优先跑 `python3 scripts/check_sdk_surface.py --mode source`
- `pulse compatibility check failed`：优先跑 `python3 scripts/check_pulse_compat.py --sdk-mode source`
- `cli_live_tx` 失败：先看 binary 是否是 fresh build，再看 tempnet 是否 ready
- `sdk_live_tx` 失败：先看 SDK source seam、`pulse_sigil`、`MsgPulse.dimensions` schema support
- `autonomy acceptance failed`：优先看 `provider-register.log`、`provider.log`、`consumer.log`、`data-agent.log`

`live_gate_local.py` 默认会清理临时目录。调试时可以加：

```bash
python3 scripts/live_gate_local.py --keep-temp-dir
```

## Sigil v2 Testnet Upgrade Prep

`x/sigil` 现在已经有 `ConsensusVersion = 2` 和 `Migrate1to2`。这一轮升级的定位是：

- state migration only
- no new store key
- chain query surface can evolve independently, but the upgrade itself adds no new store
- 目标是把旧的 active liveness index 从 `LastActiveHeight` 语义重建到 `MaxPulseHeight()` 语义

仓库内默认的升级计划名是：

```bash
v0.8.0
```

Proposal-ready artifacts live in:

```bash
docs/upgrades/v0.8.0/
```

其中固定包含：

- `proposal.template.json`
- `metadata.template.json`
- `CHECKLIST.md`

### 本地 rehearsal：真实 fixture replay

本地 rehearsal 不再追求完整的 `proposal -> vote -> apply`。它只做一件事：

- 用 **复制的 VPS pre-upgrade node home** 作为 fixture
- 在 repo-local temp dir 里重放 `UpgradeV080`
- 审计 `0x05` / `0x09` bucket、active_count、orphan index、双桶冲突

```bash
go run ./tools/v080_fixture_audit audit-home \
  --home /path/to/copied-vps-home \
  --output ./tmp/reports/v080-audit-before.json

go run ./tools/v080_fixture_audit replay-v080 \
  --source-home /path/to/copied-vps-home \
  --working-home ./tmp/v080-replay \
  --output ./tmp/reports/v080-replay-report.json
```

通过标准：

- `active_count` 前后一致
- active sigils 只在 `0x05`
- dormant sigils 只在 `0x09`
- dissolved sigils 不在任何 liveness bucket
- 没有 orphan index entry
- `sigil` 模块版本迁移到 `2`

### VPS 真实升级执行

真实 proposal / vote / upgrade 只在 VPS 那次执行。必要时可以使用现有 `--expedited`，但不要为了 rehearsal 改参数。

推荐命令：

```bash
GOOS=linux GOARCH=amd64 go build -o ./tmp/oasyced-v0.8.0 ./cmd/oasyced
shasum -a 256 ./tmp/oasyced-v0.8.0
scp -P 29222 ./tmp/oasyced-v0.8.0 root@47.93.32.88:/tmp/oasyced-v0.8.0
```

升级前后核验以 [`docs/upgrades/v0.8.0/CHECKLIST.md`](./upgrades/v0.8.0/CHECKLIST.md) 为准，尤其要填完：

- binary SHA
- proposal id
- 是否 expedited
- 升级高度
- 前后 `/health`
- 至少一个 replay-selected canary 的 `active -> dormant` 观察结果

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
scripts/ecs_cloud_run.sh 'systemctl status oasyced --no-pager'

# 2. 查看最近日志
scripts/ecs_cloud_run.sh 'journalctl -u oasyced --since "5 min ago" --no-pager | tail -50'

# 3. 重启
scripts/ecs_cloud_run.sh 'systemctl restart oasyced && sleep 5 && systemctl is-active oasyced'
```

### Faucet 无响应

```bash
scripts/ecs_cloud_run.sh 'systemctl restart oasyce-faucet && sleep 2 && curl -s localhost:8080/faucet?address=oasyce1a57fdrtq2wu65tjeyx9jyg4cku4evr8en4gyv5'
```

### Provider capability 被自动停用

先区分是短暂 upstream 抖动，还是 capability 真被链上停用：

```bash
scripts/ecs_cloud_run.sh '
curl -s http://127.0.0.1:8430/health
curl -s http://127.0.0.1:8430/health?probe=1
oasyced q oasyce_capability by-provider oasyce1a57fdrtq2wu65tjeyx9jyg4cku4evr8en4gyv5 --node http://127.0.0.1:26667 --output json
'
```

规则：

- `/health?probe=1` 只允许报告 `degraded`，不应再触发链上 `deactivate`
- 如果当前 capability 已经 `inactive`，注册一个新的 capability 并把 `/etc/oasyce/provider-capability.env` 切到新 ID
- 切换后重启 provider，并手动跑一次 consumer，确认链路恢复

```bash
scripts/ecs_cloud_run.sh 'bash /opt/oasyce/src/scripts/rotate_provider_capability.sh'
```

### Provider capability 生命周期

链上 capability 是不可删除的历史记录，所以“清理”不是删掉旧 ID，而是保证：

- 当前生效 capability 只有一个
- 当前 ID 固定写在 `/etc/oasyce/provider-capability.env`
- 轮换时总是“先注册新 capability，再切 current，最后退役旧 active capability”
- consumer 默认优先选择最新 active capability

### 邮件风暴排查

如果邮箱再次出现历史告警集中送达，先区分是“现在又在触发”，还是“旧邮件延迟投递”：

```bash
scripts/ecs_cloud_run.sh '
tail -n 50 /var/log/oasyce-alert.log 2>/dev/null || true
find /var/lib/oasyce-healthcheck -maxdepth 2 -type f -print -exec cat {} \;
'
```

判断标准：

- 如果 `/var/log/oasyce-alert.log` 没有新的 `ALERT:`，而 state 目录也没有新的 `.active` 文件，通常是旧邮件延迟投递
- 如果出现新的 `.active` 文件，说明当前 incident 还在触发，继续查 live 服务与探针

### 磁盘空间不足

```bash
# 清理旧日志
scripts/ecs_cloud_run.sh 'journalctl --vacuum-size=100M'
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
# 最小可重放状态副本（用于 v0.8.0 fixture replay / 升级审计）
ssh -p 29222 root@47.93.32.88 'systemctl stop oasyced && mkdir -p /tmp/oasyce-fixture/data && cp -R /home/oasyce/.oasyced/data/application.db /tmp/oasyce-fixture/data/ && systemctl start oasyced'
scp -P 29222 -r root@47.93.32.88:/tmp/oasyce-fixture ./backups/

# 关键文件备份
ssh -p 29222 root@47.93.32.88 'tar czf /tmp/oasyce-secrets.tar.gz /home/oasyce/secrets/'
scp -P 29222 root@47.93.32.88:/tmp/oasyce-secrets.tar.gz ./backups/
```

`oasyced export` 目前不是这条升级/审计路径的真相源；`v0.8.0` 相关 rehearsal 和 post-upgrade audit 一律使用复制的 node home / `data/application.db`。

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

如果本机 SSH 卡在 `Connection timed out during banner exchange`，先不要默认判断成 `sshd` 坏了。当前已验证过：

- VPS 侧 `sshd` 可以正常监听 `29222`
- `UFW` 和安全组都已放行 `29222/tcp`
- GitHub Actions runner 能稳定用同一把 key 登录
- Cloud Assistant 也在线

这时更可能是“本机到 ECS 的路径问题”，不是 VPS 内部问题。先改用：

- `scripts/ecs_cloud_run.sh`
- GitHub Actions 手动运维 workflow

不要因为单一本地 SSH 入口异常就误判整台 VPS 不可维护。
