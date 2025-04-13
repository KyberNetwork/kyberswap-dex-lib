package math

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	TwoPow31  = new(big.Int).Lsh(bignum.One, 31)
	TwoPow32  = new(big.Int).Lsh(bignum.One, 32)
	TwoPow64  = new(big.Int).Lsh(bignum.One, 64)
	TwoPow96  = new(big.Int).Lsh(bignum.One, 96)
	TwoPow127 = new(big.Int).Lsh(bignum.One, 127)
	TwoPow128 = new(big.Int).Lsh(bignum.One, 128)
	TwoPow160 = new(big.Int).Lsh(bignum.One, 160)
	TwoPow256 = new(big.Int).Lsh(bignum.One, 256)

	U32Max  = new(big.Int).Sub(TwoPow32, bignum.One)
	U96Max  = new(big.Int).Sub(TwoPow96, bignum.One)
	U128Max = bignum.MAX_UINT_128
	U256Max = bignum.MAX_UINT_256

	tickMasks = [27]*big.Int{
		bignum.NewBig("340282196779882608775400081051345954875"),
		bignum.NewBig("340282026638911824551550055881712329744"),
		bignum.NewBig("340281686357225467326082729798982530761"),
		bignum.NewBig("340281005794873596573094710012989410072"),
		bignum.NewBig("340279644674253220331048947518555486588"),
		bignum.NewBig("340276922449345852680114346161748269428"),
		bignum.NewBig("340271478064864046934932689706268169160"),
		bignum.NewBig("340260589557227275536970734808027446368"),
		bignum.NewBig("340238813587222069206590352882444122789"),
		bignum.NewBig("340195265827972829906238800850533898084"),
		bignum.NewBig("340108187030021964395011332102514443192"),
		bignum.NewBig("339934096296338836553381438728885759063"),
		bignum.NewBig("339586182118148867994722769619739863337"),
		bignum.NewBig("338891421642114786711725022995822495130"),
		bignum.NewBig("337506162020136007223375792700736547955"),
		bignum.NewBig("334752606878476170032608184381010470119"),
		bignum.NewBig("329312708224965356314372077522414179198"),
		bignum.NewBig("318696677643769341895946499718036526719"),
		bignum.NewBig("298480268784467865547185399201474874621"),
		bignum.NewBig("261813363001402740542020888190605903085"),
		bignum.NewBig("201439286044552466954512953679502885114"),
		bignum.NewBig("119247395418425876229630149779831629514"),
		bignum.NewBig("41788651709309085859755118322591422179"),
		bignum.NewBig("5131889223303998677939799486543748348"),
		bignum.NewBig("77395391476111124436543048938586576"),
		bignum.NewBig("17603164912545197391926702422886"),
		bignum.NewBig("910630244353091011850553"),
	}
)

const logBaseSqrtTickSize = 4.9999975000016666654166676666658333340476184226196031741031750577196410537756684185262518589393595459766211405607685305832e-7
