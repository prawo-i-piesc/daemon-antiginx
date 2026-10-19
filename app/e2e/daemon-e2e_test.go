package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func buildMockEngine(t *testing.T) string {
	t.Helper()

	tempDir := os.TempDir()

	binaryName := "mock-engine"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	enginePath := filepath.Join(tempDir, binaryName)
	sourcePath := filepath.Join("..", "testdata", "dummy-engine", "main.go")

	cmd := exec.Command("go", "build", "-o", enginePath, sourcePath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Mock engine compilation failed:\n%s", string(out))

	return enginePath
}
