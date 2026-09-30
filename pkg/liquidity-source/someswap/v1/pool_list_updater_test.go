package someswapv1

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestStaticExtraJSONCompatibility(t *testing.T) {
	t.Parallel()

	staticExtra := StaticExtra{
		WTokens: [2]*uint256.Int{
			u256OrZero(uint256.NewInt(123)),
			u256OrZero(nil),
		},
	}

	b, err := json.Marshal(staticExtra)
	require.NoError(t, err)
	require.Equal(t, `{"ws":["123","0"]}`, string(b))
}

