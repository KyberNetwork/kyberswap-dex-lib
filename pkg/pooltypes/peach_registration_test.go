package pooltypes_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPeachProductionRegistration(t *testing.T) {
	cmd := exec.Command("go", "run", "./testdata/peach_registration.go")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
