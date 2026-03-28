package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const deployHealthcheckScript = "/Users/wutongcheng/Desktop/Net/oasyce-chain/deploy/healthcheck.sh"

func writeExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func runHealthcheckCommand(t *testing.T, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("bash", args...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestDeployHealthcheckCalculatesTwelveHourThreshold(t *testing.T) {
	out := runHealthcheckCommand(
		t,
		nil,
		"-lc",
		"source "+deployHealthcheckScript+"; calc_stale_threshold_checks 12 5",
	)
	if out != "144" {
		t.Fatalf("expected 144 checks for 12h/5m, got %q", out)
	}
}

func TestDeployHealthcheckEconomyStaleIsOptInAndDoesNotSpam(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "health_state")
	alertLog := filepath.Join(tmpDir, "alert.log")
	econLog := filepath.Join(tmpDir, "econ.log")
	mailFile := filepath.Join(tmpDir, "mail.txt")
	heightFile := filepath.Join(tmpDir, "height.txt")

	writeExecutable(t, tmpDir, "systemctl", `#!/bin/bash
if [ "$1" = "is-active" ]; then
  echo active
fi
exit 0
`)

	writeExecutable(t, tmpDir, "msmtp", `#!/bin/bash
cat >> "$TEST_MAIL_FILE"
`)

	writeExecutable(t, tmpDir, "curl", `#!/bin/bash
url="${@: -1}"
case "$url" in
  *"/health")
    if [ ! -f "$TEST_HEIGHT_FILE" ]; then
      echo 120 > "$TEST_HEIGHT_FILE"
    fi
    height=$(cat "$TEST_HEIGHT_FILE")
    printf '{"status":"ok","block_height":%s}\n' "$height"
    echo $((height + 1)) > "$TEST_HEIGHT_FILE"
    ;;
  *"/cosmos/bank/v1beta1/balances/"*)
    printf '{"balances":[{"amount":"999999999"}]}\n'
    ;;
  *"/oasyce/capability/v1/earnings/"*)
    printf '{"total_calls":"7","total_earned":[{"amount":"0"}]}\n'
    ;;
  *"http://127.0.0.1:8430/health")
    printf '{"status":"ok"}\n'
    ;;
  *)
    printf '{}\n'
    ;;
esac
`)

	baseEnv := []string{
		"PATH=" + tmpDir + ":" + os.Getenv("PATH"),
		"TEST_MAIL_FILE=" + mailFile,
		"TEST_HEIGHT_FILE=" + heightFile,
		"OASYCE_HEALTH_STATE_FILE=" + stateFile,
		"OASYCE_ALERT_LOG=" + alertLog,
		"OASYCE_ECON_LOG=" + econLog,
		"OASYCE_ECON_STALE_WINDOW_HOURS=1",
		"OASYCE_HEALTHCHECK_INTERVAL_MIN=5",
	}

	if err := os.WriteFile(stateFile+"_calls", []byte("7"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile+"_stale", []byte("11"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile+"_stale_alerted", []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", deployHealthcheckScript)
	cmd.Env = append(os.Environ(), baseEnv...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("healthcheck default mode failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(mailFile); err == nil {
		body, _ := os.ReadFile(mailFile)
		t.Fatalf("economy stale should be disabled by default, but mail was sent:\n%s", body)
	}
	if err := os.WriteFile(stateFile+"_stale", []byte("11"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile+"_stale_alerted", []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("bash", deployHealthcheckScript)
	cmd.Env = append(os.Environ(), append(baseEnv, "OASYCE_MONITOR_ECONOMY_STALE=1")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("healthcheck enabled mode failed: %v\n%s", err, out)
	}

	body, err := os.ReadFile(mailFile)
	if err != nil {
		t.Fatalf("expected one stale alert email: %v", err)
	}
	if !strings.Contains(string(body), "Economy STALE — no new invocations in 1+ hours (total_calls=7)") {
		t.Fatalf("unexpected mail content:\n%s", body)
	}

	cmd = exec.Command("bash", deployHealthcheckScript)
	cmd.Env = append(os.Environ(), append(baseEnv, "OASYCE_MONITOR_ECONOMY_STALE=1")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("healthcheck second enabled run failed: %v\n%s", err, out)
	}

	body, err = os.ReadFile(mailFile)
	if err != nil {
		t.Fatalf("expected mail file after second run: %v", err)
	}
	if strings.Count(string(body), "Subject: [Oasyce Alert]") != 1 {
		t.Fatalf("expected stale alert to fire once per stale episode, got:\n%s", body)
	}
}
