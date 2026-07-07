package msgpack

import (
	"github.com/KyberNetwork/msgpack/v5"
	pancakev3_entities "github.com/KyberNetwork/pancake-v3-sdk/entities"
	uniswapv3uint256_entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	uniswapv3_entities "github.com/daoleno/uniswapv3-sdk/entities"

	pkg_source_gmx "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx"
	pkg_source_gmxglp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx-glp"
	pkg_source_madmex "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/madmex"
	pkg_source_metavault "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/metavault"
	pkg_source_quickperps "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/quickperps"
	pkg_source_swapbasedperp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/swapbased-perp"
	pkg_source_zkerafinance "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/zkera-finance"
)

func mustNotError(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_gmx.FastPriceFeedV1{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_gmx.FastPriceFeedV2{}))

	mustNotError(msgpack.RegisterConcreteType(&pkg_source_gmxglp.FastPriceFeedV1{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_gmxglp.FastPriceFeedV2{}))

	mustNotError(msgpack.RegisterConcreteType(&pkg_source_madmex.FastPriceFeedV1{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_madmex.FastPriceFeedV2{}))

	mustNotError(msgpack.RegisterConcreteType(&pkg_source_metavault.FastPriceFeedV1{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_metavault.FastPriceFeedV2{}))

	mustNotError(msgpack.RegisterConcreteType(&pancakev3_entities.TickListDataProvider{}))

	mustNotError(msgpack.RegisterConcreteType(&pkg_source_metavault.FastPriceFeedV1{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_metavault.FastPriceFeedV2{}))

	mustNotError(msgpack.RegisterConcreteType(&uniswapv3_entities.TickListDataProvider{}))

	mustNotError(msgpack.RegisterConcreteType(&uniswapv3uint256_entities.TickListDataProvider{}))

	mustNotError(msgpack.RegisterConcreteType(&pkg_source_quickperps.FastPriceFeedV1{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_quickperps.FastPriceFeedV2{}))

	mustNotError(msgpack.RegisterConcreteType(&pkg_source_swapbasedperp.FastPriceFeedV1{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_swapbasedperp.FastPriceFeedV2{}))

	mustNotError(msgpack.RegisterConcreteType(&pkg_source_zkerafinance.FastPriceFeedV1{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_zkerafinance.FastPriceFeedV2{}))
}
