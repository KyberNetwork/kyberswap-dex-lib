package twamm

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	masksAndFactors = [...]maskAndFactor{
		{
			mask:   uint256.MustFromHex("0x8000000000000000"),
			factor: big256.New("481231938336009023090067544955250113854"),
		},
		{
			mask:   uint256.MustFromHex("0x4000000000000000"),
			factor: big256.New("404666211852346594250993303657235475948"),
		},
		{
			mask:   uint256.MustFromHex("0x2000000000000000"),
			factor: big256.New("371080552416919877990254144423618836767"),
		},
		{
			mask:   uint256.MustFromHex("0x1000000000000000"),
			factor: big256.New("355347954397881497469693820312941593443"),
		},
		{
			mask:   uint256.MustFromHex("0x800000000000000"),
			factor: big256.New("347733580493780928808815525413232318461"),
		},
		{
			mask:   uint256.MustFromHex("0x400000000000000"),
			factor: big256.New("343987798952690256687074238090730651112"),
		},
		{
			mask:   uint256.MustFromHex("0x200000000000000"),
			factor: big256.New("342130066523749645191881555545647086143"),
		},
		{
			mask:   uint256.MustFromHex("0x100000000000000"),
			factor: big256.New("341204966012395051463589306197117539651"),
		},
		{
			mask:   uint256.MustFromHex("0x80000000000000"),
			factor: big256.New("340743354212339922144397487283364652955"),
		},
		{
			mask:   uint256.MustFromHex("0x40000000000000"),
			factor: big256.New("340512782555889898808859563671008026639"),
		},
		{
			mask:   uint256.MustFromHex("0x20000000000000"),
			factor: big256.New("340397555242326998647385072673097901535"),
		},
		{
			mask:   uint256.MustFromHex("0x10000000000000"),
			factor: big256.New("340339956208435708755752659506489956133"),
		},
		{
			mask:   uint256.MustFromHex("0x8000000000000"),
			factor: big256.New("340311160346490911934870813363085081661"),
		},
		{
			mask:   uint256.MustFromHex("0x4000000000000"),
			factor: big256.New("340296763329178528376528243588334151603"),
		},
		{
			mask:   uint256.MustFromHex("0x2000000000000"),
			factor: big256.New("340289565048926066557319684044576333862"),
		},
		{
			mask:   uint256.MustFromHex("0x1000000000000"),
			factor: big256.New("340285965965899358974465315064323671036"),
		},
		{
			mask:   uint256.MustFromHex("0x800000000000"),
			factor: big256.New("340284166438660709872813645066166128555"),
		},
		{
			mask:   uint256.MustFromHex("0x400000000000"),
			factor: big256.New("340283266678610039476911010529773336521"),
		},
		{
			mask:   uint256.MustFromHex("0x200000000000"),
			factor: big256.New("340282816799476865065514053322893021549"),
		},
		{
			mask:   uint256.MustFromHex("0x100000000000"),
			factor: big256.New("340282591860133317712432962519222523747"),
		},
		{
			mask:   uint256.MustFromHex("0x80000000000"),
			factor: big256.New("340282479390517303956044167089727739432"),
		},
		{
			mask:   uint256.MustFromHex("0x40000000000"),
			factor: big256.New("340282423155723237052512385577070742059"),
		},
		{
			mask:   uint256.MustFromHex("0x20000000000"),
			factor: big256.New("340282395038329688593740233918090740389"),
		},
		{
			mask:   uint256.MustFromHex("0x10000000000"),
			factor: big256.New("340282380979633785612518603506803612670"),
		},
		{
			mask:   uint256.MustFromHex("0x8000000000"),
			factor: big256.New("340282373950286051933938400987007267567"),
		},
		{
			mask:   uint256.MustFromHex("0x4000000000"),
			factor: big256.New("340282370435612239547654640565033792378"),
		},
		{
			mask:   uint256.MustFromHex("0x2000000000"),
			factor: big256.New("340282368678275346967764181521839267590"),
		},
		{
			mask:   uint256.MustFromHex("0x1000000000"),
			factor: big256.New("340282367799606904081131786786979136761"),
		},
		{
			mask:   uint256.MustFromHex("0x800000000"),
			factor: big256.New("340282367360272683488643795553082001443"),
		},
		{
			mask:   uint256.MustFromHex("0x400000000"),
			factor: big256.New("340282367140605573405106851149122747984"),
		},
		{
			mask:   uint256.MustFromHex("0x200000000"),
			factor: big256.New("340282367030772018416515141710341210063"),
		},
		{
			mask:   uint256.MustFromHex("0x100000000"),
			factor: big256.New("340282366975855240935513477676743808340"),
		},
		{
			mask:   uint256.MustFromHex("0x80000000"),
			factor: big256.New("340282366948396852198336193330767679917"),
		},
		{
			mask:   uint256.MustFromHex("0x40000000"),
			factor: big256.New("340282366934667657830578438075407037644"),
		},
		{
			mask:   uint256.MustFromHex("0x20000000"),
			factor: big256.New("340282366927803060646907282177123794346"),
		},
		{
			mask:   uint256.MustFromHex("0x10000000"),
			factor: big256.New("340282366924370762055123634660330219950"),
		},
		{
			mask:   uint256.MustFromHex("0x8000000"),
			factor: big256.New("340282366922654612759244793510020291790"),
		},
		{
			mask:   uint256.MustFromHex("0x4000000"),
			factor: big256.New("340282366921796538111308618586887023373"),
		},
		{
			mask:   uint256.MustFromHex("0x2000000"),
			factor: big256.New("340282366921367500787341342538325810693"),
		},
		{
			mask:   uint256.MustFromHex("0x1000000"),
			factor: big256.New("340282366921152982125357907367296559436"),
		},
		{
			mask:   uint256.MustFromHex("0x800000"),
			factor: big256.New("340282366921045722794366240495094772541"),
		},
		{
			mask:   uint256.MustFromHex("0x400000"),
			factor: big256.New("340282366920992093128870419737322088773"),
		},
		{
			mask:   uint256.MustFromHex("0x200000"),
			factor: big256.New("340282366920965278296122512528017799308"),
		},
		{
			mask:   uint256.MustFromHex("0x100000"),
			factor: big256.New("340282366920951870879748559715761167680"),
		},
		{
			mask:   uint256.MustFromHex("0x80000"),
			factor: big256.New("340282366920945167171561583507731730142"),
		},
		{
			mask:   uint256.MustFromHex("0x40000"),
			factor: big256.New("340282366920941815317468095453241730942"),
		},
		{
			mask:   uint256.MustFromHex("0x20000"),
			factor: big256.New("340282366920940139390421351438377911234"),
		},
		{
			mask:   uint256.MustFromHex("0x10000"),
			factor: big256.New("340282366920939301426897979434041296353"),
		},
		{
			mask:   uint256.MustFromHex("0x8000"),
			factor: big256.New("340282366920938882445136293432646812656"),
		},
		{
			mask:   uint256.MustFromHex("0x4000"),
			factor: big256.New("340282366920938672954255450432143026744"),
		},
		{
			mask:   uint256.MustFromHex("0x2000"),
			factor: big256.New("340282366920938568208815028931939497772"),
		},
		{
			mask:   uint256.MustFromHex("0x1000"),
			factor: big256.New("340282366920938515836094818181849824282"),
		},
		{
			mask:   uint256.MustFromHex("0x800"),
			factor: big256.New("340282366920938489649734712806808010286"),
		},
		{
			mask:   uint256.MustFromHex("0x400"),
			factor: big256.New("340282366920938476556554660119287858975"),
		},
		{
			mask:   uint256.MustFromHex("0x200"),
			factor: big256.New("340282366920938470009964633775527972241"),
		},
		{
			mask:   uint256.MustFromHex("0x100"),
			factor: big256.New("340282366920938466736669620603648076105"),
		},
		{
			mask:   uint256.MustFromHex("0x80"),
			factor: big256.New("340282366920938465100022114017708139844"),
		},
		{
			mask:   uint256.MustFromHex("0x40"),
			factor: big256.New("340282366920938464281698360724738174666"),
		},
		{
			mask:   uint256.MustFromHex("0x20"),
			factor: big256.New("340282366920938463872536484078253192815"),
		},
		{
			mask:   uint256.MustFromHex("0x10"),
			factor: big256.New("340282366920938463667955545755010702074"),
		},
		{
			mask:   uint256.MustFromHex("0x8"),
			factor: big256.New("340282366920938463565665076593389456749"),
		},
		{
			mask:   uint256.MustFromHex("0x4"),
			factor: big256.New("340282366920938463514519842012578834098"),
		},
		{
			mask:   uint256.MustFromHex("0x2"),
			factor: big256.New("340282366920938463488947224722173522776"),
		},
		{
			mask:   uint256.MustFromHex("0x1"),
			factor: big256.New("340282366920938463476160916076970867115"),
		},
	}

	// 64 << 64
	exponentLimit = uint256.MustFromHex("0x400000000000000000")
)

type maskAndFactor struct {
	mask   *uint256.Int
	factor *uint256.Int
}

func exp2(x *uint256.Int) *uint256.Int {
	if !x.Lt(exponentLimit) {
		return nil
	}

	result := big256.U2Pow127.Clone()
	var helper uint256.Int

	for _, maskAndFactor := range masksAndFactors {
		if !helper.And(x, maskAndFactor.mask).IsZero() {
			result.Rsh(result.Mul(result, maskAndFactor.factor), 128)
		}
	}

	return result.Rsh(result, 63-uint(helper.Rsh(x, 64).Uint64()))
}
