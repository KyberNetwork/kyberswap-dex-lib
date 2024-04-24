package bancorv21

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	// Read the embedded file
	data, err := sampleFile.ReadFile("sample_pool_data.txt")
	if err != nil {
		fmt.Println("Error reading embedded file:", err)
		return
	}

	// Unmarshal the JSON data into the Pool struct
	var pool entity.Pool
	err = json.Unmarshal(data, &pool)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}
	poolSim, err := NewPoolSimulator(pool)
	assert.Nil(t, err)

	b, err := poolSim.MarshalMsg(nil)
	require.NoError(t, err)
	actual := new(PoolSimulator)
	_, err = actual.UnmarshalMsg(b)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(poolSim, actual, testutil.CmpOpts(PoolSimulator{})...))
}
