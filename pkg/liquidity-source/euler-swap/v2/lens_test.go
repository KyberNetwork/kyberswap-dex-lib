package v2

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// TestLensTooLarge: only running out of gas splits a batch; the lens's other error and a node
// failure do not, since a smaller batch cannot fix them.
func TestLensTooLarge(t *testing.T) {
	assert.True(t, lensTooLarge(&ephemeral.RevertError{Err: &ephemeral.ContractError{Name: "GasExhausted"}}))
	assert.True(t, lensTooLarge(fmt.Errorf("%w: out of gas", ephemeral.ErrNoRevertData)))
	assert.False(t, lensTooLarge(&ephemeral.RevertError{Err: &ephemeral.ContractError{Name: "MismatchedVaultInput"}}))
	assert.False(t, lensTooLarge(errors.New("429 too many requests")))
}
