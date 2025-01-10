package shared

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

	ERC20MethodTotalSupply = "totalSupply"

	CERC20MethodIsCToken = "isCToken"

	// might not available for all chains, but we only need for some chains so it's ok
	CurveAddressProvider = "0x0000000022d53366457f9d5e68ec105046fc4383"

	// https://sonicscan.org/address/0x87fe17697d0f14a222e8bef386a0860ecffdd617#code
	CurveAddressProvider_Sonic = "0x87fe17697d0f14a222e8bef386a0860ecffdd617"

	poolMethodGamma           = "gamma"
	poolMethodUnderlyingCoins = "underlying_coins"

	addressProviderMethodGetAddress = "get_address"

	getPoolsEndpoint = "/v1/getPools/%s/%s" // <chain>/<registry>
)
