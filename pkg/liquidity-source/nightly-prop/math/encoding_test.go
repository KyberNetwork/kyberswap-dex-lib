package math

import (
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func encodingPool(t *testing.T) Pool {
	f := funded(t, levelConfigs())
	f.seedPosition(0, fraction(t, dec(t, "7097601498202505053776590463358")))
	f.pool.Levels[2].Config.AskExtraSpread = 1_000
	return f.pool
}

func TestFraction_JSON(t *testing.T) {
	f := fraction(t, dec(t, "7097601498202505053776590463358"))
	encoded, err := json.Marshal(f)
	require.NoError(t, err)
	require.JSONEq(t, `"7097601498202505053776590463358"`, string(encoded))
	var decoded Fraction
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, f, decoded)

	tooLarge := new(uint256.Int).AddUint64(maxRaw, 1)
	require.ErrorIs(t, json.Unmarshal([]byte(`"`+tooLarge.Dec()+`"`), &decoded), ErrOverflow)
}

func TestFraction_Msgpack(t *testing.T) {
	encoded, err := msgpack.Marshal(&Position{LevelID: 3, Fraction: Upper})
	require.NoError(t, err)
	var decoded Position
	require.NoError(t, msgpack.Unmarshal(encoded, &decoded))
	require.Equal(t, Position{LevelID: 3, Fraction: Upper}, decoded)

	tooLarge, err := msgpack.Marshal(new(uint256.Int).AddUint64(maxRaw, 1))
	require.NoError(t, err)
	var f Fraction
	require.ErrorIs(t, msgpack.Unmarshal(tooLarge, &f), ErrOverflow)
}

// TestPool_Encoding: a whole pool, with its position inside a level, round-trips through JSON and
// msgpack.
func TestPool_Encoding(t *testing.T) {
	pool := encodingPool(t)

	encoded, err := json.Marshal(&pool)
	require.NoError(t, err)
	var fromJSON Pool
	require.NoError(t, json.Unmarshal(encoded, &fromJSON))
	require.Equal(t, pool, fromJSON)

	encoded, err = msgpack.Marshal(&pool)
	require.NoError(t, err)
	var fromMsgpack Pool
	require.NoError(t, msgpack.Unmarshal(encoded, &fromMsgpack))
	require.Equal(t, pool, fromMsgpack)
}
