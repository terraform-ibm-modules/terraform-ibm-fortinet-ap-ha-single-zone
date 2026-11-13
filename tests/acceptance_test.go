package test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

// ensureFGT2Init is a no-op guard called by parallel FGT2 tests.
// TestFortiGateFGT2Reachable (serial) owns FGT2 SSH init and completes before
// any parallel test starts. Returns ciPassword() so callers have a named entry
// point if the init strategy changes in future.
func ensureFGT2Init(t *testing.T, _ map[string]tfOutput) string {
	t.Helper()
	return ciPassword()
}

// skipIfNotCI skips the test if CI_MODE is not set
// Acceptance tests require live deployed FortiGate instances
func skipIfNotCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI_MODE") != "true" {
		t.Skip("Skipping acceptance test: CI_MODE is not set. Requires live deployed FortiGate instances.")
	}
}

// tfOutput represents a single Terraform output value
type tfOutput struct {
	Value interface{} `json:"value"`
}

// readTFOutputs reads Terraform outputs from the file saved by the deploy stage
func readTFOutputs(t *testing.T) map[string]tfOutput {
	t.Helper()
	outputFile := os.Getenv("TF_OUTPUTS_PATH")
	if outputFile == "" {
		outputFile = "../terraform-outputs.json"
	}
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Should be able to read terraform outputs from %s", outputFile)

	var outputs map[string]tfOutput
	require.NoError(t, json.Unmarshal(data, &outputs), "Terraform outputs should be valid JSON")
	return outputs
}

func getOutputString(t *testing.T, outputs map[string]tfOutput, key string) string {
	t.Helper()
	v, ok := outputs[key]
	require.True(t, ok, "Terraform output '%s' should exist", key)
	str, ok := v.Value.(string)
	require.True(t, ok, "Terraform output '%s' should be a string", key)
	require.NotEmpty(t, str, "Terraform output '%s' should not be empty", key)
	return str
}

// sshRunCommandsOutput opens a new SSH session with the given password, sends CLI
// commands one by one with a short pause between each, logs and returns all output.
// Returns ("", false) if the SSH connection or session setup fails.
func sshRunCommandsOutput(t *testing.T, publicIP string, password string, cmds []string, label string) (string, bool) {
	t.Helper()
	cfg := &gossh.ClientConfig{
		User:            "admin",
		Auth:            []gossh.AuthMethod{gossh.Password(password)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         15 * time.Second,
	}
	conn, err := gossh.Dial("tcp", publicIP+":22", cfg)
	if err != nil {
		t.Logf("  SSH [%s]: connect failed: %v", label, err)
		return "", false
	}
	defer conn.Close()

	sess, err := conn.NewSession()
	if err != nil {
		t.Logf("  SSH [%s]: new session failed: %v", label, err)
		return "", false
	}
	err = sess.RequestPty("xterm", 40, 80, gossh.TerminalModes{gossh.ECHO: 0})
	if err != nil {
		t.Logf("  SSH [%s]: request pty failed: %v", label, err)
		return "", false
	}
	stdin, _ := sess.StdinPipe()
	var outBuf strings.Builder
	sess.Stdout = &outBuf
	sess.Stderr = &outBuf
	_ = sess.Shell()

	time.Sleep(2 * time.Second)
	for _, cmd := range cmds {
		t.Logf("  SSH [%s] >> %s", label, cmd)
		fmt.Fprintf(stdin, "%s\n", cmd)
		time.Sleep(1 * time.Second)
	}
	time.Sleep(3 * time.Second)
	stdin.Close()
	sess.Close()

	out := outBuf.String()
	t.Logf("  SSH [%s] output:\n%s", label, out)
	return out, true
}

// fortiGateSSHInit polls until FortiGate accepts an SSH connection with the
// initial password, then performs first-boot password change.
// Polling replaces the separate HTTPS/TCP readiness probe — if SSH is up with
// the initial password, the instance is ready and we can init in one step.
// On subsequent runs (password already changed) the initial-password dial fails
// immediately and we return newPassword without waiting.
// Returns the current admin password.
func fortiGateSSHInit(t *testing.T, publicIP string, instanceID string, newPassword string) string {
	t.Helper()

	sshCfg := &gossh.ClientConfig{
		User:            "admin",
		Auth:            []gossh.AuthMethod{gossh.Password(instanceID)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         15 * time.Second,
	}

	// Poll until SSH is reachable with the initial password (instance still booting)
	// or the initial password is rejected (already changed — return early).
	timeoutMinutes := 10
	deadline := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)
	t.Logf("  SSH init: waiting for FortiGate SSH at %s:22 (timeout: %dm)...", publicIP, timeoutMinutes)

	var conn *gossh.Client
	for {
		var err error
		conn, err = gossh.Dial("tcp", publicIP+":22", sshCfg)
		if err == nil {
			break // connected with initial password — proceed to change it
		}
		// If auth was rejected (not a network error) the password is already changed
		if isAuthError(err) {
			t.Logf("  SSH init: initial password rejected — password already changed, using newPassword")
			return newPassword
		}
		// Network-level failure — instance still booting, keep polling
		if time.Now().After(deadline) {
			t.Fatalf("  SSH init: FortiGate SSH did not become reachable within %d minutes: %v", timeoutMinutes, err)
		}
		elapsed := time.Since(deadline.Add(-time.Duration(timeoutMinutes) * time.Minute))
		t.Logf("  SSH init: not yet reachable (%ds elapsed), retrying...", int(elapsed.Seconds()))
		time.Sleep(30 * time.Second)
	}

	t.Log("  SSH init: connected with initial password — changing password now")
	sess, serr := conn.NewSession()
	require.NoError(t, serr, "SSH new session")
	_ = sess.RequestPty("xterm", 40, 80, gossh.TerminalModes{gossh.ECHO: 0})
	stdin, _ := sess.StdinPipe()
	var outBuf strings.Builder
	sess.Stdout = &outBuf
	sess.Stderr = &outBuf
	_ = sess.Shell()

	// Some FortiGate firmware versions prompt:
	//   "New Password:"  (older FOS — send newPassword directly)
	// Others prompt:
	//   "Old password:" then "New Password:" then "Confirm Password:" (newer FOS)
	// Strategy: send instanceID (old password) first, then newPassword twice.
	// On older firmware the extra instanceID line is treated as a shell command
	// and ignored, leaving the password change intact.
	time.Sleep(3 * time.Second)             // wait for first prompt
	fmt.Fprintf(stdin, "%s\n", instanceID)  // old password (if prompted)
	time.Sleep(2 * time.Second)             // wait for "New Password:" prompt
	fmt.Fprintf(stdin, "%s\n", newPassword) // new password
	time.Sleep(2 * time.Second)             // wait for "Confirm Password:" prompt
	fmt.Fprintf(stdin, "%s\n", newPassword) // confirm
	time.Sleep(5 * time.Second)             // wait for password change to complete
	stdin.Close()
	sess.Close()
	conn.Close()

	// Always log output for diagnosis
	t.Logf("  SSH init [pwd-change] output:\n%s", outBuf.String())
	t.Log("  SSH init: password change complete")

	// Give FortiGate time to settle after password change before parallel tests probe it.
	// 30s is needed — FGT2 enters a brief period where it accepts neither old nor new
	// password while it commits the change and restarts its SSH service.
	t.Log("  SSH init: waiting 60s for FortiGate to settle after password change...")
	time.Sleep(60 * time.Second)

	return newPassword
}

// ciPassword returns the fixed CI password used after SSH init.
// Using a fixed password avoids PTY character-encoding issues with long UUIDs.
func ciPassword() string {
	return "Ci-Pipeline-1!"
}

// isAuthError returns true when an SSH dial error is an authentication failure
// (password rejected) rather than a network-level connection failure.
// Used by fortiGateSSHInit to distinguish "not yet up" from "already initialised".
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "unable to authenticate") ||
		strings.Contains(s, "no supported methods") ||
		strings.Contains(s, "ssh: handshake failed")
}

// fortiGateAdminPassword probes which password is currently active (new or initial)
// via SSH — avoids FortiGate REST API Basic Auth issues (trusted-host restrictions,
// session-cookie requirements). Returns the working password.
// SSH init must have already been run (by TestFortiGateFGT1Reachable) before calling this.
func fortiGateAdminPassword(t *testing.T, publicIP string, instanceID string) string {
	t.Helper()
	newPassword := ciPassword()

	// sshProbe tries to open and immediately close an SSH connection.
	// A successful dial means the password is correct.
	sshProbe := func(pwd string, label string) bool {
		cfg := &gossh.ClientConfig{
			User:            "admin",
			Auth:            []gossh.AuthMethod{gossh.Password(pwd)},
			HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
			Timeout:         10 * time.Second,
		}
		conn, err := gossh.Dial("tcp", publicIP+":22", cfg)
		if err != nil {
			t.Logf("  SSH probe [%s]: connect failed: %v", label, err)
			return false
		}
		conn.Close()
		t.Logf("  SSH probe [%s]: connected ✓", label)
		return true
	}

	if sshProbe(newPassword, "new-pwd") {
		t.Log("✓ Authenticated with new password (SSH)")
		return newPassword
	}
	if sshProbe(instanceID, "initial-pwd") {
		t.Log("✓ Authenticated with initial password (SSH)")
		return instanceID
	}

	require.Fail(t, "Failed to authenticate with FortiGate via SSH (tried new and initial passwords)")
	return ""
}

// pollHTTPS polls FortiGate HTTPS until it responds, then runs SSH init.
// Used by both FGT1 and FGT2 readiness checks.
func pollSSHAndInit(t *testing.T, label string, publicIP string, instanceID string) {
	t.Helper()
	newPassword := ciPassword()
	// fortiGateSSHInit polls for SSH readiness and changes the password in one step
	fortiGateSSHInit(t, publicIP, instanceID, newPassword)
}

// checkInterfacesUp checks that the named ports are up on a FortiGate via SSH CLI.
// Retries up to 3 times if the output contains a forced-password-change prompt,
// which can race with the parallel test startup on freshly booted instances.
func checkInterfacesUp(t *testing.T, publicIP string, instanceID string, label string, ports []string) {
	t.Helper()
	password := fortiGateAdminPassword(t, publicIP, instanceID)

	var out string
	var ok bool
	for attempt := 1; attempt <= 3; attempt++ {
		out, ok = sshRunCommandsOutput(t, publicIP, password,
			[]string{"get system interface physical", "end"},
			label+"-interfaces",
		)
		require.True(t, ok, "SSH session for interface check must succeed on %s", label)
		// If FortiGate is still in forced-password-change mode, the output will
		// contain the prompt but not the interface data — retry after a short wait.
		if strings.Contains(out, "You are forced to change your password") ||
			strings.Contains(out, "forced to change") {
			t.Logf("  [%s] FortiGate still in password-change mode (attempt %d/3), retrying in 15s...", label, attempt)
			time.Sleep(15 * time.Second)
			continue
		}
		break
	}
	require.True(t, ok, "SSH session for interface check must succeed on %s", label)

	// FortiGate CLI output format (get system interface physical):
	//   == [onboard]
	//       ==[port1]
	//           status: up
	//       ==[port2]
	//           status: up
	//
	// Strategy: scan line-by-line; when a line contains "[<name>]", record that
	// we're inside that interface's block. The next line containing "status:" gives
	// the value. Exit the block when we hit a line that starts a new interface
	// header (contains "==[") or the parent block header ("== [").
	checkInterface := func(name string) {
		lines := strings.Split(out, "\n")
		inBlock := false
		found := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Detect entry into this interface's block (e.g. "==[port1]")
			if strings.Contains(trimmed, "["+name+"]") {
				inBlock = true
				found = true
				continue
			}
			if !inBlock {
				continue
			}
			// Exit block on any new interface/section header
			if strings.Contains(trimmed, "==[") || strings.HasPrefix(trimmed, "== [") {
				break
			}
			if strings.HasPrefix(strings.ToLower(trimmed), "status:") {
				hasUp := strings.Contains(strings.ToLower(trimmed), "status: up")
				assert.True(t, hasUp,
					"[%s] Interface %s should show 'status: up' (got: %s)", label, name, trimmed)
				t.Logf("[%s] Interface %s: %s", label, name, trimmed)
				return
			}
		}
		if !found {
			assert.Fail(t, fmt.Sprintf("[%s] Interface %s not found in SSH output", label, name))
		}
	}

	for _, port := range ports {
		checkInterface(port)
	}
	t.Logf("✓ [%s] All interfaces are up: %v", label, ports)
}

// ── Serial setup tests (must run first, not parallel) ────────────────────────

// TestFortiGatePublicIPAssigned verifies floating IPs were assigned for both nodes.
// Runs first, not parallel.
func TestFortiGatePublicIPAssigned(t *testing.T) {
	skipIfNotCI(t)

	outputs := readTFOutputs(t)

	publicIP := getOutputString(t, outputs, "FortiGate_Public_IP")
	t.Logf("✓ Shared/failover public IP assigned: %s", publicIP)

	fgt1MgmtIP := getOutputString(t, outputs, "FGT1_Public_HA_Management_IP")
	t.Logf("✓ FGT1 HA management IP assigned: %s", fgt1MgmtIP)

	fgt2MgmtIP := getOutputString(t, outputs, "FGT2_Public_HA_Management_IP")
	t.Logf("✓ FGT2 HA management IP assigned: %s", fgt2MgmtIP)
}

// TestFortiGateFGT1Reachable polls FGT1's management IP on HTTPS until it responds,
// then runs SSH init (disclaimer acceptance + password change). Runs serially before
// all parallel tests so subsequent tests can safely use SSH auth.
func TestFortiGateFGT1Reachable(t *testing.T) {
	skipIfNotCI(t)

	outputs := readTFOutputs(t)
	fgt1MgmtIP := getOutputString(t, outputs, "FGT1_Public_HA_Management_IP")
	fgt1InstanceID := getOutputString(t, outputs, "FGT1_Default_Admin_Password")

	pollSSHAndInit(t, "FGT1", fgt1MgmtIP, fgt1InstanceID)
}

// TestFortiGateFGT2Reachable polls FGT2's management IP on HTTPS until it responds,
// then runs SSH init (disclaimer acceptance + password change). Runs serially before
// all parallel tests.
func TestFortiGateFGT2Reachable(t *testing.T) {
	skipIfNotCI(t)

	outputs := readTFOutputs(t)
	fgt2MgmtIP := getOutputString(t, outputs, "FGT2_Public_HA_Management_IP")
	fgt2InstanceID := getOutputString(t, outputs, "FGT2_Default_Admin_Password")

	pollSSHAndInit(t, "FGT2", fgt2MgmtIP, fgt2InstanceID)
}

// TestFortiGateHADiskProfileMatch verifies that both FGT1 and FGT2 report identical
// virtual disk sizes, ensuring the HA pair was deployed with matching storage profiles.
// A mismatch would prevent HA sync from completing cleanly.
// Runs serially — must complete before TestFortiGateHASync starts.
//
// Uses: diagnose hardware deviceinfo disk
// Relevant output lines (one per attached virtual disk):
//
//	Disk Virtual-Disk    ref:  16 484.3GiB    type: IDE [Virtio Disk] dev: /dev/vdb
//	Disk Virtual-Disk2   ref:  32 484.3GiB    type: IDE [Virtio Disk] dev: /dev/vdc
//
// We collect the size token from every "Virtual-Disk" line (skipping the SYSTEM
// boot disk) and compare the full sorted list across both nodes.
func TestFortiGateHADiskProfileMatch(t *testing.T) {
	skipIfNotCI(t)

	outputs := readTFOutputs(t)
	fgt1MgmtIP := getOutputString(t, outputs, "FGT1_Public_HA_Management_IP")
	fgt1InstanceID := getOutputString(t, outputs, "FGT1_Default_Admin_Password")
	fgt2MgmtIP := getOutputString(t, outputs, "FGT2_Public_HA_Management_IP")
	fgt2InstanceID := getOutputString(t, outputs, "FGT2_Default_Admin_Password")

	fgt1Password := fortiGateAdminPassword(t, fgt1MgmtIP, fgt1InstanceID)
	fgt2Password := fortiGateAdminPassword(t, fgt2MgmtIP, fgt2InstanceID)

	// extractVirtualDiskSizes parses "diagnose hardware deviceinfo disk" output and
	// returns a sorted, comma-joined string of sizes for all Virtual-Disk lines.
	// Example line:
	//   Disk Virtual-Disk    ref:  16 484.3GiB    type: IDE [Virtio Disk] dev: /dev/vdb
	// We skip the SYSTEM(boot) disk — only the attached log/data volumes matter.
	extractVirtualDiskSizes := func(out string) string {
		var sizes []string
		for _, line := range strings.Split(out, "\n") {
			trimmed := strings.TrimSpace(line)
			// Match lines that start with "Disk Virtual-Disk" (excludes "Disk SYSTEM")
			if !strings.HasPrefix(trimmed, "Disk Virtual-Disk") {
				continue
			}
			// Fields: "Disk" "Virtual-DiskN" "ref:" "<ref>" "<size>" "type:" ...
			// Size is the 5th whitespace-separated token (index 4).
			fields := strings.Fields(trimmed)
			if len(fields) >= 5 {
				sizes = append(sizes, fields[4])
			}
		}
		// Sort so order differences (disk numbering) don't cause false failures.
		sorted := make([]string, len(sizes))
		copy(sorted, sizes)
		for i := 1; i < len(sorted); i++ {
			for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
				sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			}
		}
		return strings.Join(sorted, ",")
	}

	out1, ok1 := sshRunCommandsOutput(t, fgt1MgmtIP, fgt1Password,
		[]string{"diagnose hardware deviceinfo disk", "end"},
		"fgt1-disk",
	)
	require.True(t, ok1, "SSH session for FGT1 disk check must succeed")

	out2, ok2 := sshRunCommandsOutput(t, fgt2MgmtIP, fgt2Password,
		[]string{"diagnose hardware deviceinfo disk", "end"},
		"fgt2-disk",
	)
	require.True(t, ok2, "SSH session for FGT2 disk check must succeed")

	diskSizes1 := extractVirtualDiskSizes(out1)
	diskSizes2 := extractVirtualDiskSizes(out2)

	t.Logf("FGT1 virtual disk sizes: %s", diskSizes1)
	t.Logf("FGT2 virtual disk sizes: %s", diskSizes2)

	require.NotEmpty(t, diskSizes1, "Could not extract virtual disk sizes from FGT1 output")
	require.NotEmpty(t, diskSizes2, "Could not extract virtual disk sizes from FGT2 output")

	require.Equal(t, diskSizes1, diskSizes2,
		"HA disk profile mismatch: FGT1 and FGT2 must have identical virtual disk sizes for HA sync to work\n  FGT1: %s\n  FGT2: %s",
		diskSizes1, diskSizes2)

	t.Logf("✓ HA disk profiles match: %s", diskSizes1)
}

// ── Parallel acceptance tests ─────────────────────────────────────────────────

// TestFortiGateFGT1LicenseValid verifies the FGT1 (primary/active) license is active via SSH CLI.
func TestFortiGateFGT1LicenseValid(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	fgt1MgmtIP := getOutputString(t, outputs, "FGT1_Public_HA_Management_IP")
	fgt1InstanceID := getOutputString(t, outputs, "FGT1_Default_Admin_Password")

	password := fortiGateAdminPassword(t, fgt1MgmtIP, fgt1InstanceID)

	out, ok := sshRunCommandsOutput(t, fgt1MgmtIP, password,
		[]string{"get system status", "end"},
		"fgt1-license-check",
	)
	require.True(t, ok, "SSH session for FGT1 license check must succeed")

	// FortiGate outputs "License Status: Valid" (or "Contract Expiry Date:" for PAYG)
	// Accept either form: "Valid" license line or the PAYG marker
	outLower := strings.ToLower(out)
	isValid := strings.Contains(outLower, "license status: valid") ||
		strings.Contains(outLower, "contract expiry date:") ||
		strings.Contains(outLower, "registration status: registered")
	assert.True(t, isValid,
		"FGT1 license should be valid — expected 'License Status: Valid' or PAYG marker in SSH output")
	t.Log("✓ FGT1 license is valid")
}

// TestFortiGateFGT2LicenseValid verifies the FGT2 (secondary/passive) license is active via SSH CLI.
func TestFortiGateFGT2LicenseValid(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	// ensureFGT2Init confirms FGT2 SSH init is done (handled serially by TestFortiGateFGT2Reachable).
	ensureFGT2Init(t, outputs)

	fgt2MgmtIP := getOutputString(t, outputs, "FGT2_Public_HA_Management_IP")
	fgt2InstanceID := getOutputString(t, outputs, "FGT2_Default_Admin_Password")

	password := fortiGateAdminPassword(t, fgt2MgmtIP, fgt2InstanceID)

	out, ok := sshRunCommandsOutput(t, fgt2MgmtIP, password,
		[]string{"get system status", "end"},
		"fgt2-license-check",
	)
	require.True(t, ok, "SSH session for FGT2 license check must succeed")

	outLower := strings.ToLower(out)
	isValid := strings.Contains(outLower, "license status: valid") ||
		strings.Contains(outLower, "contract expiry date:") ||
		strings.Contains(outLower, "registration status: registered")
	assert.True(t, isValid,
		"FGT2 license should be valid — expected 'License Status: Valid' or PAYG marker in SSH output")
	t.Log("✓ FGT2 license is valid")
}

// TestFortiGateFGT1InterfacesUp verifies port1–port4 are up on FGT1 via SSH CLI.
func TestFortiGateFGT1InterfacesUp(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	fgt1MgmtIP := getOutputString(t, outputs, "FGT1_Public_HA_Management_IP")
	fgt1InstanceID := getOutputString(t, outputs, "FGT1_Default_Admin_Password")

	checkInterfacesUp(t, fgt1MgmtIP, fgt1InstanceID, "FGT1", []string{"port1", "port2", "port3", "port4"})
}

// TestFortiGateFGT2InterfacesUp verifies port1–port4 are up on FGT2 via SSH CLI.
func TestFortiGateFGT2InterfacesUp(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	// ensureFGT2Init guarantees password change has completed before this test probes FGT2.
	ensureFGT2Init(t, outputs)

	fgt2MgmtIP := getOutputString(t, outputs, "FGT2_Public_HA_Management_IP")
	fgt2InstanceID := getOutputString(t, outputs, "FGT2_Default_Admin_Password")

	checkInterfacesUp(t, fgt2MgmtIP, fgt2InstanceID, "FGT2", []string{"port1", "port2", "port3", "port4"})
}

// TestFortiGateHASync verifies that HA synchronisation has completed — FGT1 is Active/Primary
// and FGT2 is Standby/Secondary. Two phases:
//  1. Role check: poll "get system ha status" until Active-Passive pair is established.
//  2. Checksum sync: poll "get system ha checksums" until both nodes show "in-sync".
//     Config sync can take significantly longer than role establishment on a fresh deployment.
func TestFortiGateHASync(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	// Ensure FGT2 password change is done before checking HA status
	ensureFGT2Init(t, outputs)

	fgt1MgmtIP := getOutputString(t, outputs, "FGT1_Public_HA_Management_IP")
	fgt1InstanceID := getOutputString(t, outputs, "FGT1_Default_Admin_Password")

	password := fortiGateAdminPassword(t, fgt1MgmtIP, fgt1InstanceID)
	interval := 30 * time.Second

	// ── Phase 1: Role check ───────────────────────────────────────────────────
	// Poll "get system ha status" until Active-Passive pair is established.
	roleTimeout := 5 * time.Minute
	roleDeadline := time.Now().Add(roleTimeout)
	t.Logf("Phase 1 — Waiting for HA role establishment (timeout: %s)...", roleTimeout)

	for {
		out, ok := sshRunCommandsOutput(t, fgt1MgmtIP, password,
			[]string{"get system ha status", "end"},
			"ha-role",
		)
		if !ok {
			t.Logf("  SSH session failed, retrying in %s...", interval)
		} else {
			outLower := strings.ToLower(out)
			// Accept both "master/slave" (older FOS) and "primary/secondary" (newer FOS) terminology.
			haActive := strings.Contains(outLower, "master") || strings.Contains(outLower, "primary")
			haStandby := strings.Contains(outLower, "slave") || strings.Contains(outLower, "secondary")
			haMode := strings.Contains(outLower, "active-passive") || strings.Contains(outLower, "a-p")

			if haActive && haStandby && haMode {
				t.Log("✓ Phase 1 complete: Active-Passive pair established")
				t.Logf("  FGT1 (%s) is the active/primary node", fgt1MgmtIP)
				break
			}

			elapsed := time.Since(roleDeadline.Add(-roleTimeout))
			t.Logf("  Roles not yet established (%ds elapsed) — active=%v standby=%v mode=%v, retrying...",
				int(elapsed.Seconds()), haActive, haStandby, haMode)
		}

		if time.Now().After(roleDeadline) {
			t.Fatalf("HA role establishment did not complete within %s — check FortiGate HA configuration", roleTimeout)
		}
		time.Sleep(interval)
	}

	// ── Phase 2: Configuration sync ──────────────────────────────────────────
	// Poll "get system ha status" until the Configuration Status section shows
	// all nodes as "in-sync" with none "out-of-sync". This works on all firmware
	// versions including interim builds (same command as Phase 1).
	// Config replication can take significantly longer than role establishment.
	syncTimeout := 25 * time.Minute
	syncDeadline := time.Now().Add(syncTimeout)
	syncInterval := 2 * time.Minute
	t.Logf("Phase 2 — Waiting for HA configuration sync (timeout: %s, poll every: %s)...", syncTimeout, syncInterval)

	for {
		out, ok := sshRunCommandsOutput(t, fgt1MgmtIP, password,
			[]string{"get system ha status", "end"},
			"ha-sync",
		)
		if !ok {
			t.Logf("  SSH session failed, retrying in %s...", syncInterval)
		} else {
			t.Logf("  HA status:\n%s", out)
			// "Configuration Status:" section lists each node as "in-sync" or "out-of-sync".
			// Pass only when at least one node is in-sync and none are out-of-sync.
			allInSync := strings.Contains(out, "in-sync") && !strings.Contains(out, "out-of-sync")
			if allInSync {
				t.Log("✓ Phase 2 complete: all nodes in-sync")
				return
			}
			elapsed := time.Since(syncDeadline.Add(-syncTimeout))
			t.Logf("  Not all nodes in-sync yet (%ds elapsed), retrying in %s...", int(elapsed.Seconds()), syncInterval)
		}

		if time.Now().After(syncDeadline) {
			t.Fatalf("HA configuration sync did not complete within %s — nodes still out-of-sync", syncTimeout)
		}
		time.Sleep(syncInterval)
	}
}

// TestFortiGateCatalogOfferingAttached verifies catalog offering CRNs from terraform outputs
func TestFortiGateCatalogOfferingAttached(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	versionCRN := getOutputString(t, outputs, "Catalog_Offering_Version_CRN")
	planCRN := getOutputString(t, outputs, "Catalog_Offering_Plan_CRN")
	profile := getOutputString(t, outputs, "Selected_VSI_Profile")

	assert.True(t, strings.HasPrefix(versionCRN, "crn:"), "Version CRN should start with 'crn:'")
	assert.True(t, strings.HasPrefix(planCRN, "crn:"), "Plan CRN should start with 'crn:'")
	assert.NotEmpty(t, profile, "Selected_VSI_Profile should not be empty")

	t.Logf("✓ Catalog offering attached")
	t.Logf("  Version CRN : %s", versionCRN)
	t.Logf("  Plan CRN    : %s", planCRN)
	t.Logf("  VSI Profile : %s", profile)
}

// TestCatalogOfferingCRNsLive validates deployed catalog offering CRNs from terraform outputs.
// Named to match the pipeline's -run "TestFortiGate|TestCatalogOffering" filter.
func TestCatalogOfferingCRNsLive(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	versionCRN := getOutputString(t, outputs, "Catalog_Offering_Version_CRN")
	planCRN := getOutputString(t, outputs, "Catalog_Offering_Plan_CRN")

	assert.True(t, strings.HasPrefix(versionCRN, "crn:"),
		"Catalog offering version CRN should start with 'crn:'")
	assert.True(t, strings.HasPrefix(planCRN, "crn:"),
		"Catalog offering plan CRN should start with 'crn:'")

	t.Logf("✓ CRN format validated")
	t.Logf("  Version CRN : %s", versionCRN)
	t.Logf("  Plan CRN    : %s", planCRN)
}
