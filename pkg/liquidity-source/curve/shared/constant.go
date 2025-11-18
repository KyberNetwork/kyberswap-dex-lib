package shared

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	CURVE_POOL_TYPE_STABLE_PLAIN    CurvePoolType = "curve-stable-plain"
	CURVE_POOL_TYPE_STABLE_LENDING  CurvePoolType = "curve-stable-lending"
	CURVE_POOL_TYPE_STABLE_META     CurvePoolType = "curve-stable-meta"
	CURVE_POOL_TYPE_STABLE_NG_PLAIN CurvePoolType = "curve-stable-ng-plain"
	CURVE_POOL_TYPE_STABLE_NG_META  CurvePoolType = "curve-stable-ng-meta"
	CURVE_POOL_TYPE_CRYPTO          CurvePoolType = "curve-crypto"
	CURVE_POOL_TYPE_TRICRYPTO_NG    CurvePoolType = "curve-tricrypto-ng"
	CURVE_POOL_TYPE_TWOCRYPTO_NG    CurvePoolType = "curve-twocrypto-ng"
	CURVE_POOL_TYPE_CRYPTO_META     CurvePoolType = "curve-crypto-meta"

	// https://github.com/curvefi/curve-js/blob/cb26335/src/interfaces.ts#L11
	CURVE_DATASOURCE_MAIN              CurveDataSource = "main"
	CURVE_DATASOURCE_CRYPTO            CurveDataSource = "crypto"
	CURVE_DATASOURCE_FACTORY           CurveDataSource = "factory"
	CURVE_DATASOURCE_FACTORY_CRYPTO    CurveDataSource = "factory-crypto"
	CURVE_DATASOURCE_FACTORY_CRVUSD    CurveDataSource = "factory-crvusd"
	CURVE_DATASOURCE_FACTORY_TRICRYPTO CurveDataSource = "factory-tricrypto"
	CURVE_DATASOURCE_FACTORY_TWOCRYPTO CurveDataSource = "factory-twocrypto"
	CURVE_DATASOURCE_FACTORY_STABLE_NG CurveDataSource = "factory-stable-ng"
	CURVE_DATASOURCE_FACTORY_EYWA      CurveDataSource = "factory-eywa"
)

const (
	MaxTokenCount = 8

	ERC20MethodBalanceOf   = "balanceOf"
	ERC20MethodTotalSupply = "totalSupply"
	ERC20MethodDecimals    = "decimals"

	CERC20MethodIsCToken = "isCToken"

	CurveDefaultAddressProvider = "0x5ffe7FB82894076ECB99A30D6A32e969e6e35E98"

	poolMethodGamma           = "gamma"
	poolMethodUnderlyingCoins = "underlying_coins"

	addressProviderMethodGetAddress = "get_address"

	getPoolsEndpoint = "/v1/getPools/%s/%s" // <chain>/<registry>
)

var (
	CurveAddressProvider = map[valueobject.ChainID]string{
		valueobject.ChainIDEtherlink: "0x4574921eb950d3Fd5B01562162EC566Cb8bc3648",
		valueobject.ChainIDHyperEVM:  "0x1764ee18e8B3ccA4787249Ceb249356192594585",
		valueobject.ChainIDMonad:     "0x4574921eb950d3Fd5B01562162EC566Cb8bc3648",
		valueobject.ChainIDPlasma:    "0x4574921eb950d3Fd5B01562162EC566Cb8bc3648",
		valueobject.ChainIDSonic:     "0x87FE17697D0f14A222e8bEf386a0860eCffDD617",
	}
)
