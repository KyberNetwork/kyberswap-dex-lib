package litepsm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	liteDai  = common.HexToAddress("0xf6e72Db5454dd049d0788e411b06CfAF16853042") // joins itself
	wrapper  = common.HexToAddress("0xA188EEC8F81263234dA3622A406892F3D630f98c") // wraps liteDai
	usddPsm  = common.HexToAddress("0xcE355440c00014A229bbEc030A2B8f8EB45a2897")
	usddJoin = common.HexToAddress("0x217e42CEB2eAE9ECB788fDF0e31c806c531760A3") // separate gemJoin
)

// gemJoin is read from the inner psm, so a wrapper over a self-joining lite psm must not
// get GemJoin (it would be charged gasJoin for a join it never calls).
func TestSetJoin(t *testing.T) {
	t.Parallel()
	var s StaticExtra
	s.setJoin(liteDai, liteDai, liteDai)
	require.Equal(t, StaticExtra{}, s)

	s = StaticExtra{}
	s.setJoin(wrapper, liteDai, liteDai)
	require.Nil(t, s.GemJoin)
	require.Equal(t, &liteDai, s.InnerPsm) // tracker reads the dai balance here

	s = StaticExtra{}
	s.setJoin(usddPsm, usddPsm, usddJoin)
	require.Equal(t, &usddJoin, s.GemJoin)
	require.Nil(t, s.InnerPsm)
}
