package verify_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/beduardo/eve-realm-mcp/deploy/k8s/verify"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

// mockKubeClient implements verify.KubeClient for use in tests.
type mockKubeClient struct {
	deployment *verify.DeploymentInfo
	deployErr  error
	service    *verify.ServiceInfo
	serviceErr error
	configmap  *verify.ConfigMapInfo
	configErr  error
}

func (m *mockKubeClient) GetDeployment(ctx context.Context, namespace, name string) (*verify.DeploymentInfo, error) {
	return m.deployment, m.deployErr
}

func (m *mockKubeClient) GetService(ctx context.Context, namespace, name string) (*verify.ServiceInfo, error) {
	return m.service, m.serviceErr
}

func (m *mockKubeClient) GetConfigMap(ctx context.Context, namespace, name string) (*verify.ConfigMapInfo, error) {
	return m.configmap, m.configErr
}

// mockHTTPClient implements verify.HTTPClient for use in tests.
type mockHTTPClient struct {
	statusCode int
	err        error
}

func (m *mockHTTPClient) Get(url string) (int, error) {
	return m.statusCode, m.err
}

// ---------------------------------------------------------------------------
// TestCheckDeploymentReady
// ---------------------------------------------------------------------------

func TestCheckDeploymentReady_Success(t *testing.T) {
	client := &mockKubeClient{
		deployment: &verify.DeploymentInfo{
			DesiredReplicas: 1,
			ReadyReplicas:   1,
		},
	}

	fn := verify.CheckDeploymentReady(client)
	result := fn(context.Background())

	if !result.Passed {
		t.Errorf("expected Passed=true, got Passed=false; message: %s", result.Message)
	}
	if result.Category != "infrastructure" {
		t.Errorf("expected Category=%q, got %q", "infrastructure", result.Category)
	}
	if result.Name == "" {
		t.Error("expected non-empty Name")
	}
}

func TestCheckDeploymentReady_NotFound(t *testing.T) {
	client := &mockKubeClient{
		deployErr: errors.New("deployment not found"),
	}

	fn := verify.CheckDeploymentReady(client)
	result := fn(context.Background())

	if result.Passed {
		t.Error("expected Passed=false when deployment not found")
	}
	if result.Category != "infrastructure" {
		t.Errorf("expected Category=%q, got %q", "infrastructure", result.Category)
	}
	if !strings.Contains(result.Message, "eve-realm-mcp") {
		t.Errorf("error message should contain service name %q, got: %s", "eve-realm-mcp", result.Message)
	}
}

func TestCheckDeploymentReady_ReplicasMismatch(t *testing.T) {
	cases := []struct {
		name            string
		desired         int32
		ready           int32
		expectPassFail  bool
		expectMsgSubstr string
	}{
		{
			name:            "zero_ready_replicas",
			desired:         1,
			ready:           0,
			expectPassFail:  false,
			expectMsgSubstr: "eve-realm-mcp",
		},
		{
			name:            "partial_ready_replicas",
			desired:         3,
			ready:           1,
			expectPassFail:  false,
			expectMsgSubstr: "eve-realm-mcp",
		},
		{
			name:           "all_replicas_ready",
			desired:        2,
			ready:          2,
			expectPassFail: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockKubeClient{
				deployment: &verify.DeploymentInfo{
					DesiredReplicas: tc.desired,
					ReadyReplicas:   tc.ready,
				},
			}

			fn := verify.CheckDeploymentReady(client)
			result := fn(context.Background())

			if result.Passed != tc.expectPassFail {
				t.Errorf("Passed=%v, want %v; message: %s", result.Passed, tc.expectPassFail, result.Message)
			}
			if !tc.expectPassFail && !strings.Contains(result.Message, tc.expectMsgSubstr) {
				t.Errorf("error message should contain %q, got: %s", tc.expectMsgSubstr, result.Message)
			}
		})
	}
}

func TestCheckDeploymentReady_DescriptiveError(t *testing.T) {
	client := &mockKubeClient{
		deployment: &verify.DeploymentInfo{
			DesiredReplicas: 1,
			ReadyReplicas:   0,
		},
	}

	fn := verify.CheckDeploymentReady(client)
	result := fn(context.Background())

	// Error must include service name, namespace, and expected vs actual.
	if !strings.Contains(result.Message, "eve-realm-mcp") {
		t.Errorf("message should contain service name %q; got: %s", "eve-realm-mcp", result.Message)
	}
	if !strings.Contains(result.Message, "eve-realm") {
		t.Errorf("message should contain namespace %q; got: %s", "eve-realm", result.Message)
	}
}

// ---------------------------------------------------------------------------
// TestCheckServiceExists
// ---------------------------------------------------------------------------

func TestCheckServiceExists_Success(t *testing.T) {
	client := &mockKubeClient{
		service: &verify.ServiceInfo{
			Ports: []int{8080, 50051},
		},
	}

	fn := verify.CheckServiceExists(client)
	result := fn(context.Background())

	if !result.Passed {
		t.Errorf("expected Passed=true, got Passed=false; message: %s", result.Message)
	}
	if result.Category != "infrastructure" {
		t.Errorf("expected Category=%q, got %q", "infrastructure", result.Category)
	}
}

func TestCheckServiceExists_NotFound(t *testing.T) {
	client := &mockKubeClient{
		serviceErr: errors.New("service not found"),
	}

	fn := verify.CheckServiceExists(client)
	result := fn(context.Background())

	if result.Passed {
		t.Error("expected Passed=false when service not found")
	}
	if !strings.Contains(result.Message, "eve-realm-mcp") {
		t.Errorf("error message should contain service name %q, got: %s", "eve-realm-mcp", result.Message)
	}
}

func TestCheckServiceExists_MissingPorts(t *testing.T) {
	cases := []struct {
		name            string
		ports           []int
		expectPassFail  bool
		expectMsgSubstr string
	}{
		{
			name:            "missing_port_8080",
			ports:           []int{50051},
			expectPassFail:  false,
			expectMsgSubstr: "8080",
		},
		{
			name:            "missing_port_50051",
			ports:           []int{8080},
			expectPassFail:  false,
			expectMsgSubstr: "50051",
		},
		{
			name:           "both_ports_present",
			ports:          []int{8080, 50051},
			expectPassFail: true,
		},
		{
			name:           "extra_ports_still_passes",
			ports:          []int{8080, 50051, 9090},
			expectPassFail: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockKubeClient{
				service: &verify.ServiceInfo{Ports: tc.ports},
			}

			fn := verify.CheckServiceExists(client)
			result := fn(context.Background())

			if result.Passed != tc.expectPassFail {
				t.Errorf("Passed=%v, want %v; message: %s", result.Passed, tc.expectPassFail, result.Message)
			}
			if !tc.expectPassFail && !strings.Contains(result.Message, tc.expectMsgSubstr) {
				t.Errorf("error message should contain %q, got: %s", tc.expectMsgSubstr, result.Message)
			}
		})
	}
}

func TestCheckServiceExists_DescriptiveError(t *testing.T) {
	client := &mockKubeClient{
		service: &verify.ServiceInfo{Ports: []int{9999}},
	}

	fn := verify.CheckServiceExists(client)
	result := fn(context.Background())

	// Error must include service name, namespace, and expected vs actual port.
	if !strings.Contains(result.Message, "eve-realm-mcp") {
		t.Errorf("message should contain service name %q; got: %s", "eve-realm-mcp", result.Message)
	}
	if !strings.Contains(result.Message, "eve-realm") {
		t.Errorf("message should contain namespace %q; got: %s", "eve-realm", result.Message)
	}
}

// ---------------------------------------------------------------------------
// TestCheckHealthz
// ---------------------------------------------------------------------------

func TestCheckHealthz_Success(t *testing.T) {
	client := &mockHTTPClient{statusCode: 200}

	fn := verify.CheckHealthz(client)
	result := fn(context.Background())

	if !result.Passed {
		t.Errorf("expected Passed=true, got Passed=false; message: %s", result.Message)
	}
	if result.Category != "health" {
		t.Errorf("expected Category=%q, got %q", "health", result.Category)
	}
}

func TestCheckHealthz_NonOKStatus(t *testing.T) {
	cases := []struct {
		name           string
		statusCode     int
		expectPassFail bool
	}{
		{name: "status_500", statusCode: 500, expectPassFail: false},
		{name: "status_404", statusCode: 404, expectPassFail: false},
		{name: "status_503", statusCode: 503, expectPassFail: false},
		{name: "status_200", statusCode: 200, expectPassFail: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockHTTPClient{statusCode: tc.statusCode}

			fn := verify.CheckHealthz(client)
			result := fn(context.Background())

			if result.Passed != tc.expectPassFail {
				t.Errorf("Passed=%v, want %v; message: %s", result.Passed, tc.expectPassFail, result.Message)
			}
		})
	}
}

func TestCheckHealthz_RequestError(t *testing.T) {
	client := &mockHTTPClient{err: errors.New("connection refused")}

	fn := verify.CheckHealthz(client)
	result := fn(context.Background())

	if result.Passed {
		t.Error("expected Passed=false when HTTP request fails")
	}
	if !strings.Contains(result.Message, "/healthz") {
		t.Errorf("error message should contain endpoint %q; got: %s", "/healthz", result.Message)
	}
}

func TestCheckHealthz_DescriptiveError(t *testing.T) {
	client := &mockHTTPClient{statusCode: 503}

	fn := verify.CheckHealthz(client)
	result := fn(context.Background())

	// Error must include endpoint and expected vs actual status code.
	if !strings.Contains(result.Message, "/healthz") {
		t.Errorf("message should contain endpoint %q; got: %s", "/healthz", result.Message)
	}
	if !strings.Contains(result.Message, "200") {
		t.Errorf("message should contain expected status %q; got: %s", "200", result.Message)
	}
	if !strings.Contains(result.Message, "503") {
		t.Errorf("message should contain actual status %q; got: %s", "503", result.Message)
	}
}

// ---------------------------------------------------------------------------
// TestCheckReadyz
// ---------------------------------------------------------------------------

func TestCheckReadyz_Success(t *testing.T) {
	client := &mockHTTPClient{statusCode: 200}

	fn := verify.CheckReadyz(client)
	result := fn(context.Background())

	if !result.Passed {
		t.Errorf("expected Passed=true, got Passed=false; message: %s", result.Message)
	}
	if result.Category != "health" {
		t.Errorf("expected Category=%q, got %q", "health", result.Category)
	}
}

func TestCheckReadyz_NonOKStatus(t *testing.T) {
	cases := []struct {
		name           string
		statusCode     int
		expectPassFail bool
	}{
		{name: "status_500", statusCode: 500, expectPassFail: false},
		{name: "status_404", statusCode: 404, expectPassFail: false},
		{name: "status_503", statusCode: 503, expectPassFail: false},
		{name: "status_200", statusCode: 200, expectPassFail: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockHTTPClient{statusCode: tc.statusCode}

			fn := verify.CheckReadyz(client)
			result := fn(context.Background())

			if result.Passed != tc.expectPassFail {
				t.Errorf("Passed=%v, want %v; message: %s", result.Passed, tc.expectPassFail, result.Message)
			}
		})
	}
}

func TestCheckReadyz_RequestError(t *testing.T) {
	client := &mockHTTPClient{err: errors.New("connection refused")}

	fn := verify.CheckReadyz(client)
	result := fn(context.Background())

	if result.Passed {
		t.Error("expected Passed=false when HTTP request fails")
	}
	if !strings.Contains(result.Message, "/readyz") {
		t.Errorf("error message should contain endpoint %q; got: %s", "/readyz", result.Message)
	}
}

func TestCheckReadyz_DescriptiveError(t *testing.T) {
	client := &mockHTTPClient{statusCode: 503}

	fn := verify.CheckReadyz(client)
	result := fn(context.Background())

	// Error must include endpoint and expected vs actual status code.
	if !strings.Contains(result.Message, "/readyz") {
		t.Errorf("message should contain endpoint %q; got: %s", "/readyz", result.Message)
	}
	if !strings.Contains(result.Message, "200") {
		t.Errorf("message should contain expected status %q; got: %s", "200", result.Message)
	}
	if !strings.Contains(result.Message, "503") {
		t.Errorf("message should contain actual status %q; got: %s", "503", result.Message)
	}
}

// ---------------------------------------------------------------------------
// TestCheckConfigMapInjected
// ---------------------------------------------------------------------------

func TestCheckConfigMapInjected_Success(t *testing.T) {
	client := &mockKubeClient{
		configmap: &verify.ConfigMapInfo{
			Keys: []string{"NATS_URL", "LOG_LEVEL"},
		},
	}

	fn := verify.CheckConfigMapInjected(client)
	result := fn(context.Background())

	if !result.Passed {
		t.Errorf("expected Passed=true, got Passed=false; message: %s", result.Message)
	}
	if result.Category != "configmap" {
		t.Errorf("expected Category=%q, got %q", "configmap", result.Category)
	}
}

func TestCheckConfigMapInjected_NotFound(t *testing.T) {
	client := &mockKubeClient{
		configErr: errors.New("configmap not found"),
	}

	fn := verify.CheckConfigMapInjected(client)
	result := fn(context.Background())

	if result.Passed {
		t.Error("expected Passed=false when configmap not found")
	}
	if !strings.Contains(result.Message, "eve-realm-config") {
		t.Errorf("error message should contain configmap name %q, got: %s", "eve-realm-config", result.Message)
	}
}

func TestCheckConfigMapInjected_EmptyKeys(t *testing.T) {
	client := &mockKubeClient{
		configmap: &verify.ConfigMapInfo{
			Keys: []string{},
		},
	}

	fn := verify.CheckConfigMapInjected(client)
	result := fn(context.Background())

	if result.Passed {
		t.Error("expected Passed=false when configmap has no keys")
	}
	if !strings.Contains(result.Message, "eve-realm-config") {
		t.Errorf("error message should contain configmap name %q; got: %s", "eve-realm-config", result.Message)
	}
}

func TestCheckConfigMapInjected_WithKeys(t *testing.T) {
	cases := []struct {
		name           string
		keys           []string
		expectPassFail bool
	}{
		{name: "no_keys", keys: []string{}, expectPassFail: false},
		{name: "one_key", keys: []string{"NATS_URL"}, expectPassFail: true},
		{name: "multiple_keys", keys: []string{"NATS_URL", "LOG_LEVEL", "PORT"}, expectPassFail: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockKubeClient{
				configmap: &verify.ConfigMapInfo{Keys: tc.keys},
			}

			fn := verify.CheckConfigMapInjected(client)
			result := fn(context.Background())

			if result.Passed != tc.expectPassFail {
				t.Errorf("Passed=%v, want %v; message: %s", result.Passed, tc.expectPassFail, result.Message)
			}
		})
	}
}

func TestCheckConfigMapInjected_DescriptiveError(t *testing.T) {
	client := &mockKubeClient{
		configmap: &verify.ConfigMapInfo{Keys: []string{}},
	}

	fn := verify.CheckConfigMapInjected(client)
	result := fn(context.Background())

	// Error must include configmap name and namespace.
	if !strings.Contains(result.Message, "eve-realm-config") {
		t.Errorf("message should contain configmap name %q; got: %s", "eve-realm-config", result.Message)
	}
	if !strings.Contains(result.Message, "eve-realm") {
		t.Errorf("message should contain namespace %q; got: %s", "eve-realm", result.Message)
	}
}

// ---------------------------------------------------------------------------
// TestChecks registration
// ---------------------------------------------------------------------------

func TestChecks_AllRegistered(t *testing.T) {
	if len(verify.Checks) != 5 {
		t.Errorf("expected 5 registered checks, got %d", len(verify.Checks))
	}
}

func TestChecks_CategoryCounts(t *testing.T) {
	categories := make(map[string]int)
	for _, c := range verify.Checks {
		categories[c.Category]++
	}

	cases := []struct {
		category string
		count    int
	}{
		{category: "infrastructure", count: 2},
		{category: "health", count: 2},
		{category: "configmap", count: 1},
	}

	for _, tc := range cases {
		t.Run(tc.category, func(t *testing.T) {
			got := categories[tc.category]
			if got != tc.count {
				t.Errorf("category %q: expected %d checks, got %d", tc.category, tc.count, got)
			}
		})
	}
}

func TestChecks_AllHaveNames(t *testing.T) {
	for i, c := range verify.Checks {
		if c.Name == "" {
			t.Errorf("Checks[%d] has empty Name", i)
		}
		if c.Category == "" {
			t.Errorf("Checks[%d] has empty Category", i)
		}
		if c.Fn == nil {
			t.Errorf("Checks[%d] (Name=%q) has nil Fn", i, c.Name)
		}
	}
}

func TestChecks_NamesAreUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, c := range verify.Checks {
		if seen[c.Name] {
			t.Errorf("duplicate check name %q", c.Name)
		}
		seen[c.Name] = true
	}
}
