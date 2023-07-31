package crowdswapv2

import (
	"errors"
	"math/big"
)

var MinPrecision = uint(32)
var MaxPrecision = uint(127)

var Fixed1 *big.Int
var Fixed2 *big.Int

var Ln2Numerator *big.Int
var Ln2Denominator *big.Int

var OptLogMaxVal *big.Int
var OptExpMaxVal *big.Int

var One *big.Int
var Zero *big.Int

var maxExpArray []*big.Int

func NewBig(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 0)
	return res
}

func init() {
	Fixed1 = NewBig("0x080000000000000000000000000000000")
	Fixed2 = NewBig("0x100000000000000000000000000000000")

	Ln2Numerator = NewBig("0x3f80fe03f80fe03f80fe03f80fe03f8")
	Ln2Denominator = NewBig("0x5b9de1d10bf4103d647b0955897ba80")

	OptLogMaxVal = NewBig("0x15bf0a8b1457695355fb8ac404e7a79e3")
	OptExpMaxVal = NewBig("0x800000000000000000000000000000000")

	One = big.NewInt(1)
	Zero = big.NewInt(0)

	maxExpArray = make([]*big.Int, 128)

	maxExpArray[32] = NewBig("0x1c35fedd14ffffffffffffffffffffffff")
	maxExpArray[33] = NewBig("0x1b0ce43b323fffffffffffffffffffffff")
	maxExpArray[34] = NewBig("0x19f0028ec1ffffffffffffffffffffffff")
	maxExpArray[35] = NewBig("0x18ded91f0e7fffffffffffffffffffffff")
	maxExpArray[36] = NewBig("0x17d8ec7f0417ffffffffffffffffffffff")
	maxExpArray[37] = NewBig("0x16ddc6556cdbffffffffffffffffffffff")
	maxExpArray[38] = NewBig("0x15ecf52776a1ffffffffffffffffffffff")
	maxExpArray[39] = NewBig("0x15060c256cb2ffffffffffffffffffffff")
	maxExpArray[40] = NewBig("0x1428a2f98d72ffffffffffffffffffffff")
	maxExpArray[41] = NewBig("0x13545598e5c23fffffffffffffffffffff")
	maxExpArray[42] = NewBig("0x1288c4161ce1dfffffffffffffffffffff")
	maxExpArray[43] = NewBig("0x11c592761c666fffffffffffffffffffff")
	maxExpArray[44] = NewBig("0x110a688680a757ffffffffffffffffffff")
	maxExpArray[45] = NewBig("0x1056f1b5bedf77ffffffffffffffffffff")
	maxExpArray[46] = NewBig("0x0faadceceeff8bffffffffffffffffffff")
	maxExpArray[47] = NewBig("0x0f05dc6b27edadffffffffffffffffffff")
	maxExpArray[48] = NewBig("0x0e67a5a25da4107fffffffffffffffffff")
	maxExpArray[49] = NewBig("0x0dcff115b14eedffffffffffffffffffff")
	maxExpArray[50] = NewBig("0x0d3e7a392431239fffffffffffffffffff")
	maxExpArray[51] = NewBig("0x0cb2ff529eb71e4fffffffffffffffffff")
	maxExpArray[52] = NewBig("0x0c2d415c3db974afffffffffffffffffff")
	maxExpArray[53] = NewBig("0x0bad03e7d883f69bffffffffffffffffff")
	maxExpArray[54] = NewBig("0x0b320d03b2c343d5ffffffffffffffffff")
	maxExpArray[55] = NewBig("0x0abc25204e02828dffffffffffffffffff")
	maxExpArray[56] = NewBig("0x0a4b16f74ee4bb207fffffffffffffffff")
	maxExpArray[57] = NewBig("0x09deaf736ac1f569ffffffffffffffffff")
	maxExpArray[58] = NewBig("0x0976bd9952c7aa957fffffffffffffffff")
	maxExpArray[59] = NewBig("0x09131271922eaa606fffffffffffffffff")
	maxExpArray[60] = NewBig("0x08b380f3558668c46fffffffffffffffff")
	maxExpArray[61] = NewBig("0x0857ddf0117efa215bffffffffffffffff")
	maxExpArray[62] = NewBig("0x07ffffffffffffffffffffffffffffffff")
	maxExpArray[63] = NewBig("0x07abbf6f6abb9d087fffffffffffffffff")
	maxExpArray[64] = NewBig("0x075af62cbac95f7dfa7fffffffffffffff")
	maxExpArray[65] = NewBig("0x070d7fb7452e187ac13fffffffffffffff")
	maxExpArray[66] = NewBig("0x06c3390ecc8af379295fffffffffffffff")
	maxExpArray[67] = NewBig("0x067c00a3b07ffc01fd6fffffffffffffff")
	maxExpArray[68] = NewBig("0x0637b647c39cbb9d3d27ffffffffffffff")
	maxExpArray[69] = NewBig("0x05f63b1fc104dbd39587ffffffffffffff")
	maxExpArray[70] = NewBig("0x05b771955b36e12f7235ffffffffffffff")
	maxExpArray[71] = NewBig("0x057b3d49dda84556d6f6ffffffffffffff")
	maxExpArray[72] = NewBig("0x054183095b2c8ececf30ffffffffffffff")
	maxExpArray[73] = NewBig("0x050a28be635ca2b888f77fffffffffffff")
	maxExpArray[74] = NewBig("0x04d5156639708c9db33c3fffffffffffff")
	maxExpArray[75] = NewBig("0x04a23105873875bd52dfdfffffffffffff")
	maxExpArray[76] = NewBig("0x0471649d87199aa990756fffffffffffff")
	maxExpArray[77] = NewBig("0x04429a21a029d4c1457cfbffffffffffff")
	maxExpArray[78] = NewBig("0x0415bc6d6fb7dd71af2cb3ffffffffffff")
	maxExpArray[79] = NewBig("0x03eab73b3bbfe282243ce1ffffffffffff")
	maxExpArray[80] = NewBig("0x03c1771ac9fb6b4c18e229ffffffffffff")
	maxExpArray[81] = NewBig("0x0399e96897690418f785257fffffffffff")
	maxExpArray[82] = NewBig("0x0373fc456c53bb779bf0ea9fffffffffff")
	maxExpArray[83] = NewBig("0x034f9e8e490c48e67e6ab8bfffffffffff")
	maxExpArray[84] = NewBig("0x032cbfd4a7adc790560b3337ffffffffff")
	maxExpArray[85] = NewBig("0x030b50570f6e5d2acca94613ffffffffff")
	maxExpArray[86] = NewBig("0x02eb40f9f620fda6b56c2861ffffffffff")
	maxExpArray[87] = NewBig("0x02cc8340ecb0d0f520a6af58ffffffffff")
	maxExpArray[88] = NewBig("0x02af09481380a0a35cf1ba02ffffffffff")
	maxExpArray[89] = NewBig("0x0292c5bdd3b92ec810287b1b3fffffffff")
	maxExpArray[90] = NewBig("0x0277abdcdab07d5a77ac6d6b9fffffffff")
	maxExpArray[91] = NewBig("0x025daf6654b1eaa55fd64df5efffffffff")
	maxExpArray[92] = NewBig("0x0244c49c648baa98192dce88b7ffffffff")
	maxExpArray[93] = NewBig("0x022ce03cd5619a311b2471268bffffffff")
	maxExpArray[94] = NewBig("0x0215f77c045fbe885654a44a0fffffffff")
	maxExpArray[95] = NewBig("0x01ffffffffffffffffffffffffffffffff")
	maxExpArray[96] = NewBig("0x01eaefdbdaaee7421fc4d3ede5ffffffff")
	maxExpArray[97] = NewBig("0x01d6bd8b2eb257df7e8ca57b09bfffffff")
	maxExpArray[98] = NewBig("0x01c35fedd14b861eb0443f7f133fffffff")
	maxExpArray[99] = NewBig("0x01b0ce43b322bcde4a56e8ada5afffffff")
	maxExpArray[100] = NewBig("0x019f0028ec1fff007f5a195a39dfffffff")
	maxExpArray[101] = NewBig("0x018ded91f0e72ee74f49b15ba527ffffff")
	maxExpArray[102] = NewBig("0x017d8ec7f04136f4e5615fd41a63ffffff")
	maxExpArray[103] = NewBig("0x016ddc6556cdb84bdc8d12d22e6fffffff")
	maxExpArray[104] = NewBig("0x015ecf52776a1155b5bd8395814f7fffff")
	maxExpArray[105] = NewBig("0x015060c256cb23b3b3cc3754cf40ffffff")
	maxExpArray[106] = NewBig("0x01428a2f98d728ae223ddab715be3fffff")
	maxExpArray[107] = NewBig("0x013545598e5c23276ccf0ede68034fffff")
	maxExpArray[108] = NewBig("0x01288c4161ce1d6f54b7f61081194fffff")
	maxExpArray[109] = NewBig("0x011c592761c666aa641d5a01a40f17ffff")
	maxExpArray[110] = NewBig("0x0110a688680a7530515f3e6e6cfdcdffff")
	maxExpArray[111] = NewBig("0x01056f1b5bedf75c6bcb2ce8aed428ffff")
	maxExpArray[112] = NewBig("0x00faadceceeff8a0890f3875f008277fff")
	maxExpArray[113] = NewBig("0x00f05dc6b27edad306388a600f6ba0bfff")
	maxExpArray[114] = NewBig("0x00e67a5a25da41063de1495d5b18cdbfff")
	maxExpArray[115] = NewBig("0x00dcff115b14eedde6fc3aa5353f2e4fff")
	maxExpArray[116] = NewBig("0x00d3e7a3924312399f9aae2e0f868f8fff")
	maxExpArray[117] = NewBig("0x00cb2ff529eb71e41582cccd5a1ee26fff")
	maxExpArray[118] = NewBig("0x00c2d415c3db974ab32a51840c0b67edff")
	maxExpArray[119] = NewBig("0x00bad03e7d883f69ad5b0a186184e06bff")
	maxExpArray[120] = NewBig("0x00b320d03b2c343d4829abd6075f0cc5ff")
	maxExpArray[121] = NewBig("0x00abc25204e02828d73c6e80bcdb1a95bf")
	maxExpArray[122] = NewBig("0x00a4b16f74ee4bb2040a1ec6c15fbbf2df")
	maxExpArray[123] = NewBig("0x009deaf736ac1f569deb1b5ae3f36c130f")
	maxExpArray[124] = NewBig("0x00976bd9952c7aa957f5937d790ef65037")
	maxExpArray[125] = NewBig("0x009131271922eaa6064b73a22d0bd4f2bf")
	maxExpArray[126] = NewBig("0x008b380f3558668c46c91c49a2f8e967b9")
	maxExpArray[127] = NewBig("0x00857ddf0117efa215952912839f6473e6")
}

func Power(
	_baseN *big.Int,
	_baseD *big.Int,
	_expN uint,
	_expD uint,
) (res *big.Int, precision uint, err error) {
	var baseLog *big.Int
	var base = new(big.Int).Div(new(big.Int).Mul(_baseN, Fixed1), _baseD)
	if base.Cmp(OptLogMaxVal) < 0 {
		baseLog = optimalLog(base)
	} else {
		baseLog = generalLog(base)
	}
	var baseLogTimesExp = new(big.Int).Div(new(big.Int).Mul(baseLog, big.NewInt(int64(_expN))), big.NewInt(int64(_expD)))
	if baseLogTimesExp.Cmp(OptExpMaxVal) < 0 {
		return optimalExp(baseLogTimesExp), MaxPrecision, nil
	} else {
		precision, err = findPositionInMaxExpArray(baseLogTimesExp)
		if err != nil {
			return nil, precision, err
		}
		return generalExp(new(big.Int).Rsh(baseLogTimesExp, MaxPrecision-precision), precision), precision, nil
	}
}

/*
   function power(
       uint256 _baseN,
       uint256 _baseD,
       uint32 _expN,
       uint32 _expD
   ) internal view returns (uint256, uint8) {
       require(_baseN < MAX_NUM);

       uint256 baseLog;
       uint256 base = (_baseN * FIXED_1) / _baseD;
       if (base < OPT_LOG_MAX_VAL) {
           baseLog = optimalLog(base);
       } else {
           baseLog = generalLog(base);
       }

       uint256 baseLogTimesExp = (baseLog * _expN) / _expD;
       if (baseLogTimesExp < OPT_EXP_MAX_VAL) {
           return (optimalExp(baseLogTimesExp), MAX_PRECISION);
       } else {
           uint8 precision = findPositionInMaxExpArray(baseLogTimesExp);
           return (generalExp(baseLogTimesExp >> (MAX_PRECISION - precision), precision), precision);
       }
   }
*/

func optimalLog(x *big.Int) (res *big.Int) {
	res = big.NewInt(0)

	if x.Cmp(NewBig("0xd3094c70f034de4b96ff7d5b6f99fcd8")) >= 0 {
		res.Add(res, NewBig("0x40000000000000000000000000000000"))
		x = new(big.Int).Div(new(big.Int).Mul(x, Fixed1), NewBig("0xd3094c70f034de4b96ff7d5b6f99fcd8"))
	}
	// add 1 / 2^1
	if x.Cmp(NewBig("0xa45af1e1f40c333b3de1db4dd55f29a7")) >= 0 {
		res.Add(res, NewBig("0x20000000000000000000000000000000"))
		x = new(big.Int).Div(new(big.Int).Mul(x, Fixed1), NewBig("0xa45af1e1f40c333b3de1db4dd55f29a7"))
	}
	// add 1 / 2^2
	if x.Cmp(NewBig("0x910b022db7ae67ce76b441c27035c6a1")) >= 0 {
		res.Add(res, NewBig("0x10000000000000000000000000000000"))
		x = new(big.Int).Div(new(big.Int).Mul(x, Fixed1), NewBig("0x910b022db7ae67ce76b441c27035c6a1"))
	}
	// add 1 / 2^3
	if x.Cmp(NewBig("0x88415abbe9a76bead8d00cf112e4d4a8")) >= 0 {
		res.Add(res, NewBig("0x08000000000000000000000000000000"))
		x = new(big.Int).Div(new(big.Int).Mul(x, Fixed1), NewBig("0x88415abbe9a76bead8d00cf112e4d4a8"))
	}
	// add 1 / 2^4
	if x.Cmp(NewBig("0x84102b00893f64c705e841d5d4064bd3")) >= 0 {
		res.Add(res, NewBig("0x04000000000000000000000000000000"))
		x = new(big.Int).Div(new(big.Int).Mul(x, Fixed1), NewBig("0x84102b00893f64c705e841d5d4064bd3"))
	}
	// add 1 / 2^5
	if x.Cmp(NewBig("0x8204055aaef1c8bd5c3259f4822735a2")) >= 0 {
		res.Add(res, NewBig("0x02000000000000000000000000000000"))
		x = new(big.Int).Div(new(big.Int).Mul(x, Fixed1), NewBig("0x8204055aaef1c8bd5c3259f4822735a2"))
	}
	// add 1 / 2^6
	if x.Cmp(NewBig("0x810100ab00222d861931c15e39b44e99")) >= 0 {
		res.Add(res, NewBig("0x01000000000000000000000000000000"))
		x = new(big.Int).Div(new(big.Int).Mul(x, Fixed1), NewBig("0x810100ab00222d861931c15e39b44e99"))
	}
	// add 1 / 2^7
	if x.Cmp(NewBig("0x808040155aabbbe9451521693554f733")) >= 0 {
		res.Add(res, NewBig("0x00800000000000000000000000000000"))
		x = new(big.Int).Div(new(big.Int).Mul(x, Fixed1), NewBig("0x808040155aabbbe9451521693554f733"))
	}
	// add 1 / 2^8

	var y = new(big.Int).Sub(x, Fixed1)
	var z = new(big.Int).Set(y)
	var w = new(big.Int).Div(new(big.Int).Mul(y, y), Fixed1)

	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(z, new(big.Int).Sub(NewBig("0x100000000000000000000000000000000"), y)), NewBig("0x100000000000000000000000000000000")))
	z = new(big.Int).Div(new(big.Int).Mul(z, w), Fixed1)
	// add y^01 / 01 - y^02 / 02
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(z, new(big.Int).Sub(NewBig("0x0aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), y)), NewBig("0x200000000000000000000000000000000")))
	z = new(big.Int).Div(new(big.Int).Mul(z, w), Fixed1)
	// add y^03 / 03 - y^04 / 04
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(z, new(big.Int).Sub(NewBig("0x099999999999999999999999999999999"), y)), NewBig("0x300000000000000000000000000000000")))
	z = new(big.Int).Div(new(big.Int).Mul(z, w), Fixed1)
	// add y^05 / 05 - y^06 / 06
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(z, new(big.Int).Sub(NewBig("0x092492492492492492492492492492492"), y)), NewBig("0x400000000000000000000000000000000")))
	z = new(big.Int).Div(new(big.Int).Mul(z, w), Fixed1)
	// add y^07 / 07 - y^08 / 08
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(z, new(big.Int).Sub(NewBig("0x08e38e38e38e38e38e38e38e38e38e38e"), y)), NewBig("0x500000000000000000000000000000000")))
	z = new(big.Int).Div(new(big.Int).Mul(z, w), Fixed1)
	// add y^09 / 09 - y^10 / 10
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(z, new(big.Int).Sub(NewBig("0x08ba2e8ba2e8ba2e8ba2e8ba2e8ba2e8b"), y)), NewBig("0x600000000000000000000000000000000")))
	z = new(big.Int).Div(new(big.Int).Mul(z, w), Fixed1)
	// add y^11 / 11 - y^12 / 12
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(z, new(big.Int).Sub(NewBig("0x089d89d89d89d89d89d89d89d89d89d89"), y)), NewBig("0x700000000000000000000000000000000")))
	z = new(big.Int).Div(new(big.Int).Mul(z, w), Fixed1)
	// add y^13 / 13 - y^14 / 14
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(z, new(big.Int).Sub(NewBig("0x088888888888888888888888888888888"), y)), NewBig("0x800000000000000000000000000000000")))
	// add y^15 / 15 - y^16 / 16

	return res
}

func optimalExp(x *big.Int) (res *big.Int) {
	res = big.NewInt(0)
	var y = new(big.Int).Mod(x, NewBig("0x10000000000000000000000000000000"))
	var z = new(big.Int).Set(y)
	// get the input modulo 2^(-3)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x10e1b3be415a0000")))
	// add y^02 * (20! / 02!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x05a0913f6b1e0000")))
	// add y^03 * (20! / 03!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0168244fdac78000")))
	// add y^04 * (20! / 04!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x004807432bc18000")))
	// add y^05 * (20! / 05!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x000c0135dca04000")))
	// add y^06 * (20! / 06!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0001b707b1cdc000")))
	// add y^07 * (20! / 07!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x000036e0f639b800")))
	// add y^08 * (20! / 08!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x00000618fee9f800")))
	// add y^09 * (20! / 09!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0000009c197dcc00")))
	// add y^10 * (20! / 10!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0000000e30dce400")))
	// add y^11 * (20! / 11!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x000000012ebd1300")))
	// add y^12 * (20! / 12!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0000000017499f00")))
	// add y^13 * (20! / 13!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0000000001a9d480")))
	// add y^14 * (20! / 14!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x00000000001c6380")))
	// add y^15 * (20! / 15!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x000000000001c638")))
	// add y^16 * (20! / 16!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0000000000001ab8")))
	// add y^17 * (20! / 17!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x000000000000017c")))
	// add y^18 * (20! / 18!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0000000000000014")))
	// add y^19 * (20! / 19!)
	z = new(big.Int).Div(new(big.Int).Mul(z, y), Fixed1)
	res = new(big.Int).Add(res, new(big.Int).Mul(z, NewBig("0x0000000000000001")))
	// add y^20 * (20! / 20!)
	res = new(big.Int).Add(new(big.Int).Add(new(big.Int).Div(res, NewBig("0x21c3677c82b40000")), y), Fixed1)
	// divide by 20! and then add y^1 / 1! + y^0 / 0!

	if new(big.Int).And(x, NewBig("0x010000000000000000000000000000000")).Cmp(Zero) != 0 {
		res = new(big.Int).Div(new(big.Int).Mul(res, NewBig("0x1c3d6a24ed82218787d624d3e5eba95f9")), NewBig("0x18ebef9eac820ae8682b9793ac6d1e776"))
	}
	// multiply by e^2^(-3)
	if new(big.Int).And(x, NewBig("0x020000000000000000000000000000000")).Cmp(Zero) != 0 {
		res = new(big.Int).Div(new(big.Int).Mul(res, NewBig("0x18ebef9eac820ae8682b9793ac6d1e778")), NewBig("0x1368b2fc6f9609fe7aceb46aa619baed4"))
	}
	// multiply by e^2^(-2)
	if new(big.Int).And(x, NewBig("0x040000000000000000000000000000000")).Cmp(Zero) != 0 {
		res = new(big.Int).Div(new(big.Int).Mul(res, NewBig("0x1368b2fc6f9609fe7aceb46aa619baed5")), NewBig("0x0bc5ab1b16779be3575bd8f0520a9f21f"))
	}
	// multiply by e^2^(-1)
	if new(big.Int).And(x, NewBig("0x080000000000000000000000000000000")).Cmp(Zero) != 0 {
		res = new(big.Int).Div(new(big.Int).Mul(res, NewBig("0x0bc5ab1b16779be3575bd8f0520a9f21e")), NewBig("0x0454aaa8efe072e7f6ddbab84b40a55c9"))
	}
	// multiply by e^2^(+0)
	if new(big.Int).And(x, NewBig("0x100000000000000000000000000000000")).Cmp(Zero) != 0 {
		res = new(big.Int).Div(new(big.Int).Mul(res, NewBig("0x0454aaa8efe072e7f6ddbab84b40a55c5")), NewBig("0x00960aadc109e7a3bf4578099615711ea"))
	}
	// multiply by e^2^(+1)
	if new(big.Int).And(x, NewBig("0x200000000000000000000000000000000")).Cmp(Zero) != 0 {
		res = new(big.Int).Div(new(big.Int).Mul(res, NewBig("0x00960aadc109e7a3bf4578099615711d7")), NewBig("0x0002bf84208204f5977f9a8cf01fdce3d"))
	}
	// multiply by e^2^(+2)
	if new(big.Int).And(x, NewBig("0x400000000000000000000000000000000")).Cmp(Zero) != 0 {
		res = new(big.Int).Div(new(big.Int).Mul(res, NewBig("0x0002bf84208204f5977f9a8cf01fdc307")), NewBig("0x0000003c6ab775dd0b95b4cbee7e65d11"))
	}
	return res
}

func generalLog(x *big.Int) (res *big.Int) {
	res = big.NewInt(0)
	// If x >= 2, then we compute the integer part of log2(x), which is larger than 0.
	if x.Cmp(Fixed2) >= 0 {
		var count = floorLog2(new(big.Int).Div(x, Fixed1))
		x = new(big.Int).Rsh(x, count)
		// now x < 2
		res = new(big.Int).Mul(Fixed1, big.NewInt(int64(count)))
	}
	// If x > 1, then we compute the fraction part of log2(x), which is larger than 0.
	if x.Cmp(Fixed1) > 0 {
		for i := MaxPrecision; i > 0; i -= 1 {
			x = new(big.Int).Div(new(big.Int).Mul(x, x), Fixed1)
			// now 1 < x < 4
			if x.Cmp(Fixed2) >= 0 {
				x = new(big.Int).Rsh(x, 1)
				// now 1 < x < 2
				res = new(big.Int).Add(res, new(big.Int).Lsh(One, uint(i-1)))
			}
		}
	}
	res = new(big.Int).Div(new(big.Int).Mul(res, Ln2Numerator), Ln2Denominator)
	return res
}

func floorLog2(_n *big.Int) (res uint) {
	res = 0
	if res < 256 {
		for _n.Cmp(One) > 0 {
			_n = new(big.Int).Rsh(_n, 1)
			res += 1
		}
	} else {
		for s := uint(128); s > 0; s >>= 1 {
			if _n.Cmp(new(big.Int).Lsh(One, s)) >= 0 {
				_n = new(big.Int).Rsh(_n, s)
				res = res | s
			}
		}
	}
	return res
}

func findPositionInMaxExpArray(_x *big.Int) (uint, error) {
	var lo = MinPrecision
	var hi = MaxPrecision
	for lo+1 < hi {
		var mid = (lo + hi) / 2
		if maxExpArray[mid].Cmp(_x) >= 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	if maxExpArray[hi].Cmp(_x) >= 0 {
		return hi, nil
	}
	if maxExpArray[lo].Cmp(_x) >= 0 {
		return lo, nil
	}

	return 0, errors.New("revert")
}

func generalExp(_x *big.Int, _precision uint) (res *big.Int) {
	var xi = new(big.Int).Set(_x)
	res = big.NewInt(0)

	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x3442c4e6074a82f1797f72ac0000000")))
	// add x^02 * (33! / 02!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x116b96f757c380fb287fd0e40000000")))
	// add x^03 * (33! / 03!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x045ae5bdd5f0e03eca1ff4390000000")))
	// add x^04 * (33! / 04!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00defabf91302cd95b9ffda50000000")))
	// add x^05 * (33! / 05!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x002529ca9832b22439efff9b8000000")))
	// add x^06 * (33! / 06!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00054f1cf12bd04e516b6da88000000")))
	// add x^07 * (33! / 07!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000a9e39e257a09ca2d6db51000000")))
	// add x^08 * (33! / 08!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x000012e066e7b839fa050c309000000")))
	// add x^09 * (33! / 09!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x000001e33d7d926c329a1ad1a800000")))
	// add x^10 * (33! / 10!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000002bee513bdb4a6b19b5f800000")))
	// add x^11 * (33! / 11!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00000003a9316fa79b88eccf2a00000")))
	// add x^12 * (33! / 12!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000048177ebe1fa812375200000")))
	// add x^13 * (33! / 13!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000005263fe90242dcbacf00000")))
	// add x^14 * (33! / 14!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x000000000057e22099c030d94100000")))
	// add x^15 * (33! / 15!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000000057e22099c030d9410000")))
	// add x^16 * (33! / 16!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00000000000052b6b54569976310000")))
	// add x^17 * (33! / 17!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00000000000004985f67696bf748000")))
	// add x^18 * (33! / 18!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x000000000000003dea12ea99e498000")))
	// add x^19 * (33! / 19!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00000000000000031880f2214b6e000")))
	// add x^20 * (33! / 20!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x000000000000000025bcff56eb36000")))
	// add x^21 * (33! / 21!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x000000000000000001b722e10ab1000")))
	// add x^22 * (33! / 22!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000000000000001317c70077000")))
	// add x^23 * (33! / 23!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00000000000000000000cba84aafa00")))
	// add x^24 * (33! / 24!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00000000000000000000082573a0a00")))
	// add x^25 * (33! / 25!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00000000000000000000005035ad900")))
	// add x^26 * (33! / 26!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x000000000000000000000002f881b00")))
	// add x^27 * (33! / 27!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000000000000000000001b29340")))
	// add x^28 * (33! / 28!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x00000000000000000000000000efc40")))
	// add x^29 * (33! / 29!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000000000000000000000007fe0")))
	// add x^30 * (33! / 30!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000000000000000000000000420")))
	// add x^31 * (33! / 31!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000000000000000000000000021")))
	// add x^32 * (33! / 32!)
	xi = new(big.Int).Rsh(new(big.Int).Mul(xi, _x), _precision)
	res = new(big.Int).Add(res, new(big.Int).Mul(xi, NewBig("0x0000000000000000000000000000001")))
	// add x^33 * (33! / 33!)
	res = new(big.Int).Add(new(big.Int).Add(new(big.Int).Div(res, NewBig("0x688589cc0e9505e2f2fee5580000000")), _x), new(big.Int).Lsh(One, _precision))
	// divide by 33! and then add x^1 / 1! + x^0 / 0!

	return res
}
