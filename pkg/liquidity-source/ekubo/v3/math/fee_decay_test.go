package math

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecayedLaunchFeeFrozenDigests(t *testing.T) {
	data, err := os.ReadFile("../testdata/zero-seed-launch.json")
	require.NoError(t, err)
	var fixture struct {
		FeeCurve struct {
			Digests []struct {
				Label   string `json:"label"`
				Initial uint64 `json:"initialFee,string"`
				Final   uint64 `json:"finalFee,string"`
				Rate    uint64 `json:"decayRateX32,string"`
				Last    uint64 `json:"last"`
				Step    uint64 `json:"step"`
				Digest  string `json:"digest"`
			} `json:"digests_sha256"`
		} `json:"fee_curve"`
	}
	require.NoError(t, json.Unmarshal(data, &fixture))
	require.Len(t, fixture.FeeCurve.Digests, 5)
	for _, v := range fixture.FeeCurve.Digests {
		t.Run(v.Label, func(t *testing.T) {
			var digest [32]byte
			var input [40]byte
			previous := v.Initial
			for elapsed := uint64(0); elapsed <= v.Last; elapsed += v.Step {
				fee := DecayedLaunchFee(v.Initial, v.Final, v.Rate, elapsed)
				require.LessOrEqual(t, fee, previous)
				require.GreaterOrEqual(t, fee, v.Final)
				previous = fee
				copy(input[:32], digest[:])
				binary.BigEndian.PutUint64(input[32:], fee)
				digest = sha256.Sum256(input[:])
			}
			require.Equal(t, v.Digest, hex.EncodeToString(digest[:]))
		})
	}
}

func TestDecayedLaunchFeeBoundaries(t *testing.T) {
	max := ^uint64(0)
	require.Equal(t, max, DecayedLaunchFee(max, 0, 0, max))
	require.Equal(t, max, DecayedLaunchFee(max, 0, 1, (1<<32)-1))
	require.Equal(t, uint64(1), DecayedLaunchFee(max, 0, 1, MaxDecayProduct))
	require.Zero(t, DecayedLaunchFee(max, 0, 1, MaxDecayProduct+1))
	require.Equal(t, uint64(123), DecayedLaunchFee(max, 123, max, max))
	require.Equal(t, max, DecayedLaunchFee(max, max, max, 1))
}
