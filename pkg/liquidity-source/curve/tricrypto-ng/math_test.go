package tricryptong

import (
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCbrt(t *testing.T) {
	assert.Equal(t, uint256.MustFromDecimal("2080083823051904114"), cbrt(uint256.MustFromDecimal("9000000000000000000")))
	assert.Equal(t, uint256.MustFromDecimal("2000000000000000000"), cbrt(uint256.MustFromDecimal("8000000000000000000")))
	assert.Equal(t, uint256.MustFromDecimal("1000000000000000000"), cbrt(uint256.MustFromDecimal("1000000000000000000")))
	assert.Equal(t, uint256.MustFromDecimal("1000000000000"), cbrt(uint256.MustFromDecimal("1")))

	assert.Equal(t, uint256.MustFromDecimal("3409609420638816171"), cbrt(uint256.MustFromDecimal("39638197472940459652")))
	assert.Equal(t, uint256.MustFromDecimal("3158674443496939484"), cbrt(uint256.MustFromDecimal("31514803223928387165")))
	assert.Equal(t, uint256.MustFromDecimal("1856232217231345934"), cbrt(uint256.MustFromDecimal("6395830097435958518")))
	assert.Equal(t, uint256.MustFromDecimal("3524826715323100363"), cbrt(uint256.MustFromDecimal("43793868931296244089")))
	assert.Equal(t, uint256.MustFromDecimal("4187864666460287008"), cbrt(uint256.MustFromDecimal("73447651917585987727")))
	assert.Equal(t, uint256.MustFromDecimal("2468777895279877116"), cbrt(uint256.MustFromDecimal("15046866249244751554")))
	assert.Equal(t, uint256.MustFromDecimal("4469815408513126588"), cbrt(uint256.MustFromDecimal("89303558544806071690")))
	assert.Equal(t, uint256.MustFromDecimal("4625328409064976070"), cbrt(uint256.MustFromDecimal("98952716746955562190")))
	assert.Equal(t, uint256.MustFromDecimal("4548539715828898244"), cbrt(uint256.MustFromDecimal("94105709505396938196")))
	assert.Equal(t, uint256.MustFromDecimal("3673416897699794768"), cbrt(uint256.MustFromDecimal("49569057144019925136")))
	assert.Equal(t, uint256.MustFromDecimal("2362419845455654115"), cbrt(uint256.MustFromDecimal("13184730185935573520")))
	assert.Equal(t, uint256.MustFromDecimal("2344178045855768927"), cbrt(uint256.MustFromDecimal("12881658538187347937")))
	assert.Equal(t, uint256.MustFromDecimal("3673880058852684590"), cbrt(uint256.MustFromDecimal("49587809186428770707")))
	assert.Equal(t, uint256.MustFromDecimal("4386504252680991788"), cbrt(uint256.MustFromDecimal("84402568722244644433")))
	assert.Equal(t, uint256.MustFromDecimal("3888393165254530811"), cbrt(uint256.MustFromDecimal("58790954774677425911")))
	assert.Equal(t, uint256.MustFromDecimal("4555814634348496939"), cbrt(uint256.MustFromDecimal("94557969105517983451")))
	assert.Equal(t, uint256.MustFromDecimal("3981029051310698240"), cbrt(uint256.MustFromDecimal("63093706398058068219")))
	assert.Equal(t, uint256.MustFromDecimal("4558449262620664845"), cbrt(uint256.MustFromDecimal("94722112655436198453")))
	assert.Equal(t, uint256.MustFromDecimal("2748818881504641819"), cbrt(uint256.MustFromDecimal("20770089881576278309")))
	assert.Equal(t, uint256.MustFromDecimal("4256607066661130706"), cbrt(uint256.MustFromDecimal("77124202293076254502")))
	assert.Equal(t, uint256.MustFromDecimal("4455595329061464586"), cbrt(uint256.MustFromDecimal("88453947644288418009")))
	assert.Equal(t, uint256.MustFromDecimal("2208934008153983984"), cbrt(uint256.MustFromDecimal("10778249300388314410")))
	assert.Equal(t, uint256.MustFromDecimal("4045716245899149624"), cbrt(uint256.MustFromDecimal("66219555050645902876")))
	assert.Equal(t, uint256.MustFromDecimal("4040302918631942611"), cbrt(uint256.MustFromDecimal("65954097462384673949")))
	assert.Equal(t, uint256.MustFromDecimal("2858489406665044483"), cbrt(uint256.MustFromDecimal("23356607427460460977")))
	assert.Equal(t, uint256.MustFromDecimal("4056474950468156649"), cbrt(uint256.MustFromDecimal("66749250785174326058")))
	assert.Equal(t, uint256.MustFromDecimal("3470257887884324264"), cbrt(uint256.MustFromDecimal("41791239299025366018")))
	assert.Equal(t, uint256.MustFromDecimal("4128402790283034868"), cbrt(uint256.MustFromDecimal("70363298264528807179")))
	assert.Equal(t, uint256.MustFromDecimal("3526413842616623686"), cbrt(uint256.MustFromDecimal("43853052901221995056")))
	assert.Equal(t, uint256.MustFromDecimal("3415542418485287260"), cbrt(uint256.MustFromDecimal("39845478808679822889")))
	assert.Equal(t, uint256.MustFromDecimal("2939557905130632850"), cbrt(uint256.MustFromDecimal("25400721850125252260")))
	assert.Equal(t, uint256.MustFromDecimal("4206256023393282213"), cbrt(uint256.MustFromDecimal("74419562139461252527")))
	assert.Equal(t, uint256.MustFromDecimal("3261184285411192906"), cbrt(uint256.MustFromDecimal("34683748053331305215")))
	assert.Equal(t, uint256.MustFromDecimal("3630467098993525152"), cbrt(uint256.MustFromDecimal("47850614126281462690")))
	assert.Equal(t, uint256.MustFromDecimal("3124048797125245052"), cbrt(uint256.MustFromDecimal("30489719334795299057")))
	assert.Equal(t, uint256.MustFromDecimal("2929983213371115048"), cbrt(uint256.MustFromDecimal("25153324667885994091")))
	assert.Equal(t, uint256.MustFromDecimal("2792423642659410418"), cbrt(uint256.MustFromDecimal("21774285810458041019")))
	assert.Equal(t, uint256.MustFromDecimal("4523315548248715022"), cbrt(uint256.MustFromDecimal("92548771030453170946")))
	assert.Equal(t, uint256.MustFromDecimal("2354141187123277916"), cbrt(uint256.MustFromDecimal("13046605092170978360")))
	assert.Equal(t, uint256.MustFromDecimal("4118195484032492575"), cbrt(uint256.MustFromDecimal("69842676514201980104")))
	assert.Equal(t, uint256.MustFromDecimal("3288496488100488535"), cbrt(uint256.MustFromDecimal("35562488818755808579")))
	assert.Equal(t, uint256.MustFromDecimal("4575946389050901921"), cbrt(uint256.MustFromDecimal("95817047211660172078")))
	assert.Equal(t, uint256.MustFromDecimal("2961582149183954049"), cbrt(uint256.MustFromDecimal("25975944707211662726")))
	assert.Equal(t, uint256.MustFromDecimal("4257787923902810799"), cbrt(uint256.MustFromDecimal("77188406908758470310")))
	assert.Equal(t, uint256.MustFromDecimal("3716897211340187582"), cbrt(uint256.MustFromDecimal("51350142518998422961")))
	assert.Equal(t, uint256.MustFromDecimal("3653281779914996139"), cbrt(uint256.MustFromDecimal("48758407506467183165")))
	assert.Equal(t, uint256.MustFromDecimal("4250110072626739673"), cbrt(uint256.MustFromDecimal("76771589714941574954")))
	assert.Equal(t, uint256.MustFromDecimal("4410936878351629951"), cbrt(uint256.MustFromDecimal("85820794124947375258")))
	assert.Equal(t, uint256.MustFromDecimal("4181162923347933638"), cbrt(uint256.MustFromDecimal("73095606146265577015")))
	assert.Equal(t, uint256.MustFromDecimal("1869039676918329898"), cbrt(uint256.MustFromDecimal("6529133711418056774")))
	assert.Equal(t, uint256.MustFromDecimal("4445756704268882349"), cbrt(uint256.MustFromDecimal("87869281706658851945")))
	assert.Equal(t, uint256.MustFromDecimal("2481720255736366816"), cbrt(uint256.MustFromDecimal("15284754804775270319")))
	assert.Equal(t, uint256.MustFromDecimal("4500875554903673351"), cbrt(uint256.MustFromDecimal("91178200310120609507")))
	assert.Equal(t, uint256.MustFromDecimal("4081007172090215278"), cbrt(uint256.MustFromDecimal("67967621785671730115")))
	assert.Equal(t, uint256.MustFromDecimal("2075022505038574562"), cbrt(uint256.MustFromDecimal("8934462572922967033")))
	assert.Equal(t, uint256.MustFromDecimal("1794185734853014464"), cbrt(uint256.MustFromDecimal("5775667696883795287")))
	assert.Equal(t, uint256.MustFromDecimal("3989313006130366807"), cbrt(uint256.MustFromDecimal("63488393615732029472")))
	assert.Equal(t, uint256.MustFromDecimal("2396114951232700270"), cbrt(uint256.MustFromDecimal("13756974972609928294")))
	assert.Equal(t, uint256.MustFromDecimal("4022855213744342480"), cbrt(uint256.MustFromDecimal("65103330527939662293")))
	assert.Equal(t, uint256.MustFromDecimal("4297701580975102819"), cbrt(uint256.MustFromDecimal("79379574831764206951")))
	assert.Equal(t, uint256.MustFromDecimal("3767972190502361796"), cbrt(uint256.MustFromDecimal("53496216337683145273")))
	assert.Equal(t, uint256.MustFromDecimal("4184602103965571404"), cbrt(uint256.MustFromDecimal("73276127090639580767")))
	assert.Equal(t, uint256.MustFromDecimal("3035383688797450737"), cbrt(uint256.MustFromDecimal("27966671946998014451")))
	assert.Equal(t, uint256.MustFromDecimal("3561530741909140962"), cbrt(uint256.MustFromDecimal("45176241060629919277")))
	assert.Equal(t, uint256.MustFromDecimal("3398500895106248130"), cbrt(uint256.MustFromDecimal("39252033961533644730")))
	assert.Equal(t, uint256.MustFromDecimal("3473625660542181604"), cbrt(uint256.MustFromDecimal("41913028539491435462")))
	assert.Equal(t, uint256.MustFromDecimal("1663295954945230577"), cbrt(uint256.MustFromDecimal("4601597135474867054")))
	assert.Equal(t, uint256.MustFromDecimal("4161479238637871243"), cbrt(uint256.MustFromDecimal("72068120647825333469")))
	assert.Equal(t, uint256.MustFromDecimal("2490026845523177655"), cbrt(uint256.MustFromDecimal("15438748340168276082")))
	assert.Equal(t, uint256.MustFromDecimal("2308451681853867250"), cbrt(uint256.MustFromDecimal("12301621668122832714")))
	assert.Equal(t, uint256.MustFromDecimal("3644992784995359482"), cbrt(uint256.MustFromDecimal("48427273549373148100")))
	assert.Equal(t, uint256.MustFromDecimal("3626693500016573012"), cbrt(uint256.MustFromDecimal("47701557764695278776")))
	assert.Equal(t, uint256.MustFromDecimal("4113280324356244440"), cbrt(uint256.MustFromDecimal("69592898413781159022")))
	assert.Equal(t, uint256.MustFromDecimal("4008462274982311635"), cbrt(uint256.MustFromDecimal("64407049126309813316")))
}

func TestExp(t *testing.T) {
	_, err := _snekmate_wad_exp(i256.MustFromDecimal("135305999368893231589"))
	require.NotNil(t, err)
	_, err = _snekmate_wad_exp(i256.MustFromDecimal("135305999368893231590"))
	require.NotNil(t, err)

	testcases := []struct {
		x   string
		exp string
	}{
		{"-42139678854452767551", "0"},
		{"-42139678854452767552", "0"},
		{"-10", "999999999999999990"},
		{"-8293361", "999999999991706639"},
		{"-8293361234", "999999991706638800"},
		{"10", "1000000000000000010"},
		{"8293361", "1000000000008293361"},
		{"8293361234", "1000000008293361268"},
	}

	for _, tc := range testcases {
		r, err := _snekmate_wad_exp(i256.MustFromDecimal(tc.x))
		require.Nil(t, err)
		assert.Equal(t, tc.exp, r.Dec())
	}
}

func TestGetP(t *testing.T) {

	testcases := []struct {
		xp0   string
		xp1   string
		xp2   string
		d     string
		a     string
		gamma string

		out1 string
		out2 string
	}{
		{"3848079558071253519125552", "4044947947211999230664846", "4100762045474938390484016", "11990883592660056140229065", "540000", "80500000000000",
			"959209273598579670", "948357297280457582"},
	}

	for _, tc := range testcases {
		var xp [NumTokens]uint256.Int
		xp[0].SetFromDecimal(tc.xp0)
		xp[1].SetFromDecimal(tc.xp1)
		xp[2].SetFromDecimal(tc.xp2)
		var out [NumTokens - 1]uint256.Int
		err := get_p(xp, uint256.MustFromDecimal(tc.d), uint256.MustFromDecimal(tc.a), uint256.MustFromDecimal(tc.gamma), out[:])
		require.Nil(t, err)
		assert.Equal(t, tc.out1, out[0].Dec())
		assert.Equal(t, tc.out2, out[1].Dec())
	}
}
