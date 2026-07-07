package helper

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestSettlementPostInteractionData_invalid_data_length(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		_, err := DecodeSettlementPostInteractionData([]byte{})
		require.Error(t, err)
	})

	t.Run("invalid data", func(t *testing.T) {
		data, err := hexutil.Decode("0x010203")
		require.NoError(t, err)

		_, err = DecodeSettlementPostInteractionData(data)
		require.Error(t, err)
	})
}
