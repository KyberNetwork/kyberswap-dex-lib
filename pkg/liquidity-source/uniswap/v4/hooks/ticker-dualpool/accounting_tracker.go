package tickerdualpool

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

//go:embed abis/Accounting.json
var accountingJSON string
var accountingABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(accountingJSON))
	if err != nil {
		panic(err)
	}
	return a
}()
var implementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
var stataImplementationHash = common.HexToHash("0xaae6d8b1278cc94bb511e0d26bc88c503152c51502178d4e94ccfe041bfedf06")
var monadPoolManager = common.HexToAddress("0x188d586Ddcf52439676Ca21A244753fA19F9Ea8e")

// These are the configured Monad StataToken proxies, identified in Ticker's
// strategy provenance report. A governance implementation change must be reviewed.
var stataProxyHashes = map[common.Address]common.Hash{
	common.HexToAddress("0xC554aFfE2f581F5E0811e0D42D484ECaC5c6B8e2"): common.HexToHash("0xbb6020ea435669d42783dd81b07c1d6d689f010cba98478cd6a68afe1eddf621"),
	common.HexToAddress("0x5E7109A755fD2b86361F95065cDE07FbAb50CA1D"): common.HexToHash("0xc62ca00295a9790d68b11506005b02d5ac9cfeff6c7cf81ff02e6f44eb0738b1"),
	common.HexToAddress("0x967d427eF0Eb4AEB7a1B523A98B04061F094276B"): common.HexToHash("0xf4e412543dc42ed395f7dc7bf3cee8fbb387d5f99c54affc0754238f2b3144f7"),
}

type AaveLimits struct{ MaxDeposit, MaxWithdraw *uint256.Int }

// StorageAt has no state-override parameter. Apply the exact eth_call override
// semantics ourselves before falling back to the same pinned RPC block.
func storageAt(ctx context.Context, p *uniswapv4.HookParam, address common.Address, slot common.Hash) (common.Hash, error) {
	if override, ok := p.Overrides[address]; ok {
		if override.State != nil && override.StateDiff != nil {
			return common.Hash{}, ErrInvalidState
		}
		if override.State != nil {
			return override.State[slot], nil
		}
		if value, ok := override.StateDiff[slot]; ok {
			return value, nil
		}
	}
	data, err := p.RpcClient.GetETHClient().StorageAt(ctx, address, slot, p.BlockNumber)
	return common.BytesToHash(data), err
}
func codeAt(ctx context.Context, p *uniswapv4.HookParam, address common.Address) ([]byte, error) {
	if override, ok := p.Overrides[address]; ok && override.Code != nil {
		return override.Code, nil
	}
	return p.RpcClient.GetETHClient().CodeAt(ctx, address, p.BlockNumber)
}

// verifiedCode holds addresses whose runtime code already matched. Deployed code
// does not change on Monad, so later Tracks skip the bytecode download. Code from
// a state override is never cached.
var verifiedCode sync.Map

type codeKey struct {
	address common.Address
	label   string
}

// requireCode errors with label unless the code at address passes match.
func requireCode(ctx context.Context, p *uniswapv4.HookParam, address common.Address, label string, match func([]byte) bool) error {
	overridden := p.Overrides[address].Code != nil
	key := codeKey{address, label}
	if _, ok := verifiedCode.Load(key); ok && !overridden {
		return nil
	}
	code, err := codeAt(ctx, p, address)
	if err != nil {
		return err
	}
	if !match(code) {
		return fmt.Errorf("%s: %w", label, ErrInvalidState)
	}
	if !overridden {
		verifiedCode.Store(key, struct{}{})
	}
	return nil
}

func (h *Hook) Track(ctx context.Context, p *uniswapv4.HookParam) (json.RawMessage, error) {
	if h.address == productionHookAddress && (p == nil || p.Cfg == nil || p.Cfg.ChainID != valueobject.ChainIDMonad) {
		return nil, ErrInvalidState
	}
	// Failed reads must leave the previous snapshot intact.
	candidate := *h
	if _, err := candidate.trackBase(ctx, p); err != nil {
		return nil, err
	}
	if err := requireCode(ctx, p, h.address, "hook proxy bytecode", tickerProxyFingerprint.matches); err != nil {
		return nil, err
	}
	var poolManager common.Address
	managerRequest := p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
	managerRequest.AddCall(&ethrpc.Call{ABI: accountingABI, Target: hexutil.Encode(h.address[:]), Method: "poolManager"}, []any{&poolManager})
	if _, err := managerRequest.Aggregate(); err != nil {
		return nil, err
	}
	if poolManager != monadPoolManager {
		return nil, fmt.Errorf("PoolManager address: %w", ErrInvalidState)
	}
	slot, err := storageAt(ctx, p, h.address, implementationSlot)
	if err != nil {
		return nil, err
	}
	if err := requireCode(ctx, p, common.BytesToAddress(slot[:]), "hook implementation", matchesTickerHook); err != nil {
		return nil, err
	}
	var config uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(p.Pool.StaticExtra), &config); err != nil {
		return nil, err
	}
	// This accounting model currently covers ERC-20 pools on Monad only.
	// Native balances require a different read and settlement model.
	for _, native := range config.IsNative {
		if native {
			return nil, ErrInvalidState
		}
	}
	id := common.HexToHash(p.Pool.Address)
	currencies := [2]common.Address{common.HexToAddress(p.Pool.Tokens[0].Address), common.HexToAddress(p.Pool.Tokens[1].Address)}
	candidate.Currencies = currencies
	for side, currency := range currencies {
		partition := crypto.Keccak256Hash(id[:], common.LeftPadBytes(currency.Bytes(), 32))
		mapSlot := func(index int64) common.Hash {
			slot := common.BigToHash(big.NewInt(index))
			return crypto.Keccak256Hash(partition[:], slot[:])
		}
		packed, err := storageAt(ctx, p, h.address, mapSlot(5))
		if err != nil {
			return nil, err
		}
		word := new(uint256.Int).SetBytes(packed[:])
		claims := new(uint256.Int).Rsh(word, 128)
		raw := new(uint256.Int).And(word, new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 128), uint256.NewInt(1)))
		candidate.Raw[side] = raw
		candidate.Claims[side] = claims
		// Claims count as spendable only to the extent PoolManager holds cash.
		// Read both currencies, including a side with no ERC-4626 vault.
		var managerCash *big.Int
		cashRequest := p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
		cashRequest.AddCall(&ethrpc.Call{ABI: accountingABI, Target: hexutil.Encode(currency[:]), Method: "balanceOf", Params: []any{monadPoolManager}}, []any{&managerCash})
		if _, err := cashRequest.Aggregate(); err != nil {
			return nil, err
		}
		if managerCash == nil || managerCash.Sign() < 0 || managerCash.BitLen() > 256 {
			return nil, ErrInvalidState
		}
		candidate.ManagerCash[side] = uint256.MustFromBig(managerCash)
		unbacked := new(uint256.Int)
		if claims.Cmp(candidate.ManagerCash[side]) > 0 {
			unbacked.Sub(claims, candidate.ManagerCash[side])
		}
		if side == 0 {
			candidate.Unbacked0 = unbacked
		} else {
			candidate.Unbacked1 = unbacked
		}
		vault := candidate.Vaults[side]
		if vault == (common.Address{}) {
			continue
		}
		if err := requireCode(ctx, p, vault, "vault implementation", tickerVaultFingerprint.matches); err != nil {
			return nil, err
		}
		shareWord, err := storageAt(ctx, p, h.address, mapSlot(4))
		if err != nil {
			return nil, err
		}
		var strategy, asset common.Address
		var supply, pending, idle *big.Int
		request := p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
		target := hexutil.Encode(vault[:])
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: target, Method: "strategy"}, []any{&strategy})
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: target, Method: "asset"}, []any{&asset})
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: target, Method: "totalSupply"}, []any{&supply})
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: target, Method: "pendingYieldShares"}, []any{&pending})
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: hexutil.Encode(currency[:]), Method: "balanceOf", Params: []any{vault}}, []any{&idle})
		if _, err = request.Aggregate(); err != nil {
			return nil, err
		}
		if asset != currency {
			return nil, fmt.Errorf("vault asset: %w", ErrInvalidState)
		}
		expected, ok := stataProxyHashes[strategy]
		if !ok {
			return nil, fmt.Errorf("unrecognized strategy: %w", ErrInvalidState)
		}
		if err = requireCode(ctx, p, strategy, "strategy proxy bytecode", func(code []byte) bool { return crypto.Keccak256Hash(code) == expected }); err != nil {
			return nil, err
		}
		implementation, err := storageAt(ctx, p, strategy, implementationSlot)
		if err != nil {
			return nil, err
		}
		if err = requireCode(ctx, p, common.BytesToAddress(implementation[:]), "strategy implementation", func(code []byte) bool { return crypto.Keccak256Hash(code) == stataImplementationHash }); err != nil {
			return nil, err
		}
		var held, rate, maxDeposit, maxWithdraw *big.Int
		request = p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
		target = hexutil.Encode(strategy[:])
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: target, Method: "balanceOf", Params: []any{vault}}, []any{&held})
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: target, Method: "convertToAssets", Params: []any{ray.ToBig()}}, []any{&rate})
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: target, Method: "maxDeposit", Params: []any{vault}}, []any{&maxDeposit})
		request.AddCall(&ethrpc.Call{ABI: accountingABI, Target: target, Method: "maxWithdraw", Params: []any{vault}}, []any{&maxWithdraw})
		if _, err = request.Aggregate(); err != nil {
			return nil, err
		}
		for _, value := range []*big.Int{supply, pending, idle, held, rate, maxDeposit, maxWithdraw, managerCash} {
			if value == nil || value.Sign() < 0 || value.BitLen() > 256 {
				return nil, ErrInvalidState
			}
		}
		inventory := &AaveInventory{Raw: raw, Claims: claims, PoolShares: new(uint256.Int).SetBytes(shareWord[:]), Supply: uint256.MustFromBig(supply), Idle: uint256.MustFromBig(idle), StrategyShares: uint256.MustFromBig(held), Rate: uint256.MustFromBig(rate), PendingYield: uint256.MustFromBig(pending)}
		total, err := inventory.poolAssets()
		if err != nil {
			return nil, err
		}
		expectedTotal := candidate.Total0
		if side == 1 {
			expectedTotal = candidate.Total1
		}
		if total.Cmp(expectedTotal) != 0 {
			return nil, fmt.Errorf("pool share valuation: %w", ErrInvalidState)
		}
		reserve, err := readAaveReserve(ctx, p, strategy, currency, inventory.Rate)
		if err != nil {
			return nil, err
		}
		depositLimit, err := reserve.depositLimit(inventory.Rate)
		if err != nil {
			return nil, err
		}
		withdrawalLimit, err := reserve.withdrawalLimit(inventory.StrategyShares, inventory.Rate)
		if err != nil {
			return nil, err
		}
		if depositLimit.ToBig().Cmp(maxDeposit) != 0 || withdrawalLimit.ToBig().Cmp(maxWithdraw) != 0 {
			return nil, fmt.Errorf("Aave native limits: %w", ErrInvalidState)
		}
		candidate.Reserves[side] = reserve
		candidate.Aave[side] = inventory
		candidate.Limits[side] = &AaveLimits{uint256.MustFromBig(maxDeposit), uint256.MustFromBig(maxWithdraw)}
		candidate.ManagerCash[side] = uint256.MustFromBig(managerCash)
	}
	raw, err := json.Marshal(candidate.Snapshot)
	if err != nil {
		return nil, err
	}
	h.Snapshot = candidate.Snapshot
	p.Pool.BlockNumber = candidate.BlockNumber
	h.decodeErr = nil
	h.Invalid = false
	h.RefreshRequired = false
	return raw, nil
}
