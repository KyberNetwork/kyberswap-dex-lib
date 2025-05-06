package twamm

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type maskAndFactor struct {
	mask   *big.Int
	factor *big.Int
}

var masksAndFactors = [...]maskAndFactor{
	{
		mask:   bignum.NewBig("0x8000000000000000"),
		factor: bignum.NewBig("481231938336009023090067544955250113854"),
	},
	{
		mask:   bignum.NewBig("0x4000000000000000"),
		factor: bignum.NewBig("404666211852346594250993303657235475948"),
	},
	{
		mask:   bignum.NewBig("0x2000000000000000"),
		factor: bignum.NewBig("371080552416919877990254144423618836767"),
	},
	{
		mask:   bignum.NewBig("0x1000000000000000"),
		factor: bignum.NewBig("355347954397881497469693820312941593443"),
	},
	{
		mask:   bignum.NewBig("0x800000000000000"),
		factor: bignum.NewBig("347733580493780928808815525413232318461"),
	},
	{
		mask:   bignum.NewBig("0x400000000000000"),
		factor: bignum.NewBig("343987798952690256687074238090730651112"),
	},
	{
		mask:   bignum.NewBig("0x200000000000000"),
		factor: bignum.NewBig("342130066523749645191881555545647086143"),
	},
	{
		mask:   bignum.NewBig("0x100000000000000"),
		factor: bignum.NewBig("341204966012395051463589306197117539651"),
	},
	{
		mask:   bignum.NewBig("0x80000000000000"),
		factor: bignum.NewBig("340743354212339922144397487283364652955"),
	},
	{
		mask:   bignum.NewBig("0x40000000000000"),
		factor: bignum.NewBig("340512782555889898808859563671008026639"),
	},
	{
		mask:   bignum.NewBig("0x20000000000000"),
		factor: bignum.NewBig("340397555242326998647385072673097901535"),
	},
	{
		mask:   bignum.NewBig("0x10000000000000"),
		factor: bignum.NewBig("340339956208435708755752659506489956133"),
	},
	{
		mask:   bignum.NewBig("0x8000000000000"),
		factor: bignum.NewBig("340311160346490911934870813363085081661"),
	},
	{
		mask:   bignum.NewBig("0x4000000000000"),
		factor: bignum.NewBig("340296763329178528376528243588334151603"),
	},
	{
		mask:   bignum.NewBig("0x2000000000000"),
		factor: bignum.NewBig("340289565048926066557319684044576333862"),
	},
	{
		mask:   bignum.NewBig("0x1000000000000"),
		factor: bignum.NewBig("340285965965899358974465315064323671036"),
	},
	{
		mask:   bignum.NewBig("0x800000000000"),
		factor: bignum.NewBig("340284166438660709872813645066166128555"),
	},
	{
		mask:   bignum.NewBig("0x400000000000"),
		factor: bignum.NewBig("340283266678610039476911010529773336521"),
	},
	{
		mask:   bignum.NewBig("0x200000000000"),
		factor: bignum.NewBig("340282816799476865065514053322893021549"),
	},
	{
		mask:   bignum.NewBig("0x100000000000"),
		factor: bignum.NewBig("340282591860133317712432962519222523747"),
	},
	{
		mask:   bignum.NewBig("0x80000000000"),
		factor: bignum.NewBig("340282479390517303956044167089727739432"),
	},
	{
		mask:   bignum.NewBig("0x40000000000"),
		factor: bignum.NewBig("340282423155723237052512385577070742059"),
	},
	{
		mask:   bignum.NewBig("0x20000000000"),
		factor: bignum.NewBig("340282395038329688593740233918090740389"),
	},
	{
		mask:   bignum.NewBig("0x10000000000"),
		factor: bignum.NewBig("340282380979633785612518603506803612670"),
	},
	{
		mask:   bignum.NewBig("0x8000000000"),
		factor: bignum.NewBig("340282373950286051933938400987007267567"),
	},
	{
		mask:   bignum.NewBig("0x4000000000"),
		factor: bignum.NewBig("340282370435612239547654640565033792378"),
	},
	{
		mask:   bignum.NewBig("0x2000000000"),
		factor: bignum.NewBig("340282368678275346967764181521839267590"),
	},
	{
		mask:   bignum.NewBig("0x1000000000"),
		factor: bignum.NewBig("340282367799606904081131786786979136761"),
	},
	{
		mask:   bignum.NewBig("0x800000000"),
		factor: bignum.NewBig("340282367360272683488643795553082001443"),
	},
	{
		mask:   bignum.NewBig("0x400000000"),
		factor: bignum.NewBig("340282367140605573405106851149122747984"),
	},
	{
		mask:   bignum.NewBig("0x200000000"),
		factor: bignum.NewBig("340282367030772018416515141710341210063"),
	},
	{
		mask:   bignum.NewBig("0x100000000"),
		factor: bignum.NewBig("340282366975855240935513477676743808340"),
	},
	{
		mask:   bignum.NewBig("0x80000000"),
		factor: bignum.NewBig("340282366948396852198336193330767679917"),
	},
	{
		mask:   bignum.NewBig("0x40000000"),
		factor: bignum.NewBig("340282366934667657830578438075407037644"),
	},
	{
		mask:   bignum.NewBig("0x20000000"),
		factor: bignum.NewBig("340282366927803060646907282177123794346"),
	},
	{
		mask:   bignum.NewBig("0x10000000"),
		factor: bignum.NewBig("340282366924370762055123634660330219950"),
	},
	{
		mask:   bignum.NewBig("0x8000000"),
		factor: bignum.NewBig("340282366922654612759244793510020291790"),
	},
	{
		mask:   bignum.NewBig("0x4000000"),
		factor: bignum.NewBig("340282366921796538111308618586887023373"),
	},
	{
		mask:   bignum.NewBig("0x2000000"),
		factor: bignum.NewBig("340282366921367500787341342538325810693"),
	},
	{
		mask:   bignum.NewBig("0x1000000"),
		factor: bignum.NewBig("340282366921152982125357907367296559436"),
	},
	{
		mask:   bignum.NewBig("0x800000"),
		factor: bignum.NewBig("340282366921045722794366240495094772541"),
	},
	{
		mask:   bignum.NewBig("0x400000"),
		factor: bignum.NewBig("340282366920992093128870419737322088773"),
	},
	{
		mask:   bignum.NewBig("0x200000"),
		factor: bignum.NewBig("340282366920965278296122512528017799308"),
	},
	{
		mask:   bignum.NewBig("0x100000"),
		factor: bignum.NewBig("340282366920951870879748559715761167680"),
	},
	{
		mask:   bignum.NewBig("0x80000"),
		factor: bignum.NewBig("340282366920945167171561583507731730142"),
	},
	{
		mask:   bignum.NewBig("0x40000"),
		factor: bignum.NewBig("340282366920941815317468095453241730942"),
	},
	{
		mask:   bignum.NewBig("0x20000"),
		factor: bignum.NewBig("340282366920940139390421351438377911234"),
	},
	{
		mask:   bignum.NewBig("0x10000"),
		factor: bignum.NewBig("340282366920939301426897979434041296353"),
	},
	{
		mask:   bignum.NewBig("0x8000"),
		factor: bignum.NewBig("340282366920938882445136293432646812656"),
	},
	{
		mask:   bignum.NewBig("0x4000"),
		factor: bignum.NewBig("340282366920938672954255450432143026744"),
	},
	{
		mask:   bignum.NewBig("0x2000"),
		factor: bignum.NewBig("340282366920938568208815028931939497772"),
	},
	{
		mask:   bignum.NewBig("0x1000"),
		factor: bignum.NewBig("340282366920938515836094818181849824282"),
	},
	{
		mask:   bignum.NewBig("0x800"),
		factor: bignum.NewBig("340282366920938489649734712806808010286"),
	},
	{
		mask:   bignum.NewBig("0x400"),
		factor: bignum.NewBig("340282366920938476556554660119287858975"),
	},
	{
		mask:   bignum.NewBig("0x200"),
		factor: bignum.NewBig("340282366920938470009964633775527972241"),
	},
	{
		mask:   bignum.NewBig("0x100"),
		factor: bignum.NewBig("340282366920938466736669620603648076105"),
	},
	{
		mask:   bignum.NewBig("0x80"),
		factor: bignum.NewBig("340282366920938465100022114017708139844"),
	},
	{
		mask:   bignum.NewBig("0x40"),
		factor: bignum.NewBig("340282366920938464281698360724738174666"),
	},
	{
		mask:   bignum.NewBig("0x20"),
		factor: bignum.NewBig("340282366920938463872536484078253192815"),
	},
	{
		mask:   bignum.NewBig("0x10"),
		factor: bignum.NewBig("340282366920938463667955545755010702074"),
	},
	{
		mask:   bignum.NewBig("0x8"),
		factor: bignum.NewBig("340282366920938463565665076593389456749"),
	},
	{
		mask:   bignum.NewBig("0x4"),
		factor: bignum.NewBig("340282366920938463514519842012578834098"),
	},
	{
		mask:   bignum.NewBig("0x2"),
		factor: bignum.NewBig("340282366920938463488947224722173522776"),
	},
	{
		mask:   bignum.NewBig("0x1"),
		factor: bignum.NewBig("340282366920938463476160916076970867115"),
	},
}

// 64 << 64
var exponentLimit = bignum.NewBig("0x400000000000000000")

func exp2(x *big.Int) *big.Int {
	if x.Cmp(exponentLimit) != -1 {
		return nil
	}

	result := new(big.Int).Set(math.TwoPow127)
	helper := new(big.Int)

	for _, maskAndFactor := range masksAndFactors {
		if helper.And(x, maskAndFactor.mask).Sign() != 0 {
			result.Rsh(result.Mul(result, maskAndFactor.factor), 128)
		}
	}

	return result.Rsh(result, 63-uint(helper.Rsh(x, 64).Uint64()))
}
