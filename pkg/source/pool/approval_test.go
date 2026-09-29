package pool

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetaInfoJSONIsFlat(t *testing.T) {
	data, err := json.Marshal(MetaInfo{
		ApprovalAddress: "0xrouter",
		BlockNumber:     123,
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"approvalAddress":"0xrouter","blockNumber":123}`, string(data))
}
