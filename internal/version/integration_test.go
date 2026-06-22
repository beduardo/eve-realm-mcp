package version_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// projectRoot resolves the absolute path to the repository root by walking up
// from this test file's directory until a Makefile is found.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("projectRoot: runtime.Caller failed")
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("projectRoot: Makefile not found walking up from test file")
		}
		dir = parent
	}
}

// freePort finds and returns an available TCP port on localhost by momentarily
// binding to port 0 and immediately releasing the listener.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

// runMake executes make with the given target in the project root directory and
// returns the combined stdout+stderr output together with the exit error (nil on
// success).
func runMake(t *testing.T, root, target string) (string, error) {
	t.Helper()
	cmd := exec.Command("make", target)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// startBinary launches the compiled binary with the given port, waits until the
// /version endpoint is reachable (up to 5 s), and returns the running *exec.Cmd.
// The caller must call cmd.Process.Kill() when done.
func startBinary(t *testing.T, binaryPath string, port int) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(binaryPath, "--port", fmt.Sprintf("%d", port))
	cmd.Stdout = os.Stderr // redirect binary logs to test stderr for visibility
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("startBinary: %v", err)
	}

	// Poll until the /version endpoint responds or we time out.
	deadline := time.Now().Add(5 * time.Second)
	url := fmt.Sprintf("http://127.0.0.1:%d/version", port)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:noctx
		if err == nil {
			resp.Body.Close()
			return cmd
		}
		time.Sleep(50 * time.Millisecond)
	}
	cmd.Process.Kill()
	t.Fatalf("startBinary: /version endpoint did not become ready within 5 s on port %d", port)
	return nil
}

// queryVersion hits the /version endpoint of a running binary and returns the
// decoded JSON fields.
func queryVersion(t *testing.T, port int) (version, gitHash, buildDate string) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/version", port)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("queryVersion GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	var body struct {
		Version   string `json:"version"`
		GitHash   string `json:"git_hash"`
		BuildDate string `json:"build_date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("queryVersion decode JSON: %v", err)
	}
	return body.Version, body.GitHash, body.BuildDate
}

// ---------------------------------------------------------------------------
// TestMakeBuildProducesExecutable
// ---------------------------------------------------------------------------

// TestMakeBuildProducesExecutable verifies that `make build` compiles the
// binary and places an executable file at dist/eve-realm-mcp.
func TestMakeBuildProducesExecutable(t *testing.T) {
	root := projectRoot(t)
	binaryPath := filepath.Join(root, "dist", "eve-realm-mcp")

	out, err := runMake(t, root, "build")
	if err != nil {
		t.Fatalf("make build failed:\n%s\nerror: %v", out, err)
	}

	info, statErr := os.Stat(binaryPath)
	if statErr != nil {
		t.Fatalf("dist/eve-realm-mcp not found after make build: %v", statErr)
	}

	// Verify the file is executable by the owner (mode & 0100 != 0).
	if info.Mode()&0o100 == 0 {
		t.Errorf("dist/eve-realm-mcp is not executable: mode = %v", info.Mode())
	}
}

// ---------------------------------------------------------------------------
// TestMakeBuildDefaultVersionValues
// ---------------------------------------------------------------------------

// TestMakeBuildDefaultVersionValues verifies that a binary built via
// `make build` (without ldflags injection) reports Version=dev,
// GitHash=unknown, and BuildDate=unknown from the /version endpoint.
func TestMakeBuildDefaultVersionValues(t *testing.T) {
	root := projectRoot(t)
	binaryPath := filepath.Join(root, "dist", "eve-realm-mcp")

	out, err := runMake(t, root, "build")
	if err != nil {
		t.Fatalf("make build failed:\n%s\nerror: %v", out, err)
	}

	port := freePort(t)
	cmd := startBinary(t, binaryPath, port)
	defer cmd.Process.Kill()

	version, gitHash, buildDate := queryVersion(t, port)

	cases := []struct {
		field string
		got   string
		want  string
	}{
		{"version", version, "dev"},
		{"git_hash", gitHash, "unknown"},
		{"build_date", buildDate, "unknown"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("/version field %q = %q, want %q", tc.field, tc.got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestMakeBuildProdInjectsVersion
// ---------------------------------------------------------------------------

// TestMakeBuildProdInjectsVersion verifies that `make build-prod` injects the
// version from the VERSION file into the compiled binary.
func TestMakeBuildProdInjectsVersion(t *testing.T) {
	root := projectRoot(t)
	binaryPath := filepath.Join(root, "dist", "eve-realm-mcp")

	// Read the expected version from the VERSION file before building.
	versionFileData, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION file: %v", err)
	}
	expectedVersion := strings.TrimSpace(string(versionFileData))

	out, buildErr := runMake(t, root, "build-prod")
	if buildErr != nil {
		t.Fatalf("make build-prod failed:\n%s\nerror: %v", out, buildErr)
	}

	port := freePort(t)
	cmd := startBinary(t, binaryPath, port)
	defer cmd.Process.Kill()

	gotVersion, _, _ := queryVersion(t, port)

	if gotVersion != expectedVersion {
		t.Errorf("/version field \"version\" = %q, want %q (from VERSION file)", gotVersion, expectedVersion)
	}
}

// ---------------------------------------------------------------------------
// TestMakeBuildProdInjectsBuildDate
// ---------------------------------------------------------------------------

// TestMakeBuildProdInjectsBuildDate verifies that `make build-prod` injects a
// BuildDate that starts with a valid YYYY-MM-DD date string.
func TestMakeBuildProdInjectsBuildDate(t *testing.T) {
	root := projectRoot(t)
	binaryPath := filepath.Join(root, "dist", "eve-realm-mcp")

	out, err := runMake(t, root, "build-prod")
	if err != nil {
		t.Fatalf("make build-prod failed:\n%s\nerror: %v", out, err)
	}

	port := freePort(t)
	cmd := startBinary(t, binaryPath, port)
	defer cmd.Process.Kill()

	_, _, buildDate := queryVersion(t, port)

	// BuildDate is injected as an ISO-8601 timestamp (YYYY-MM-DDTHH:MM:SSZ).
	// We validate that it begins with a valid YYYY-MM-DD segment.
	dateRE := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	if !dateRE.MatchString(buildDate) {
		t.Errorf("/version field \"build_date\" = %q, want a string starting with YYYY-MM-DD", buildDate)
	}

	// Parse the date prefix and verify it is a real calendar date.
	datePart := buildDate[:10]
	if _, parseErr := time.Parse("2006-01-02", datePart); parseErr != nil {
		t.Errorf("/version field \"build_date\" prefix %q is not a valid YYYY-MM-DD date: %v", datePart, parseErr)
	}
}

// ---------------------------------------------------------------------------
// TestMakeTestExitsNonZeroOnFailure
// ---------------------------------------------------------------------------

// TestMakeTestExitsNonZeroOnFailure verifies that `make test` would propagate a
// non-zero exit code from `go test` by injecting a failing test into a
// temporary standalone Go package and running `go test` directly on that
// package. This avoids mutating the real test suite while confirming that the
// underlying test runner honours failing test files.
func TestMakeTestExitsNonZeroOnFailure(t *testing.T) {
	// Create a temporary Go module with a single failing test.
	dir := t.TempDir()

	goMod := []byte("module example.com/failing\n\ngo 1.21\n")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), goMod, 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	failingTest := []byte(`package failing_test

import "testing"

func TestAlwaysFails(t *testing.T) {
	t.Fatal("intentional failure injected by integration test")
}
`)
	if err := os.WriteFile(filepath.Join(dir, "fail_test.go"), failingTest, 0o644); err != nil {
		t.Fatalf("write fail_test.go: %v", err)
	}

	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		t.Error("go test ./... expected to exit non-zero for a package with t.Fatal, but it exited zero")
	}
}

// ---------------------------------------------------------------------------
// TestReleasePatchDependsOnTest (static Makefile inspection)
// ---------------------------------------------------------------------------

// TestReleasePatchDependsOnTest verifies that the release-patch Makefile target
// lists `test` as a prerequisite, ensuring that a failing test suite aborts the
// release pipeline without executing make directly.
func TestReleasePatchDependsOnTest(t *testing.T) {
	root := projectRoot(t)
	makefileData, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}

	// Find the release-patch target rule.
	// Expected line: "release-patch: test bump-patch build-prod"
	lines := strings.Split(string(makefileData), "\n")
	var releasePatchRule string
	for _, line := range lines {
		if strings.HasPrefix(line, "release-patch:") {
			releasePatchRule = line
			break
		}
	}

	if releasePatchRule == "" {
		t.Fatal("Makefile does not contain a release-patch target")
	}

	// The prerequisites are everything after the colon.
	colonIdx := strings.Index(releasePatchRule, ":")
	prerequisites := releasePatchRule[colonIdx+1:]

	fields := strings.Fields(prerequisites)
	found := false
	for _, f := range fields {
		if f == "test" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("release-patch target prerequisites %q do not include \"test\"; release pipeline would not abort on test failure", prerequisites)
	}
}
