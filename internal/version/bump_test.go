package version_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beduardo/eve-realm-mcp/internal/version"
)

// writeVersionFile creates a VERSION file containing the given version string
// inside dir and returns the absolute path to that file.
func writeVersionFile(t *testing.T, dir, v string) string {
	t.Helper()
	path := filepath.Join(dir, "VERSION")
	if err := os.WriteFile(path, []byte(v+"\n"), 0o644); err != nil {
		t.Fatalf("setup: write VERSION file: %v", err)
	}
	return path
}

// readVersionFile reads the VERSION file at path and returns the trimmed
// version string.
func readVersionFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read VERSION file: %v", err)
	}
	return strings.TrimSpace(string(data))
}

// captureStdout temporarily replaces os.Stdout with a pipe, runs fn, and
// returns everything written to stdout during fn's execution.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("captureStdout: create pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("captureStdout: read pipe: %v", err)
	}
	return buf.String()
}

// ---------------------------------------------------------------------------
// BumpPatch
// ---------------------------------------------------------------------------

func TestBumpPatch(t *testing.T) {
	cases := []struct {
		name    string
		initial string
		want    string
	}{
		{
			name:    "patch_increments_from_zero",
			initial: "0.1.0",
			want:    "0.1.1",
		},
		{
			name:    "patch_increments_sequentially",
			initial: "0.1.1",
			want:    "0.1.2",
		},
		{
			name:    "patch_increments_large_value",
			initial: "1.2.9",
			want:    "1.2.10",
		},
		{
			name:    "major_and_minor_unchanged",
			initial: "3.7.4",
			want:    "3.7.5",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeVersionFile(t, dir, tc.initial)

			if err := version.BumpPatch(path); err != nil {
				t.Fatalf("BumpPatch(%q) returned unexpected error: %v", tc.initial, err)
			}

			got := readVersionFile(t, path)
			if got != tc.want {
				t.Errorf("BumpPatch(%q): VERSION = %q, want %q", tc.initial, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BumpMinor
// ---------------------------------------------------------------------------

func TestBumpMinor(t *testing.T) {
	cases := []struct {
		name    string
		initial string
		want    string
	}{
		{
			name:    "minor_increments_and_patch_resets",
			initial: "0.1.3",
			want:    "0.2.0",
		},
		{
			name:    "minor_increments_from_zero_patch_nine",
			initial: "0.0.9",
			want:    "0.1.0",
		},
		{
			name:    "minor_increments_large_value",
			initial: "1.9.5",
			want:    "1.10.0",
		},
		{
			name:    "major_unchanged",
			initial: "2.3.1",
			want:    "2.4.0",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeVersionFile(t, dir, tc.initial)

			if err := version.BumpMinor(path); err != nil {
				t.Fatalf("BumpMinor(%q) returned unexpected error: %v", tc.initial, err)
			}

			got := readVersionFile(t, path)
			if got != tc.want {
				t.Errorf("BumpMinor(%q): VERSION = %q, want %q", tc.initial, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BumpMajor
// ---------------------------------------------------------------------------

func TestBumpMajor(t *testing.T) {
	cases := []struct {
		name    string
		initial string
		want    string
	}{
		{
			name:    "major_increments_and_minor_patch_reset",
			initial: "0.3.5",
			want:    "1.0.0",
		},
		{
			name:    "major_increments_large_value",
			initial: "2.7.9",
			want:    "3.0.0",
		},
		{
			name:    "minor_and_patch_both_reset",
			initial: "1.5.3",
			want:    "2.0.0",
		},
		{
			name:    "major_increments_sequentially",
			initial: "3.0.0",
			want:    "4.0.0",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeVersionFile(t, dir, tc.initial)

			if err := version.BumpMajor(path); err != nil {
				t.Fatalf("BumpMajor(%q) returned unexpected error: %v", tc.initial, err)
			}

			got := readVersionFile(t, path)
			if got != tc.want {
				t.Errorf("BumpMajor(%q): VERSION = %q, want %q", tc.initial, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Stdout confirmation messages
// ---------------------------------------------------------------------------

func TestBumpStdoutMessage(t *testing.T) {
	cases := []struct {
		name    string
		initial string
		bump    func(string) error
		want    string
	}{
		{
			name:    "patch_prints_confirmation",
			initial: "0.1.0",
			bump:    version.BumpPatch,
			want:    "Version bumped to 0.1.1",
		},
		{
			name:    "minor_prints_confirmation",
			initial: "0.1.3",
			bump:    version.BumpMinor,
			want:    "Version bumped to 0.2.0",
		},
		{
			name:    "major_prints_confirmation",
			initial: "0.3.5",
			bump:    version.BumpMajor,
			want:    "Version bumped to 1.0.0",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeVersionFile(t, dir, tc.initial)

			var bumpErr error
			output := captureStdout(t, func() {
				bumpErr = tc.bump(path)
			})
			if bumpErr != nil {
				t.Fatalf("bump returned unexpected error: %v", bumpErr)
			}

			if !strings.Contains(output, tc.want) {
				t.Errorf("stdout = %q, want it to contain %q", output, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Leading-zero safety (no octal interpretation)
// ---------------------------------------------------------------------------

func TestBumpLeadingZeroSafety(t *testing.T) {
	// Version segments like "09" must be treated as decimal 9 — not as an
	// invalid octal literal. strconv.Atoi handles this correctly; this test
	// guards against any future regression that uses a numeric literal or
	// bit-shifting approach that could misinterpret leading zeros.
	cases := []struct {
		name    string
		initial string
		bump    func(string) error
		want    string
	}{
		{
			name:    "patch_with_leading_zero_segment",
			initial: "0.0.09",
			bump:    version.BumpPatch,
			want:    "0.0.10",
		},
		{
			name:    "minor_with_leading_zero_segment",
			initial: "0.09.0",
			bump:    version.BumpMinor,
			want:    "0.10.0",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeVersionFile(t, dir, tc.initial)

			if err := tc.bump(path); err != nil {
				t.Fatalf("bump returned unexpected error: %v", err)
			}

			got := readVersionFile(t, path)
			if got != tc.want {
				t.Errorf("bump(%q): VERSION = %q, want %q", tc.initial, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Sequential bumps (each call reads back the file the previous call wrote)
// ---------------------------------------------------------------------------

func TestSequentialBumps(t *testing.T) {
	t.Run("sequential_patch_bumps", func(t *testing.T) {
		dir := t.TempDir()
		path := writeVersionFile(t, dir, "0.1.0")

		steps := []string{"0.1.1", "0.1.2", "0.1.3"}
		for i, want := range steps {
			if err := version.BumpPatch(path); err != nil {
				t.Fatalf("BumpPatch call %d: %v", i+1, err)
			}
			got := readVersionFile(t, path)
			if got != want {
				t.Errorf("after %d BumpPatch calls: VERSION = %q, want %q", i+1, got, want)
			}
		}
	})

	t.Run("patch_then_minor_resets_patch", func(t *testing.T) {
		dir := t.TempDir()
		path := writeVersionFile(t, dir, "0.1.0")

		// Bump patch twice: 0.1.0 → 0.1.1 → 0.1.2
		for i := 0; i < 2; i++ {
			if err := version.BumpPatch(path); err != nil {
				t.Fatalf("BumpPatch call %d: %v", i+1, err)
			}
		}
		// Now bump minor: 0.1.2 → 0.2.0
		if err := version.BumpMinor(path); err != nil {
			t.Fatalf("BumpMinor: %v", err)
		}
		got := readVersionFile(t, path)
		if got != "0.2.0" {
			t.Errorf("after patch×2 + minor: VERSION = %q, want %q", got, "0.2.0")
		}
	})

	t.Run("minor_then_major_resets_all", func(t *testing.T) {
		dir := t.TempDir()
		path := writeVersionFile(t, dir, "0.1.0")

		// Bump minor: 0.1.0 → 0.2.0
		if err := version.BumpMinor(path); err != nil {
			t.Fatalf("BumpMinor: %v", err)
		}
		// Bump major: 0.2.0 → 1.0.0
		if err := version.BumpMajor(path); err != nil {
			t.Fatalf("BumpMajor: %v", err)
		}
		got := readVersionFile(t, path)
		if got != "1.0.0" {
			t.Errorf("after minor + major: VERSION = %q, want %q", got, "1.0.0")
		}
	})
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

func TestBumpErrorCases(t *testing.T) {
	cases := []struct {
		name string
		bump func(string) error
	}{
		{"patch_missing_file", version.BumpPatch},
		{"minor_missing_file", version.BumpMinor},
		{"major_missing_file", version.BumpMajor},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "VERSION") // file does not exist

			err := tc.bump(path)
			if err == nil {
				t.Errorf("expected an error for missing VERSION file, got nil")
			}
		})
	}

	t.Run("patch_invalid_version_format", func(t *testing.T) {
		dir := t.TempDir()
		path := writeVersionFile(t, dir, "not-a-version")

		err := version.BumpPatch(path)
		if err == nil {
			t.Error("expected an error for invalid version format, got nil")
		}
	})

	t.Run("minor_invalid_version_format", func(t *testing.T) {
		dir := t.TempDir()
		path := writeVersionFile(t, dir, "1.2")

		err := version.BumpMinor(path)
		if err == nil {
			t.Error("expected an error for version missing patch segment, got nil")
		}
	})
}

// Ensure fmt is used (it is imported for potential test helper expansion).
var _ = fmt.Sprintf
