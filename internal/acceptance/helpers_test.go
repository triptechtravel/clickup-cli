package acceptance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/triptechtravel/clickup-cli/internal/testutil"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/root"
)

// runCLI executes the real root command against the test factory's server and
// returns stdout, stderr and the command error.
func runCLI(t *testing.T, tf *testutil.TestFactory, args ...string) (string, string, error) {
	t.Helper()
	// Reset between invocations so a test that runs the CLI twice sees only the
	// second run's output. Without this, asserting the absence of a string in a
	// later run silently matches the earlier one.
	tf.OutBuf.Reset()
	tf.ErrBuf.Reset()
	cmd := root.NewCmdRoot(tf.Factory)
	cmd.SetOut(tf.OutBuf)
	cmd.SetErr(tf.ErrBuf)
	err := testutil.RunCommand(t, cmd, args...)
	return tf.OutBuf.String(), tf.ErrBuf.String(), err
}

// repoRoot walks up from the test's working directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate go.mod above the test directory")
		}
		dir = parent
	}
}

// readRepoFile reads a file relative to the repo root.
func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// grepRepo returns "path:line" for every match of pattern in non-generated,
// non-test Go files under dir. Used by the Phase 5 structural guards.
func grepRepo(t *testing.T, dir, pattern string) []string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	var hits []string
	rootDir := filepath.Join(repoRoot(t), dir)

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") ||
			strings.HasSuffix(name, ".gen.go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(b), "\n") {
			if re.MatchString(line) {
				rel, _ := filepath.Rel(repoRoot(t), path)
				hits = append(hits, rel+":"+itoa(i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return hits
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// makefileRecipe returns the body of a single Makefile target, so assertions
// can be scoped to one recipe instead of matching anywhere in the file.
func makefileRecipe(t *testing.T, target string) string {
	t.Helper()
	var body []string
	inTarget := false
	for _, line := range strings.Split(readRepoFile(t, "Makefile"), "\n") {
		if strings.HasPrefix(line, target+":") {
			inTarget = true
			continue
		}
		if inTarget {
			if line != "" && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") {
				break
			}
			body = append(body, line)
		}
	}
	if !inTarget {
		t.Fatalf("Makefile target %q not found", target)
	}
	return strings.Join(body, "\n")
}
