package version

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// parse reads the VERSION file at path and returns the major, minor, and patch
// integers. Returns an error if the file cannot be read or the content is not a
// valid three-segment semver string (MAJOR.MINOR.PATCH).
func parse(path string) (major, minor, patch int, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read VERSION file %q: %w", path, err)
	}

	raw := strings.TrimSpace(string(data))
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format %q: expected MAJOR.MINOR.PATCH", raw)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major segment %q: %w", parts[0], err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor segment %q: %w", parts[1], err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch segment %q: %w", parts[2], err)
	}

	return major, minor, patch, nil
}

// write formats the given version triple as MAJOR.MINOR.PATCH and writes it
// to the VERSION file at path, appending a trailing newline.
func write(path string, major, minor, patch int) error {
	content := fmt.Sprintf("%d.%d.%d\n", major, minor, patch)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write VERSION file %q: %w", path, err)
	}
	return nil
}

// BumpPatch reads the VERSION file at the given path, increments the patch
// segment by 1, writes the new version back, and prints a confirmation message
// to stdout. Returns an error if the file cannot be read, parsed, or written.
func BumpPatch(versionFilePath string) error {
	major, minor, patch, err := parse(versionFilePath)
	if err != nil {
		return err
	}
	patch++
	if err := write(versionFilePath, major, minor, patch); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Version bumped to %d.%d.%d\n", major, minor, patch)
	return nil
}

// BumpMinor reads the VERSION file at the given path, increments the minor
// segment by 1, resets patch to 0, writes the new version back, and prints a
// confirmation message to stdout. Returns an error if the file cannot be read,
// parsed, or written.
func BumpMinor(versionFilePath string) error {
	major, minor, _, err := parse(versionFilePath)
	if err != nil {
		return err
	}
	minor++
	patch := 0
	if err := write(versionFilePath, major, minor, patch); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Version bumped to %d.%d.%d\n", major, minor, patch)
	return nil
}

// BumpMajor reads the VERSION file at the given path, increments the major
// segment by 1, resets minor and patch to 0, writes the new version back, and
// prints a confirmation message to stdout. Returns an error if the file cannot
// be read, parsed, or written.
func BumpMajor(versionFilePath string) error {
	major, _, _, err := parse(versionFilePath)
	if err != nil {
		return err
	}
	major++
	minor := 0
	patch := 0
	if err := write(versionFilePath, major, minor, patch); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Version bumped to %d.%d.%d\n", major, minor, patch)
	return nil
}
