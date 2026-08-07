package flap

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

// tokenStateV8Result mirrors IPortalTypes.TokenStateV8, as confirmed by the Portal ABI/source and
// cross-checked live on-chain (buyTaxRate/sellTaxRate matched the board API's tax.buyTaxBps/sellTaxBps
// exactly for a live tax token). Reserve is the canonical on-chain quote-token reserve (already in the
// quote token's own decimals) - it happens to equal estimateReserveV2(curve, circulatingSupply,
// quoteDecimals), but reading it directly avoids any risk of the tracker's own LibCurve port drifting
// from the contract's. Shared between the tracker and the TokenCreated pool factory decoder.
type tokenStateV8Result struct {
	Status                   uint8
	Reserve                  *big.Int
	CirculatingSupply        *big.Int
	Price                    *big.Int
	TokenVersion             uint8
	R                        *big.Int
	H                        *big.Int
	K                        *big.Int
	DexSupplyThresh          *big.Int
	QuoteTokenAddress        common.Address
	NativeToQuoteSwapEnabled bool
	ExtensionID              [32]byte
	BuyTaxRate               *big.Int
	SellTaxRate              *big.Int
	Pool                     common.Address
	Progress                 *big.Int
	LpFeeProfile             uint8
	DexId                    uint8
}

type feeRateResult struct {
	BuyFeeRate  *big.Int
	SellFeeRate *big.Int
}

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address})

	// Once a token has left the bonding curve (graduated to DEX, or an invalid/unknown status), it
	// permanently stops trading through Portal's bonding-curve path. Early-return without RPC once
	// that's already recorded, per the permanent-disable pattern: downstream persists Reserves=0 and
	// stops feeding this pool to path-finding.
	var prevExtra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &prevExtra); err == nil && prevExtra.Status != TokenStatusTradable {
			return p, nil
		}
	}

	lg.Info("Started getting new pool state")
	defer lg.Info("Finished getting new pool state")

	var (
		stateResult tokenStateV8Result
		feeResult   feeRateResult
		taxEnabled  bool
	)
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    portalABI,
			Target: t.config.PortalAddress,
			Method: "getTokenV8",
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&struct{ *tokenStateV8Result }{&stateResult}}).
		AddCall(&ethrpc.Call{
			ABI:    portalABI,
			Target: t.config.PortalAddress,
			Method: "getFeeRate",
		}, []any{&struct{ *feeRateResult }{&feeResult}}).
		AddCall(&ethrpc.Call{
			ABI:    portalABI,
			Target: t.config.PortalAddress,
			Method: "enableTaxOnBondingCurve",
		}, []any{&taxEnabled}).
		Aggregate()
	if err != nil {
		return entity.Pool{}, err
	}
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	status := TokenStatus(stateResult.Status)

	if status != TokenStatusTradable {
		// Graduated (DEX) or invalid: stop feeding this pool to path-finding, permanently. Timestamp
		// is set to 1 (not time.Now()) so pool-service's IsPoolActive/WouldArchive - which treats
		// Timestamp==0 as always-active but an old-enough Timestamp as inactive - immediately marks
		// this pool eligible for archival instead of waiting out the usual inactive-duration window.
		p.Reserves = entity.PoolReserves{"0", "0"}
		extraBytes, err := json.Marshal(Extra{Status: status})
		if err != nil {
			return entity.Pool{}, err
		}
		p.Extra = string(extraBytes)
		p.Timestamp = 1
		return p, nil
	}

	curve := Curve{
		R: uint256.MustFromBig(stateResult.R),
		H: uint256.MustFromBig(stateResult.H),
		K: uint256.MustFromBig(stateResult.K),
	}
	circulatingSupply := uint256.MustFromBig(stateResult.CirculatingSupply)
	dexSupplyThresh := uint256.MustFromBig(stateResult.DexSupplyThresh)
	reserve := uint256.MustFromBig(stateResult.Reserve)

	tokenReserve := new(uint256.Int).Sub(totalSupply, circulatingSupply)

	extra := Extra{
		Status:                   status,
		Curve:                    curve,
		CirculatingSupply:        circulatingSupply,
		DexSupplyThresh:          dexSupplyThresh,
		BuyFeeBps:                feeResult.BuyFeeRate.Uint64(),
		SellFeeBps:               feeResult.SellFeeRate.Uint64(),
		BuyTaxBps:                stateResult.BuyTaxRate.Uint64(),
		SellTaxBps:               stateResult.SellTaxRate.Uint64(),
		TaxOnBondingCurveEnabled: taxEnabled,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserve.Dec(), tokenReserve.Dec()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}
