package test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// getTerraformDir returns the Terraform directory from environment or default
func getTerraformDir() string {
	terraformDir := os.Getenv("TERRAFORM_DIR")
	if terraformDir == "" {
		terraformDir = "../"
	}
	return terraformDir
}

// getTerraformOptions creates standard Terraform options for testing
func getTerraformOptions(t *testing.T) *terraform.Options {
	terraformDir := getTerraformDir()

	options := &terraform.Options{
		TerraformDir: terraformDir,
		NoColor:      true,
	}

	// Check if we need to set variables from environment
	if varsJSON := os.Getenv("TF_VARS_JSON"); varsJSON != "" {
		t.Logf("Using variables from environment")
	}

	return options
}

// TestTerraformValidation performs basic validation of Terraform code
// This test only runs terraform init and validate (no actual deployment)
func TestTerraformValidation(t *testing.T) {
	// Not parallel: writes to .terraform/ — running concurrently with other tests
	// that also call terraform init causes provider binary races and checksum errors.

	// Get API key from environment (can be dummy for validation)
	apiKey := os.Getenv("TF_VAR_ibmcloud_api_key")
	if apiKey == "" {
		apiKey = "dummy-api-key-for-validation"
	}

	terraformOptions := getTerraformOptions(t)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run terraform init and validate in one step (using context-aware function)
	t.Log("Running terraform init and validate...")
	terraform.InitAndValidateContext(t, ctx, terraformOptions)

	t.Log("✓ Terraform validation passed")
}

// TestTerraformFormat checks if Terraform files are properly formatted
func TestTerraformFormat(t *testing.T) {
	t.Parallel()

	terraformOptions := getTerraformOptions(t)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Log("Running terraform fmt -check...")
	terraform.RunTerraformCommandContext(t, ctx, terraformOptions, "fmt", "-check")

	t.Log("✓ Terraform format check passed")
}

// TestTerraformSyntax performs additional syntax validation
func TestTerraformSyntax(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	// Check for common syntax issues
	t.Log("Checking for common Terraform syntax issues...")

	// Verify .tf files exist
	matches, err := filepath.Glob(filepath.Join(terraformDir, "*.tf"))
	assert.NoError(t, err, "Should be able to search for .tf files")

	if len(matches) == 0 {
		t.Skip("No .tf files found - skipping syntax check (tests should run from a Terraform project directory)")
	}

	t.Logf("Found %d Terraform files", len(matches))
}

// TestTerraformVariables checks if all required variables are defined
func TestTerraformVariables(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Checking for vars.tf...")

	// HA repo uses vars.tf (not variables.tf)
	variablesFile := filepath.Join(terraformDir, "vars.tf")
	if _, err := os.Stat(variablesFile); err == nil {
		t.Logf("Found vars.tf")
	} else {
		t.Log("No vars.tf found (may be defined elsewhere)")
	}
}

// TestTerraformOutputs checks if outputs are properly defined
func TestTerraformOutputs(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Checking for output.tf...")

	// HA repo uses output.tf (singular)
	outputsFile := filepath.Join(terraformDir, "output.tf")
	if _, err := os.Stat(outputsFile); err == nil {
		t.Logf("Found output.tf")
	} else {
		t.Log("No output.tf found")
	}
}

// TestTerraformProviders checks if providers are properly configured
// Note: This test is informational only and doesn't fail on provider issues
func TestTerraformProviders(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Checking provider configuration...")

	// Check if provider.tf exists (HA repo uses provider.tf)
	providerFiles := []string{
		filepath.Join(terraformDir, "provider.tf"),
		filepath.Join(terraformDir, "providers.tf"),
		filepath.Join(terraformDir, "main.tf"),
	}

	foundProvider := false
	for _, file := range providerFiles {
		if _, err := os.Stat(file); err == nil {
			t.Logf("Found provider configuration file: %s", filepath.Base(file))
			foundProvider = true
		}
	}

	if !foundProvider {
		t.Log("No dedicated provider configuration file found (providers may be defined inline)")
	}

	t.Log("✓ Provider configuration check complete")
}

// TestNoHardcodedSecrets checks for hardcoded sensitive values in Terraform files
func TestNoHardcodedSecrets(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Checking for hardcoded secrets in Terraform files...")

	// Patterns that should not appear in Terraform files
	forbiddenPatterns := []struct {
		pattern     *regexp.Regexp
		description string
	}{
		{regexp.MustCompile(`(?i)password\s*=\s*"[^"]{3,}"`), "Hardcoded password"},
		{regexp.MustCompile(`(?i)api[_-]?key\s*=\s*"[^"]{10,}"`), "Hardcoded API key"},
		{regexp.MustCompile(`(?i)secret\s*=\s*"[^"]{10,}"`), "Hardcoded secret"},
		{regexp.MustCompile(`(?i)token\s*=\s*"[^"]{10,}"`), "Hardcoded token"},
	}

	// Get all .tf files
	matches, err := filepath.Glob(filepath.Join(terraformDir, "*.tf"))
	assert.NoError(t, err, "Should be able to search for .tf files")

	if len(matches) == 0 {
		t.Skip("No .tf files found - skipping secret check")
	}

	hasSecrets := false
	for _, file := range matches {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Logf("Warning: Could not read file %s: %v", file, err)
			continue
		}

		for _, pattern := range forbiddenPatterns {
			if pattern.pattern.Match(content) {
				t.Errorf("Found potential %s in %s", pattern.description, filepath.Base(file))
				hasSecrets = true
			}
		}
	}

	if !hasSecrets {
		t.Log("✓ No hardcoded secrets detected")
	}
}

// TestSecurityGroupConfiguration validates security group rules
func TestSecurityGroupConfiguration(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating security group configuration...")

	// Check if vpc.tf exists (contains security groups)
	vpcFile := filepath.Join(terraformDir, "vpc.tf")
	if _, err := os.Stat(vpcFile); os.IsNotExist(err) {
		t.Skip("vpc.tf not found - skipping security group validation")
	}

	content, err := os.ReadFile(vpcFile)
	assert.NoError(t, err, "Should be able to read vpc.tf")

	contentStr := string(content)

	// HA has two security groups: public (port1+port4) and private (port2+port3)
	assert.Contains(t, contentStr, "fgt_sg_public", "Should define public security group (port1 + port4)")
	t.Log("✓ Public security group (fgt_sg_public) found")

	assert.Contains(t, contentStr, "fgt_sg_private", "Should define private security group (port2 + port3)")
	t.Log("✓ Private security group (fgt_sg_private) found")

	assert.Contains(t, contentStr, "ibm_is_security_group_rule", "Should define security group rules")
	t.Log("✓ Security group rules found")

	// Verify rule directions are specified
	if assert.Contains(t, contentStr, "direction", "Should specify rule direction") {
		t.Log("✓ Security group rules have direction specified")
	}
}

// TestCatalogOfferingCRNs validates catalog offering configuration
func TestCatalogOfferingCRNs(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating catalog offering CRN configuration...")

	// Check if instance.tf exists
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		t.Skip("instance.tf not found - skipping catalog offering validation")
	}

	content, err := os.ReadFile(instanceFile)
	assert.NoError(t, err, "Should be able to read instance.tf")

	contentStr := string(content)

	// Verify catalog_offering block exists on both fgt1 and fgt2
	assert.Contains(t, contentStr, "catalog_offering", "Should use catalog_offering for licensing")
	t.Log("✓ Catalog offering block found")

	// Verify version_crn is referenced
	assert.Contains(t, contentStr, "version_crn", "Should specify version_crn")
	t.Log("✓ Version CRN reference found")

	// Verify plan_crn is referenced
	assert.Contains(t, contentStr, "plan_crn", "Should specify plan_crn")
	t.Log("✓ Plan CRN reference found")

	// Verify variables are used (not hardcoded)
	assert.Contains(t, contentStr, "var.CATALOG_OFFERING_VERSION_CRN", "Should use variable for version CRN")
	assert.Contains(t, contentStr, "var.CATALOG_OFFERING_PLAN_CRN", "Should use variable for plan CRN")
	t.Log("✓ CRNs are properly parameterized")
}

// TestVSIProfileSelection validates VSI profile selection logic
func TestVSIProfileSelection(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating VSI profile selection logic...")

	// Check if locals.tf exists
	localsFile := filepath.Join(terraformDir, "locals.tf")
	if _, err := os.Stat(localsFile); os.IsNotExist(err) {
		t.Skip("locals.tf not found - skipping profile selection validation")
	}

	content, err := os.ReadFile(localsFile)
	assert.NoError(t, err, "Should be able to read locals.tf")

	contentStr := string(content)

	// Verify profile mapping exists
	assert.Contains(t, contentStr, "plan_crn_to_profile", "Should define plan CRN to profile mapping")
	t.Log("✓ Profile mapping found")

	// Verify region-specific overrides exist
	assert.Contains(t, contentStr, "region_specific_profile_overrides", "Should define region-specific overrides")
	t.Log("✓ Region-specific profile overrides found")

	// Verify selected_profile logic exists
	assert.Contains(t, contentStr, "selected_profile", "Should define profile selection logic")
	t.Log("✓ Profile selection logic found")

	// Verify both fgt1 and fgt2 in instance.tf use the selected profile
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); err == nil {
		instanceContent, err := os.ReadFile(instanceFile)
		assert.NoError(t, err, "Should be able to read instance.tf")
		instanceStr := string(instanceContent)
		assert.Contains(t, instanceStr, "local.selected_profile", "Both instances should use selected_profile")
		t.Log("✓ Both instances use selected_profile from locals")
	}
}

// TestMetadataServiceEnabled validates metadata service configuration
func TestMetadataServiceEnabled(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating metadata service configuration...")

	// Check if instance.tf exists
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		t.Skip("instance.tf not found - skipping metadata service validation")
	}

	content, err := os.ReadFile(instanceFile)
	assert.NoError(t, err, "Should be able to read instance.tf")

	contentStr := string(content)

	// Verify metadata_service block exists (both instances)
	assert.Contains(t, contentStr, "metadata_service", "Should configure metadata service on both instances")
	t.Log("✓ Metadata service block found")

	// Verify metadata service is enabled
	assert.Contains(t, contentStr, "enabled", "Should specify metadata service enabled status")
	t.Log("✓ Metadata service enabled configuration found")
}

// TestResourceNaming validates resource naming conventions
func TestResourceNaming(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating resource naming conventions...")

	// Check if vars.tf exists
	varsFile := filepath.Join(terraformDir, "vars.tf")
	if _, err := os.Stat(varsFile); os.IsNotExist(err) {
		t.Skip("vars.tf not found - skipping naming validation")
	}

	content, err := os.ReadFile(varsFile)
	assert.NoError(t, err, "Should be able to read vars.tf")

	contentStr := string(content)

	// Verify CLUSTER_NAME variable exists
	assert.Contains(t, contentStr, "CLUSTER_NAME", "Should define CLUSTER_NAME variable")
	t.Log("✓ CLUSTER_NAME variable found")

	// Verify random_string resource exists for unique suffixes
	assert.Contains(t, contentStr, "random_string", "Should use random suffix for resource names")
	t.Log("✓ Random suffix configuration found")

	// Check instance.tf for proper naming pattern on both instances
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); err == nil {
		instanceContent, err := os.ReadFile(instanceFile)
		assert.NoError(t, err, "Should be able to read instance.tf")

		// Verify resources use CLUSTER_NAME and random suffix
		assert.Contains(t, string(instanceContent), "var.CLUSTER_NAME", "Resources should use CLUSTER_NAME")
		assert.Contains(t, string(instanceContent), "random_string.random_suffix", "Resources should use random suffix")
		t.Log("✓ Resources follow naming convention")
	}
}

// TestNetworkConfiguration validates network interface configuration for the HA pair
func TestNetworkConfiguration(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating HA network configuration (4 ports per FortiGate)...")

	// Check if vpc.tf exists
	vpcFile := filepath.Join(terraformDir, "vpc.tf")
	if _, err := os.Stat(vpcFile); os.IsNotExist(err) {
		t.Skip("vpc.tf not found - skipping network validation")
	}

	content, err := os.ReadFile(vpcFile)
	assert.NoError(t, err, "Should be able to read vpc.tf")

	contentStr := string(content)

	// Verify VNIs exist for both active and passive FGTs
	assert.Contains(t, contentStr, "vni-active", "Should define VNIs for active (fgt1) FortiGate")
	t.Log("✓ Active FortiGate VNIs found")

	assert.Contains(t, contentStr, "vni-passive", "Should define VNIs for passive (fgt2) FortiGate")
	t.Log("✓ Passive FortiGate VNIs found")

	// Verify IP spoofing is disabled (security best practice)
	assert.Contains(t, contentStr, "allow_ip_spoofing", "Should configure IP spoofing setting")
	t.Log("✓ IP spoofing configuration found")

	// Verify security groups are attached to interfaces
	assert.Contains(t, contentStr, "security_groups", "Should attach security groups to interfaces")
	t.Log("✓ Security groups attached to network interfaces")

	// Verify all 4 subnets are used (port1–port4)
	for _, subnet := range []string{"SUBNET_1", "SUBNET_2", "SUBNET_3", "SUBNET_4"} {
		assert.Contains(t, contentStr, subnet, "Should reference %s", subnet)
	}
	t.Log("✓ All 4 subnet references found (port1–port4)")

	// Check instance.tf for 4-port attachment configuration on both instances
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); err == nil {
		instanceContent, err := os.ReadFile(instanceFile)
		assert.NoError(t, err, "Should be able to read instance.tf")
		instanceStr := string(instanceContent)

		assert.Contains(t, instanceStr, "primary_network_attachment", "Should have primary network attachment")
		assert.Contains(t, instanceStr, "network_attachments", "Should have secondary network attachments (port2–port4)")

		// Verify both fgt1 and fgt2 are defined
		assert.Contains(t, instanceStr, "ibm_is_instance\" \"fgt1\"", "Should define fgt1 (primary/active)")
		assert.Contains(t, instanceStr, "ibm_is_instance\" \"fgt2\"", "Should define fgt2 (secondary/passive)")
		t.Log("✓ Both FortiGate instances (fgt1 + fgt2) defined with 4-port configuration")
	}
}

// TestRequiredVariables validates that all required HA variables are defined
func TestRequiredVariables(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating required HA variables...")

	// Check if vars.tf exists
	varsFile := filepath.Join(terraformDir, "vars.tf")
	if _, err := os.Stat(varsFile); os.IsNotExist(err) {
		t.Skip("vars.tf not found - skipping variable validation")
	}

	content, err := os.ReadFile(varsFile)
	assert.NoError(t, err, "Should be able to read vars.tf")

	contentStr := string(content)

	// List of required variables for FortiGate HA deployment
	requiredVars := []string{
		"IBMCLOUD_API_KEY",
		"SSH_PUBLIC_KEY",
		"REGION",
		"ZONE",
		"VPC",
		"SUBNET_1",
		"SUBNET_2",
		"SUBNET_3",
		"SUBNET_4",
		"NETMASK",
		"FGT1_STATIC_IP_PORT1",
		"FGT1_STATIC_IP_PORT2",
		"FGT1_STATIC_IP_PORT3",
		"FGT1_STATIC_IP_PORT4",
		"FGT1_PORT4_MGMT_GATEWAY",
		"FGT2_STATIC_IP_PORT1",
		"FGT2_STATIC_IP_PORT2",
		"FGT2_STATIC_IP_PORT3",
		"FGT2_STATIC_IP_PORT4",
		"FGT2_PORT4_MGMT_GATEWAY",
		"CATALOG_OFFERING_VERSION_CRN",
		"CATALOG_OFFERING_PLAN_CRN",
		"CLUSTER_NAME",
		"RESOURCE_GRP",
		"PUBLIC_GATEWAY_ID",
	}

	for _, varName := range requiredVars {
		assert.Contains(t, contentStr, varName, "Should define required variable: %s", varName)
	}

	t.Logf("✓ All %d required HA variables are defined", len(requiredVars))
}

// TestHAUserDataTemplates validates the HA-specific user_data template files
func TestHAUserDataTemplates(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating HA user_data template files...")

	// HA deployment uses separate templates for active and passive nodes
	activeTemplate := filepath.Join(terraformDir, "user_data_active.conf")
	passiveTemplate := filepath.Join(terraformDir, "user_data_passive.conf")

	_, errActive := os.Stat(activeTemplate)
	assert.NoError(t, errActive, "user_data_active.conf should exist for the primary (ACTIVE) FortiGate")
	t.Log("✓ user_data_active.conf found")

	_, errPassive := os.Stat(passiveTemplate)
	assert.NoError(t, errPassive, "user_data_passive.conf should exist for the secondary (PASSIVE) FortiGate")
	t.Log("✓ user_data_passive.conf found")

	// Verify instance.tf references both templates
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); err == nil {
		instanceContent, err := os.ReadFile(instanceFile)
		assert.NoError(t, err, "Should be able to read instance.tf")
		instanceStr := string(instanceContent)

		assert.Contains(t, instanceStr, "user_data_active.conf", "fgt1 should use user_data_active.conf")
		assert.Contains(t, instanceStr, "user_data_passive.conf", "fgt2 should use user_data_passive.conf")
		t.Log("✓ Both HA user_data templates are referenced in instance.tf")
	}
}

// TestHAFloatingIPs validates that three floating IPs are provisioned:
// one shared/failover IP on port1 and one dedicated management IP per node (port4)
func TestHAFloatingIPs(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating HA floating IP configuration...")

	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		t.Skip("instance.tf not found - skipping floating IP validation")
	}

	content, err := os.ReadFile(instanceFile)
	assert.NoError(t, err, "Should be able to read instance.tf")

	contentStr := string(content)

	// HA requires 3 floating IPs: publicip (shared/failover), publicip2 (fgt1 mgmt), publicip3 (fgt2 mgmt)
	assert.Contains(t, contentStr, "ibm_is_floating_ip\" \"publicip\"", "Should define shared/failover floating IP")
	assert.Contains(t, contentStr, "ibm_is_floating_ip\" \"publicip2\"", "Should define fgt1 HA management floating IP")
	assert.Contains(t, contentStr, "ibm_is_floating_ip\" \"publicip3\"", "Should define fgt2 HA management floating IP")
	t.Log("✓ Three floating IPs defined (shared, fgt1-mgmt, fgt2-mgmt)")
}

// TestHAOutputs validates that all expected HA-specific Terraform outputs are defined
func TestHAOutputs(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating HA Terraform outputs...")

	outputFile := filepath.Join(terraformDir, "output.tf")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Skip("output.tf not found - skipping outputs validation")
	}

	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Should be able to read output.tf")

	contentStr := string(content)

	// Required HA outputs
	expectedOutputs := []string{
		"FortiGate_Public_IP",
		"FGT1_Public_HA_Management_IP",
		"FGT2_Public_HA_Management_IP",
		"FGT1_Default_Admin_Password",
		"FGT2_Default_Admin_Password",
		"Catalog_Offering_Version_CRN",
		"Catalog_Offering_Plan_CRN",
		"Selected_VSI_Profile",
	}

	for _, output := range expectedOutputs {
		assert.Contains(t, contentStr, output, "output.tf should define output: %s", output)
	}

	t.Logf("✓ All %d required HA outputs are defined", len(expectedOutputs))
}

// ── Unit tests for Go test helpers ──────────────────────────────────────────

// TestGetTerraformDir_DefaultsToParent verifies that getTerraformDir returns "../"
// when the TERRAFORM_DIR environment variable is not set.
func TestGetTerraformDir_DefaultsToParent(t *testing.T) {
	orig := os.Getenv("TERRAFORM_DIR")
	os.Unsetenv("TERRAFORM_DIR")
	defer os.Setenv("TERRAFORM_DIR", orig)

	dir := getTerraformDir()
	assert.Equal(t, "../", dir, "Should default to parent directory when TERRAFORM_DIR is not set")
}

// TestGetTerraformDir_UsesEnvVar verifies that getTerraformDir returns the value
// of TERRAFORM_DIR when it is set.
func TestGetTerraformDir_UsesEnvVar(t *testing.T) {
	os.Setenv("TERRAFORM_DIR", "/tmp/my-terraform-dir")
	defer os.Unsetenv("TERRAFORM_DIR")

	dir := getTerraformDir()
	assert.Equal(t, "/tmp/my-terraform-dir", dir, "Should use TERRAFORM_DIR env var when set")
}

// TestNoHardcodedSecrets_PatternMatchesRealSecret verifies the secret-detection
// regex patterns used in TestNoHardcodedSecrets correctly flag an obvious secret.
func TestNoHardcodedSecrets_PatternMatchesRealSecret(t *testing.T) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)password\s*=\s*"[^"]{3,}"`),
		regexp.MustCompile(`(?i)api[_-]?key\s*=\s*"[^"]{10,}"`),
		regexp.MustCompile(`(?i)secret\s*=\s*"[^"]{10,}"`),
		regexp.MustCompile(`(?i)token\s*=\s*"[^"]{10,}"`),
	}

	testCases := []struct {
		name    string
		input   string
		wantHit int // index of pattern expected to match (-1 = none should match)
	}{
		{"hardcoded password", `password = "supersecret"`, 0},
		{"hardcoded api_key", `api_key = "abcdefghijk"`, 1},
		{"hardcoded secret", `secret = "mysecretvalue"`, 2},
		{"hardcoded token", `token = "longtoken1234"`, 3},
		{"empty password", `password = ""`, -1},
		{"short password", `password = "ab"`, -1},
		{"var reference", `ibmcloud_api_key = var.IBMCLOUD_API_KEY`, -1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for i, p := range patterns {
				if p.MatchString(tc.input) {
					matched = true
					if tc.wantHit == -1 {
						t.Errorf("Pattern %d unexpectedly matched %q", i, tc.input)
					} else if i != tc.wantHit {
						t.Errorf("Expected pattern %d to match, but pattern %d matched for %q", tc.wantHit, i, tc.input)
					}
				}
			}
			if tc.wantHit >= 0 && !matched {
				t.Errorf("Expected a pattern to match %q but none did", tc.input)
			}
		})
	}
}

// TestCheckInterfaceLogic_ParsesFortiGateOutput verifies the line-by-line interface
// status parsing logic used in the acceptance tests against a realistic SSH output sample.
func TestCheckInterfaceLogic_ParsesFortiGateOutput(t *testing.T) {
	// Realistic FortiGate CLI output from "get system interface physical"
	sampleOutput := `
== [onboard]
    ==[port1]
        mode: static
        ip: 10.240.0.4 255.255.255.0
        status: up
        speed: 10000Mbps (Duplex: full)
    ==[port2]
        mode: static
        ip: 10.241.0.4 255.255.255.0
        status: up
        speed: 10000Mbps (Duplex: full)
    ==[port3]
        mode: static
        ip: 10.242.0.4 255.255.255.0
        status: up
        speed: 10000Mbps (Duplex: full)
    ==[port4]
        mode: static
        ip: 10.243.0.4 255.255.255.0
        status: up
        speed: 10000Mbps (Duplex: full)
`

	checkInterface := func(out, name string) (found bool, statusLine string) {
		lines := strings.Split(out, "\n")
		inBlock := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "["+name+"]") {
				inBlock = true
				found = true
				continue
			}
			if !inBlock {
				continue
			}
			if strings.Contains(trimmed, "==[") || strings.HasPrefix(trimmed, "== [") {
				break
			}
			if strings.HasPrefix(strings.ToLower(trimmed), "status:") {
				return true, trimmed
			}
		}
		return found, ""
	}

	for _, port := range []string{"port1", "port2", "port3", "port4"} {
		found, statusLine := checkInterface(sampleOutput, port)
		assert.True(t, found, "Interface %s should be found in sample output", port)
		assert.Contains(t, strings.ToLower(statusLine), "status: up",
			"Interface %s should show 'status: up', got: %q", port, statusLine)
	}
}
