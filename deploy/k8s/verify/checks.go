// Package verify provides cluster integration check functions for the
// eve-realm-mcp deployment. Each check conforms to the CheckFunc contract
// and is registered in the Checks slice for use by make verify-cluster.
//
// All checks are idempotent and read-only — they do not modify cluster state.
// Each check completes within 10 seconds when the context has no tighter
// deadline set by the caller.
package verify

import (
	"context"
	"fmt"
)

// CheckResult is the outcome of a single cluster check. Name and Category
// identify the check; Passed indicates success; Message contains a
// human-readable description of the result (or a descriptive error on failure).
type CheckResult struct {
	Name     string
	Category string
	Passed   bool
	Message  string
}

// CheckFunc is the signature every cluster check must satisfy. The function
// must be idempotent and must not modify cluster state.
type CheckFunc func(ctx context.Context) CheckResult

// CheckRegistration pairs a check with its human-readable name and category
// label. Checks is the authoritative list consumed by make verify-cluster.
type CheckRegistration struct {
	Name     string
	Category string
	Fn       CheckFunc
}

// ---------------------------------------------------------------------------
// Client interfaces (defined at the consumption site for testability)
// ---------------------------------------------------------------------------

// KubeClient is the narrow interface through which check functions query the
// Kubernetes API. I/O is mocked via this interface in tests.
type KubeClient interface {
	GetDeployment(ctx context.Context, namespace, name string) (*DeploymentInfo, error)
	GetService(ctx context.Context, namespace, name string) (*ServiceInfo, error)
	GetConfigMap(ctx context.Context, namespace, name string) (*ConfigMapInfo, error)
}

// HTTPClient is the narrow interface through which check functions issue HTTP
// requests. I/O is mocked via this interface in tests.
type HTTPClient interface {
	// Get issues an HTTP GET to url and returns the response status code.
	Get(url string) (statusCode int, err error)
}

// ---------------------------------------------------------------------------
// Minimal info structs — only the fields the checks need
// ---------------------------------------------------------------------------

// DeploymentInfo carries the subset of Deployment fields required for the
// deployment-readiness check.
type DeploymentInfo struct {
	DesiredReplicas int32
	ReadyReplicas   int32
}

// ServiceInfo carries the subset of Service fields required for the service
// existence check.
type ServiceInfo struct {
	// Ports is the list of port numbers exposed by the Service.
	Ports []int
}

// ConfigMapInfo carries the subset of ConfigMap fields required for the
// configmap-injection check.
type ConfigMapInfo struct {
	// Keys is the list of keys present in the ConfigMap's data.
	Keys []string
}

// ---------------------------------------------------------------------------
// Check functions
// ---------------------------------------------------------------------------

const (
	namespace     = "eve-realm"
	deployName    = "eve-realm-mcp"
	serviceName   = "eve-realm-mcp"
	configMapName = "eve-realm-config"
	healthPort    = 8080
)

// CheckDeploymentReady returns a CheckFunc that verifies the eve-realm-mcp
// Deployment exists in the eve-realm namespace and all desired replicas are
// ready.
//
// Category: infrastructure
func CheckDeploymentReady(client KubeClient) CheckFunc {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:     "deployment-ready",
			Category: "infrastructure",
		}

		info, err := client.GetDeployment(ctx, namespace, deployName)
		if err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"deployment %q in namespace %q not found: %v",
				deployName, namespace, err,
			)
			return result
		}

		if info.ReadyReplicas < info.DesiredReplicas {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"deployment %q in namespace %q: expected %d ready replica(s), got %d",
				deployName, namespace, info.DesiredReplicas, info.ReadyReplicas,
			)
			return result
		}

		result.Passed = true
		result.Message = fmt.Sprintf(
			"deployment %q in namespace %q: %d/%d replicas ready",
			deployName, namespace, info.ReadyReplicas, info.DesiredReplicas,
		)
		return result
	}
}

// CheckServiceExists returns a CheckFunc that verifies the eve-realm-mcp
// Service exists in the eve-realm namespace and exposes both port 8080 (HTTP)
// and port 50051 (gRPC).
//
// Category: infrastructure
func CheckServiceExists(client KubeClient) CheckFunc {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:     "service-exists",
			Category: "infrastructure",
		}

		info, err := client.GetService(ctx, namespace, serviceName)
		if err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"service %q in namespace %q not found: %v",
				serviceName, namespace, err,
			)
			return result
		}

		requiredPorts := []int{8080, 50051}
		portSet := make(map[int]bool, len(info.Ports))
		for _, p := range info.Ports {
			portSet[p] = true
		}

		for _, want := range requiredPorts {
			if !portSet[want] {
				result.Passed = false
				result.Message = fmt.Sprintf(
					"service %q in namespace %q: required port %d not found (exposed ports: %v)",
					serviceName, namespace, want, info.Ports,
				)
				return result
			}
		}

		result.Passed = true
		result.Message = fmt.Sprintf(
			"service %q in namespace %q: ports 8080 and 50051 present",
			serviceName, namespace,
		)
		return result
	}
}

// CheckHealthz returns a CheckFunc that issues an HTTP GET to /healthz on
// port 8080 of the MCP Server and asserts a 200 status code.
//
// Category: health
func CheckHealthz(client HTTPClient) CheckFunc {
	return func(ctx context.Context) CheckResult {
		const endpoint = "/healthz"
		result := CheckResult{
			Name:     "healthz",
			Category: "health",
		}

		url := fmt.Sprintf("http://localhost:%d%s", healthPort, endpoint)
		statusCode, err := client.Get(url)
		if err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"GET %s failed: %v",
				endpoint, err,
			)
			return result
		}

		if statusCode != 200 {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"GET %s: expected status 200, got %d",
				endpoint, statusCode,
			)
			return result
		}

		result.Passed = true
		result.Message = fmt.Sprintf("GET %s returned 200", endpoint)
		return result
	}
}

// CheckReadyz returns a CheckFunc that issues an HTTP GET to /readyz on
// port 8080 of the MCP Server and asserts a 200 status code.
//
// Category: health
func CheckReadyz(client HTTPClient) CheckFunc {
	const endpoint = "/readyz"
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:     "readyz",
			Category: "health",
		}

		url := fmt.Sprintf("http://localhost:%d%s", healthPort, endpoint)
		statusCode, err := client.Get(url)
		if err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"GET %s failed: %v",
				endpoint, err,
			)
			return result
		}

		if statusCode != 200 {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"GET %s: expected status 200, got %d",
				endpoint, statusCode,
			)
			return result
		}

		result.Passed = true
		result.Message = fmt.Sprintf("GET %s returned 200", endpoint)
		return result
	}
}

// CheckConfigMapInjected returns a CheckFunc that verifies the eve-realm-config
// ConfigMap exists in the eve-realm namespace and contains at least one key
// (confirming it is populated and available for envFrom injection).
//
// Category: configmap
func CheckConfigMapInjected(client KubeClient) CheckFunc {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:     "configmap-injected",
			Category: "configmap",
		}

		info, err := client.GetConfigMap(ctx, namespace, configMapName)
		if err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"configmap %q in namespace %q not found: %v",
				configMapName, namespace, err,
			)
			return result
		}

		if len(info.Keys) == 0 {
			result.Passed = false
			result.Message = fmt.Sprintf(
				"configmap %q in namespace %q exists but contains no keys; expected at least 1 key for envFrom injection",
				configMapName, namespace,
			)
			return result
		}

		result.Passed = true
		result.Message = fmt.Sprintf(
			"configmap %q in namespace %q present with %d key(s)",
			configMapName, namespace, len(info.Keys),
		)
		return result
	}
}

// ---------------------------------------------------------------------------
// Check registry
// ---------------------------------------------------------------------------

// defaultKubeClient is a nil KubeClient placeholder. Real-cluster runs replace
// this with an actual implementation (e.g., a kubectl-backed client).
// Unit tests inject mocks directly via the constructor functions above.
var defaultKubeClient KubeClient

// defaultHTTPClient is a nil HTTPClient placeholder. Real-cluster runs replace
// this with an actual net/http-backed implementation.
var defaultHTTPClient HTTPClient

// Checks is the authoritative slice of all registered cluster checks. The
// verify-cluster make target iterates this slice in order.
//
// In the nil-client registrations below the Fn fields are closures that capture
// the default*Client vars. Production callers are expected to replace those vars
// before invoking Checks, or construct their own CheckFunc values using the
// exported constructor functions.
var Checks = []CheckRegistration{
	{
		Name:     "deployment-ready",
		Category: "infrastructure",
		Fn:       CheckDeploymentReady(defaultKubeClient),
	},
	{
		Name:     "service-exists",
		Category: "infrastructure",
		Fn:       CheckServiceExists(defaultKubeClient),
	},
	{
		Name:     "healthz",
		Category: "health",
		Fn:       CheckHealthz(defaultHTTPClient),
	},
	{
		Name:     "readyz",
		Category: "health",
		Fn:       CheckReadyz(defaultHTTPClient),
	},
	{
		Name:     "configmap-injected",
		Category: "configmap",
		Fn:       CheckConfigMapInjected(defaultKubeClient),
	},
}
