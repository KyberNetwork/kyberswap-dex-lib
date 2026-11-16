package lazy

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	erc4626 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const probeVault = "0x00000000000000000000000000000000000000aa"

var (
	// keccak256(pad32(AddrDummy) . pad32(2)): the balanceOf[AddrDummy] key of a vault with the mapping at slot 2.
	slot2Key = crypto.Keccak256Hash(common.LeftPadBytes(erc4626.AddrDummy[:], 32), common.LeftPadBytes([]byte{2}, 32))
	probeCfg = erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 1, Redeem: 1}}
)

func word(v *big.Int) []byte { return common.LeftPadBytes(v.Bytes(), 32) }

func marker(i int64) *big.Int {
	return new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(i))
}

// run builds the lazy request, answers every call (maxRedeem/balanceOf from the args, the rest 1000), as a
// batch consumer would, and applies the result.
func run(t *testing.T, cfg erc4626.VaultCfg, maxRedeem, balanceOf *big.Int, v2 bool) (poolpkg.ILazyRequest, entity.Pool, erc4626.Extra, error) {
	t.Helper()
	cl := ethrpc.New("http://127.0.0.1:1") // never reached unless the tracker re-reads
	var p entity.Pool
	req, apply := Lazycall(context.Background(), &p, cl, probeVault, "", cfg, false, nil)
	for i, unpack := range req.GetUnpacks() {
		v := big.NewInt(1000)
		switch m := req.GetEthRpcCall(i).Method; m {
		case erc4626.VaultV2MethodCanReceiveShares, erc4626.VaultV2MethodCanSendShares, erc4626.VaultV2MethodCanSendAssets:
			if !v2 {
				continue // a plain vault has no such view: the read stays nil
			}
		case erc4626.Erc4626MethodMaxRedeem:
			v = maxRedeem
		case "balanceOf":
			v = balanceOf
		}
		require.NoError(t, unpack(word(v)))
	}
	out, err := apply(big.NewInt(1))
	var e erc4626.Extra
	if err == nil {
		require.NoError(t, json.Unmarshal([]byte(out.Extra), &e))
	}
	return req, out, e, err
}

func TestLazycallProbe(t *testing.T) {
	t.Parallel()
	cached := probeCfg
	cached.Balance = erc4626.BalanceSlot{Key: slot2Key}
	none := probeCfg
	none.Balance = erc4626.BalanceSlot{None: true}
	noRedeem := erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 1}}

	tests := []struct {
		name         string
		cfg          erc4626.VaultCfg
		wantKeys     int
		wantBalCall  bool
		maxRedeem    int64
		readBack     *big.Int
		v2           bool // Morpho VaultV2 permission views answer
		wantErr      bool // a failed clean re-read (no node here) fails the update
		wantRedeem   uint64
		wantStaticBK string
	}{
		{name: "unprobed, hit with cap 0 stays closed", cfg: probeCfg, wantKeys: 22, wantBalCall: true,
			maxRedeem: 0, readBack: marker(2), wantRedeem: 0, wantStaticBK: slot2Key.Hex()},
		{name: "unprobed, hit with real cap", cfg: probeCfg, wantKeys: 22, wantBalCall: true,
			maxRedeem: 5, readBack: marker(2), wantRedeem: 5, wantStaticBK: slot2Key.Hex()},
		{name: "cached key verified, cap 0 closed", cfg: cached, wantKeys: 1, wantBalCall: true,
			maxRedeem: 0, readBack: marker(0), wantRedeem: 0}, // unchanged slot: StaticExtra untouched
		{name: "VaultV2 hit with cap 0 keeps the by-design totalSupply fallback", cfg: probeCfg, wantKeys: 22,
			wantBalCall: true, v2: true, maxRedeem: 0, readBack: marker(2), wantRedeem: 1000, wantStaticBK: slot2Key.Hex()},
		{name: "cached key stopped working triggers clean re-read", cfg: cached, wantKeys: 1, wantBalCall: true,
			maxRedeem: 0, readBack: big.NewInt(0), wantErr: true},
		{name: "unprobed, no marker found triggers clean re-read", cfg: probeCfg, wantKeys: 22, wantBalCall: true,
			maxRedeem: 13, readBack: big.NewInt(0), wantErr: true},
		{name: "already known none: no probe, old fallback", cfg: none, wantKeys: 0, wantBalCall: false,
			maxRedeem: 0, readBack: big.NewInt(0), wantRedeem: 1000},
		{name: "redeem unsupported: no probe", cfg: noRedeem, wantKeys: 0, wantBalCall: false,
			maxRedeem: 0, readBack: big.NewInt(0), wantRedeem: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req, out, e, err := run(t, tc.cfg, big.NewInt(tc.maxRedeem), tc.readBack, tc.v2)

			ov, ok := req.(poolpkg.ILazyRequestWithOverrides)
			require.True(t, ok)
			assert.Len(t, ov.GetOverrides()[common.HexToAddress(probeVault)].StateDiff, tc.wantKeys)

			hasBal := false
			for i := range req.GetUnpacks() {
				hasBal = hasBal || req.GetEthRpcCall(i).Method == "balanceOf"
			}
			assert.Equal(t, tc.wantBalCall, hasBal)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.cfg.Gas.Redeem > 0 {
				require.NotNil(t, e.MaxRedeem)
				assert.Equal(t, tc.wantRedeem, e.MaxRedeem.Uint64())
			}
			var se erc4626.StaticExtra
			_ = json.Unmarshal([]byte(out.StaticExtra), &se)
			assert.Equal(t, tc.wantStaticBK, se.BalanceKey)
		})
	}
}
