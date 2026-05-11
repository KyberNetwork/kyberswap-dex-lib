package uniswapv4

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// BenchmarkCalculateReservesFromTicks/monad/0x327e...4b71-16         		   34908	     30883 ns/op
// BenchmarkCalculateReservesFromTicks/ethereum/0xce93...f4fb-16         	   12676	     94247 ns/op
// BenchmarkCalculateReservesFromTicks/ethereum/0xce93...f4fb/left-16    	   12915	     94428 ns/op
// BenchmarkCalculateReservesFromTicks/ethereum/0xce93...f4fb/right-16   	   12956	     90901 ns/op
// BenchmarkEstimateReservesFromTicks/monad/0x327e...4b71-16         		  517102	      2202 ns/op
// BenchmarkEstimateReservesFromTicks/ethereum/0xce93...f4fb-16         	  319519	      3693 ns/op
// BenchmarkEstimateReservesFromTicks/ethereum/0xce93...f4fb/left-16    	  323589	      3702 ns/op
// BenchmarkEstimateReservesFromTicks/ethereum/0xce93...f4fb/right-16   	  331810	      3608 ns/op
func BenchmarkEstimateReservesFromTicks(b *testing.B) {
	type args struct {
		sqrtPriceX96 *big.Int
		ticks        []Tick
	}
	tests := []struct {
		name  string
		args  args
		want  *big.Int
		want1 *big.Int
	}{
		{
			"monad/0x327ebb1d4930262df6ad436b73805224caab4b71",
			args{
				bignumber.NewBig10("7453123888791196607554378126430981"),
				[]Tick{
					{Index: -887272, LiquidityGross: bignumber.NewBig10("765114625359041857607"),
						LiquidityNet: bignumber.NewBig10("765114625359041857607")},
					{Index: -174, LiquidityGross: bignumber.NewBig10("277704853565806526590328"),
						LiquidityNet: bignumber.NewBig10("277704853565806526590328")},
					{Index: -172, LiquidityGross: bignumber.NewBig10("98885750943734208485244"),
						LiquidityNet: bignumber.NewBig10("98885750943734208485244")},
					{Index: -154, LiquidityGross: bignumber.NewBig10("189884459431526769416431"),
						LiquidityNet: bignumber.NewBig10("-189884459431526769416431")},
					{Index: -152, LiquidityGross: bignumber.NewBig10("186706145078013965659141"),
						LiquidityNet: bignumber.NewBig10("-186706145078013965659141")},
					{Index: -100, LiquidityGross: bignumber.NewBig10("663690190241077530894"),
						LiquidityNet: bignumber.NewBig10("663690190241077530894")},
					{Index: -52, LiquidityGross: bignumber.NewBig10("168409174034058582220347"),
						LiquidityNet: bignumber.NewBig10("168409174034058582220347")},
					{Index: -38, LiquidityGross: bignumber.NewBig10("1904539471348770338275145"),
						LiquidityNet: bignumber.NewBig10("1904539471348770338275145")},
					{Index: -34, LiquidityGross: bignumber.NewBig10("9476451026852285606311"),
						LiquidityNet: bignumber.NewBig10("9476451026852285606311")},
					{Index: -30, LiquidityGross: bignumber.NewBig10("168409174034058582220347"),
						LiquidityNet: bignumber.NewBig10("-168409174034058582220347")},
					{Index: -18, LiquidityGross: bignumber.NewBig10("9909836923256278760334"),
						LiquidityNet: bignumber.NewBig10("-9909836923256278760334")},
					{Index: -16, LiquidityGross: bignumber.NewBig10("1894629634425514059514811"),
						LiquidityNet: bignumber.NewBig10("-1894629634425514059514811")},
					{Index: -12, LiquidityGross: bignumber.NewBig10("9476451026852285606311"),
						LiquidityNet: bignumber.NewBig10("-9476451026852285606311")},
					{Index: 4, LiquidityGross: bignumber.NewBig10("1870398237390118250087"),
						LiquidityNet: bignumber.NewBig10("1870398237390118250087")},
					{Index: 24, LiquidityGross: bignumber.NewBig10("1870398237390118250087"),
						LiquidityNet: bignumber.NewBig10("-1870398237390118250087")},
					{Index: 198, LiquidityGross: bignumber.NewBig10("663690190241077530894"),
						LiquidityNet: bignumber.NewBig10("-663690190241077530894")},
					{Index: 887272, LiquidityGross: bignumber.NewBig10("765114625359041857607"),
						LiquidityNet: bignumber.NewBig10("-765114625359041857607")},
				},
			},
			bignumber.NewBig10("8133317892537814"),           // 8133317892535114"),
			bignumber.NewBig10("71978274773382566248448000"), // 71978274773358695221586273"),
		},
		{
			"ethereum/0xce93ea3914c62e0008348cf39fd006e130e7c503935fb01d154b971c8663f4fb",
			args{
				bignumber.NewBig10("79197716553732212498174"),
				[]Tick{
					{Index: -276527, LiquidityGross: bignumber.NewBig10("98980320539201"),
						LiquidityNet: bignumber.NewBig10("98980320539201")},
					{Index: -276347, LiquidityGross: bignumber.NewBig10("833658155516142"),
						LiquidityNet: bignumber.NewBig10("833658155516142")},
					{Index: -276342, LiquidityGross: bignumber.NewBig10("2008532549106444469"),
						LiquidityNet: bignumber.NewBig10("2008532549106444469")},
					{Index: -276341, LiquidityGross: bignumber.NewBig10("691673467972982181"),
						LiquidityNet: bignumber.NewBig10("691673467972982181")},
					{Index: -276340, LiquidityGross: bignumber.NewBig10("1859731088173396593"),
						LiquidityNet: bignumber.NewBig10("1859731088173396593")},
					{Index: -276337, LiquidityGross: bignumber.NewBig10("7816803935579981"),
						LiquidityNet: bignumber.NewBig10("7816803935579981")},
					{Index: -276336, LiquidityGross: bignumber.NewBig10("2959517901749615"),
						LiquidityNet: bignumber.NewBig10("2959517901749615")},
					{Index: -276334, LiquidityGross: bignumber.NewBig10("199856248859500479246"),
						LiquidityNet: bignumber.NewBig10("199856248859500479246")},
					{Index: -276333, LiquidityGross: bignumber.NewBig10("1673262166553490730910"),
						LiquidityNet: bignumber.NewBig10("1673262166553490730910")},
					{Index: -276332, LiquidityGross: bignumber.NewBig10("4540933772548223631"),
						LiquidityNet: bignumber.NewBig10("4540933772548223631")},
					{Index: -276331, LiquidityGross: bignumber.NewBig10("1808804715782718853662"),
						LiquidityNet: bignumber.NewBig10("1808804715782718853662")},
					{Index: -276330, LiquidityGross: bignumber.NewBig10("1665674055704541238983"),
						LiquidityNet: bignumber.NewBig10("-1665674055704541238983")},
					{Index: -276329, LiquidityGross: bignumber.NewBig10("7588110848949491927"),
						LiquidityNet: bignumber.NewBig10("-7588110848949491927")},
					{Index: -276327, LiquidityGross: bignumber.NewBig10("802993731331559738851"),
						LiquidityNet: bignumber.NewBig10("802993731331559738851")},
					{Index: -276326, LiquidityGross: bignumber.NewBig10("525052980235936978976"),
						LiquidityNet: bignumber.NewBig10("525052980235936978976")},
					{Index: -276324, LiquidityGross: bignumber.NewBig10("209378751800081674986"),
						LiquidityNet: bignumber.NewBig10("-199380698516943547426")},
					{Index: -276322, LiquidityGross: bignumber.NewBig10("1865332746238602322054"),
						LiquidityNet: bignumber.NewBig10("811211719953889177960")},
					{Index: -276321, LiquidityGross: bignumber.NewBig10("514050212161307294996"),
						LiquidityNet: bignumber.NewBig10("-514050212161307294996")},
					{Index: -276320, LiquidityGross: bignumber.NewBig10("305463532258316544878"),
						LiquidityNet: bignumber.NewBig10("-277526315194481100380")},
					{Index: -276319, LiquidityGross: bignumber.NewBig10("1808804715782718853662"),
						LiquidityNet: bignumber.NewBig10("-1808804715782718853662")},
					{Index: -276318, LiquidityGross: bignumber.NewBig10("188403224252742553255"),
						LiquidityNet: bignumber.NewBig10("188403224252742553255")},
					{Index: -276317, LiquidityGross: bignumber.NewBig10("13992882772818615111"),
						LiquidityNet: bignumber.NewBig10("-13992882772818615111")},
					{Index: -276316, LiquidityGross: bignumber.NewBig10("1526678416866890052877"),
						LiquidityNet: bignumber.NewBig10("-1526678416866890052877")},
					{Index: -276314, LiquidityGross: bignumber.NewBig10("4997985752181552452"),
						LiquidityNet: bignumber.NewBig10("-4997985752181552452")},
					{Index: -276313, LiquidityGross: bignumber.NewBig10("2040925958290118"),
						LiquidityNet: bignumber.NewBig10("-2040925958290118")},
					{Index: -276306, LiquidityGross: bignumber.NewBig10("999642686851398"),
						LiquidityNet: bignumber.NewBig10("-999642686851398")},
					{Index: -276299, LiquidityGross: bignumber.NewBig10("833658155516142"),
						LiquidityNet: bignumber.NewBig10("-833658155516142")},
					{Index: -276126, LiquidityGross: bignumber.NewBig10("98980320539201"),
						LiquidityNet: bignumber.NewBig10("-98980320539201")},
				},
			},
			bignumber.NewBig10("2095552366121604704370688"), // 2095552357818872570419923"),
			bignumber.NewBig10("132570769007"),              // 132570777309"),
		},
		{
			"ethereum/0xce93ea3914c62e0008348cf39fd006e130e7c503935fb01d154b971c8663f4fb/left",
			args{
				bignumber.NewBig10("78428000000000000000000"),
				[]Tick{
					{Index: -276527, LiquidityGross: bignumber.NewBig10("98980320539201"),
						LiquidityNet: bignumber.NewBig10("98980320539201")},
					{Index: -276347, LiquidityGross: bignumber.NewBig10("833658155516142"),
						LiquidityNet: bignumber.NewBig10("833658155516142")},
					{Index: -276342, LiquidityGross: bignumber.NewBig10("2008532549106444469"),
						LiquidityNet: bignumber.NewBig10("2008532549106444469")},
					{Index: -276341, LiquidityGross: bignumber.NewBig10("691673467972982181"),
						LiquidityNet: bignumber.NewBig10("691673467972982181")},
					{Index: -276340, LiquidityGross: bignumber.NewBig10("1859731088173396593"),
						LiquidityNet: bignumber.NewBig10("1859731088173396593")},
					{Index: -276337, LiquidityGross: bignumber.NewBig10("7816803935579981"),
						LiquidityNet: bignumber.NewBig10("7816803935579981")},
					{Index: -276336, LiquidityGross: bignumber.NewBig10("2959517901749615"),
						LiquidityNet: bignumber.NewBig10("2959517901749615")},
					{Index: -276334, LiquidityGross: bignumber.NewBig10("199856248859500479246"),
						LiquidityNet: bignumber.NewBig10("199856248859500479246")},
					{Index: -276333, LiquidityGross: bignumber.NewBig10("1673262166553490730910"),
						LiquidityNet: bignumber.NewBig10("1673262166553490730910")},
					{Index: -276332, LiquidityGross: bignumber.NewBig10("4540933772548223631"),
						LiquidityNet: bignumber.NewBig10("4540933772548223631")},
					{Index: -276331, LiquidityGross: bignumber.NewBig10("1808804715782718853662"),
						LiquidityNet: bignumber.NewBig10("1808804715782718853662")},
					{Index: -276330, LiquidityGross: bignumber.NewBig10("1665674055704541238983"),
						LiquidityNet: bignumber.NewBig10("-1665674055704541238983")},
					{Index: -276329, LiquidityGross: bignumber.NewBig10("7588110848949491927"),
						LiquidityNet: bignumber.NewBig10("-7588110848949491927")},
					{Index: -276327, LiquidityGross: bignumber.NewBig10("802993731331559738851"),
						LiquidityNet: bignumber.NewBig10("802993731331559738851")},
					{Index: -276326, LiquidityGross: bignumber.NewBig10("525052980235936978976"),
						LiquidityNet: bignumber.NewBig10("525052980235936978976")},
					{Index: -276324, LiquidityGross: bignumber.NewBig10("209378751800081674986"),
						LiquidityNet: bignumber.NewBig10("-199380698516943547426")},
					{Index: -276322, LiquidityGross: bignumber.NewBig10("1865332746238602322054"),
						LiquidityNet: bignumber.NewBig10("811211719953889177960")},
					{Index: -276321, LiquidityGross: bignumber.NewBig10("514050212161307294996"),
						LiquidityNet: bignumber.NewBig10("-514050212161307294996")},
					{Index: -276320, LiquidityGross: bignumber.NewBig10("305463532258316544878"),
						LiquidityNet: bignumber.NewBig10("-277526315194481100380")},
					{Index: -276319, LiquidityGross: bignumber.NewBig10("1808804715782718853662"),
						LiquidityNet: bignumber.NewBig10("-1808804715782718853662")},
					{Index: -276318, LiquidityGross: bignumber.NewBig10("188403224252742553255"),
						LiquidityNet: bignumber.NewBig10("188403224252742553255")},
					{Index: -276317, LiquidityGross: bignumber.NewBig10("13992882772818615111"),
						LiquidityNet: bignumber.NewBig10("-13992882772818615111")},
					{Index: -276316, LiquidityGross: bignumber.NewBig10("1526678416866890052877"),
						LiquidityNet: bignumber.NewBig10("-1526678416866890052877")},
					{Index: -276314, LiquidityGross: bignumber.NewBig10("4997985752181552452"),
						LiquidityNet: bignumber.NewBig10("-4997985752181552452")},
					{Index: -276313, LiquidityGross: bignumber.NewBig10("2040925958290118"),
						LiquidityNet: bignumber.NewBig10("-2040925958290118")},
					{Index: -276306, LiquidityGross: bignumber.NewBig10("999642686851398"),
						LiquidityNet: bignumber.NewBig10("-999642686851398")},
					{Index: -276299, LiquidityGross: bignumber.NewBig10("833658155516142"),
						LiquidityNet: bignumber.NewBig10("-833658155516142")},
					{Index: -276126, LiquidityGross: bignumber.NewBig10("98980320539201"),
						LiquidityNet: bignumber.NewBig10("-98980320539201")},
				},
			},
			bignumber.NewBig10("2228235627014885761613824"), // 2228235627026004796995655"),
			bignumber.NewBig10("0"),
		},
		{
			"ethereum/0xce93ea3914c62e0008348cf39fd006e130e7c503935fb01d154b971c8663f4fb/right",
			args{
				bignumber.NewBig10("80020483637005745701403"),
				[]Tick{
					{Index: -276527, LiquidityGross: bignumber.NewBig10("98980320539201"),
						LiquidityNet: bignumber.NewBig10("98980320539201")},
					{Index: -276347, LiquidityGross: bignumber.NewBig10("833658155516142"),
						LiquidityNet: bignumber.NewBig10("833658155516142")},
					{Index: -276342, LiquidityGross: bignumber.NewBig10("2008532549106444469"),
						LiquidityNet: bignumber.NewBig10("2008532549106444469")},
					{Index: -276341, LiquidityGross: bignumber.NewBig10("691673467972982181"),
						LiquidityNet: bignumber.NewBig10("691673467972982181")},
					{Index: -276340, LiquidityGross: bignumber.NewBig10("1859731088173396593"),
						LiquidityNet: bignumber.NewBig10("1859731088173396593")},
					{Index: -276337, LiquidityGross: bignumber.NewBig10("7816803935579981"),
						LiquidityNet: bignumber.NewBig10("7816803935579981")},
					{Index: -276336, LiquidityGross: bignumber.NewBig10("2959517901749615"),
						LiquidityNet: bignumber.NewBig10("2959517901749615")},
					{Index: -276334, LiquidityGross: bignumber.NewBig10("199856248859500479246"),
						LiquidityNet: bignumber.NewBig10("199856248859500479246")},
					{Index: -276333, LiquidityGross: bignumber.NewBig10("1673262166553490730910"),
						LiquidityNet: bignumber.NewBig10("1673262166553490730910")},
					{Index: -276332, LiquidityGross: bignumber.NewBig10("4540933772548223631"),
						LiquidityNet: bignumber.NewBig10("4540933772548223631")},
					{Index: -276331, LiquidityGross: bignumber.NewBig10("1808804715782718853662"),
						LiquidityNet: bignumber.NewBig10("1808804715782718853662")},
					{Index: -276330, LiquidityGross: bignumber.NewBig10("1665674055704541238983"),
						LiquidityNet: bignumber.NewBig10("-1665674055704541238983")},
					{Index: -276329, LiquidityGross: bignumber.NewBig10("7588110848949491927"),
						LiquidityNet: bignumber.NewBig10("-7588110848949491927")},
					{Index: -276327, LiquidityGross: bignumber.NewBig10("802993731331559738851"),
						LiquidityNet: bignumber.NewBig10("802993731331559738851")},
					{Index: -276326, LiquidityGross: bignumber.NewBig10("525052980235936978976"),
						LiquidityNet: bignumber.NewBig10("525052980235936978976")},
					{Index: -276324, LiquidityGross: bignumber.NewBig10("209378751800081674986"),
						LiquidityNet: bignumber.NewBig10("-199380698516943547426")},
					{Index: -276322, LiquidityGross: bignumber.NewBig10("1865332746238602322054"),
						LiquidityNet: bignumber.NewBig10("811211719953889177960")},
					{Index: -276321, LiquidityGross: bignumber.NewBig10("514050212161307294996"),
						LiquidityNet: bignumber.NewBig10("-514050212161307294996")},
					{Index: -276320, LiquidityGross: bignumber.NewBig10("305463532258316544878"),
						LiquidityNet: bignumber.NewBig10("-277526315194481100380")},
					{Index: -276319, LiquidityGross: bignumber.NewBig10("1808804715782718853662"),
						LiquidityNet: bignumber.NewBig10("-1808804715782718853662")},
					{Index: -276318, LiquidityGross: bignumber.NewBig10("188403224252742553255"),
						LiquidityNet: bignumber.NewBig10("188403224252742553255")},
					{Index: -276317, LiquidityGross: bignumber.NewBig10("13992882772818615111"),
						LiquidityNet: bignumber.NewBig10("-13992882772818615111")},
					{Index: -276316, LiquidityGross: bignumber.NewBig10("1526678416866890052877"),
						LiquidityNet: bignumber.NewBig10("-1526678416866890052877")},
					{Index: -276314, LiquidityGross: bignumber.NewBig10("4997985752181552452"),
						LiquidityNet: bignumber.NewBig10("-4997985752181552452")},
					{Index: -276313, LiquidityGross: bignumber.NewBig10("2040925958290118"),
						LiquidityNet: bignumber.NewBig10("-2040925958290118")},
					{Index: -276306, LiquidityGross: bignumber.NewBig10("999642686851398"),
						LiquidityNet: bignumber.NewBig10("-999642686851398")},
					{Index: -276299, LiquidityGross: bignumber.NewBig10("833658155516142"),
						LiquidityNet: bignumber.NewBig10("-833658155516142")},
					{Index: -276126, LiquidityGross: bignumber.NewBig10("98980320539201"),
						LiquidityNet: bignumber.NewBig10("-98980320539201")},
				},
			},
			bignumber.NewBig10("0"),
			bignumber.NewBig10("2228112054269"), // 2228112054248"),
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			got, got1 := EstimateReservesFromTicks(tt.args.sqrtPriceX96, tt.args.ticks)
			assert.Equal(b, tt.want, got)
			assert.Equal(b, tt.want1, got1)
			for range b.N - 1 {
				_, _ = EstimateReservesFromTicks(tt.args.sqrtPriceX96, tt.args.ticks)
			}
		})
	}
}

func TestEstimateReservesFromTicksU256(t *testing.T) {
	type args struct {
		sqrtPriceX96 *uint256.Int
		ticks        []v3.TickU256
	}
	tests := []struct {
		name     string
		args     args
		wantAmt0 *uint256.Int
		wantAmt1 *uint256.Int
	}{
		{
			"alphix",
			args{
				big256.New("3580966902810831567784431"),
				[]v3.TickU256{
					{Index: -201420, LiquidityGross: big256.New("8104963630759"),
						LiquidityNet: big256.SNew("8104963630759")},
					{Index: -200280, LiquidityGross: big256.New("208606306005373"),
						LiquidityNet: big256.SNew("208606306005373")},
					{Index: -199920, LiquidityGross: big256.New("208606306005373"),
						LiquidityNet: big256.SNew("-208606306005373")},
					{Index: -199260, LiquidityGross: big256.New("8104963630759"),
						LiquidityNet: big256.SNew("-8104963630759")},
				},
			},
			big256.New("48508975244978208"),
			big256.New("108309717"),
		},
		{
			"bsc/pancake-v4",
			args{
				big256.New("79237179352329355941828490223"),
				[]v3.TickU256{
					{Index: -2623, LiquidityGross: big256.New("742805861479521102"), LiquidityNet: big256.SNew("742805861479521102")},
					{Index: -125, LiquidityGross: big256.New("771584970321333610037"), LiquidityNet: big256.SNew("771584970321333610037")},
					{Index: -19, LiquidityGross: big256.New("375032944358810947165"), LiquidityNet: big256.SNew("375032944358810947165")},
					{Index: -17, LiquidityGross: big256.New("24978251690715226032651"), LiquidityNet: big256.SNew("24978251690715226032651")},
					{Index: -16, LiquidityGross: big256.New("3533516441703208035294"), LiquidityNet: big256.SNew("3533516441703208035294")},
					{Index: -15, LiquidityGross: big256.New("3517852157951758611390"), LiquidityNet: big256.SNew("3517852157951758611390")},
					{Index: -14, LiquidityGross: big256.New("2019305140245811772763"), LiquidityNet: big256.SNew("2019305140245811772763")},
					{Index: -13, LiquidityGross: big256.New("2000926026437466748174"), LiquidityNet: big256.SNew("2000926026437466748174")},
					{Index: -12, LiquidityGross: big256.New("5787554390341317058440"), LiquidityNet: big256.SNew("5787554390341317058438")},
					{Index: -11, LiquidityGross: big256.New("13957090781843067809029"), LiquidityNet: big256.SNew("13957090781843067809027")},
					{Index: -10, LiquidityGross: big256.New("2083597657534767062318"), LiquidityNet: big256.SNew("2083597657534767062316")},
					{Index: -9, LiquidityGross: big256.New("39676979711830492622700185"), LiquidityNet: big256.SNew("39626273142560344548740549")},
					{Index: -8, LiquidityGross: big256.New("2132151964836209762945"), LiquidityNet: big256.SNew("2132151964836209762941")},
					{Index: -7, LiquidityGross: big256.New("6000246638243649451492"), LiquidityNet: big256.SNew("6000246638243649451488")},
					{Index: -6, LiquidityGross: big256.New("62421504804543879016162872"), LiquidityNet: big256.SNew("62414437771660472600092284")},
					{Index: -5, LiquidityGross: big256.New("129898863127017567031422975"), LiquidityNet: big256.SNew("129885995530737819772279511")},
					{Index: -4, LiquidityGross: big256.New("12211057763283757700132113"), LiquidityNet: big256.SNew("12201789589148560655114017")},
					{Index: -3, LiquidityGross: big256.New("167640575545865715930993346"), LiquidityNet: big256.SNew("167636573693812840997496998")},
					{Index: -2, LiquidityGross: big256.New("947329834238157980135405"), LiquidityNet: big256.SNew("-942662186739577000624639")},
					{Index: -1, LiquidityGross: big256.New("179529325987065345612512"), LiquidityNet: big256.SNew("150815326652870946355214")},
					{Index: 0, LiquidityGross: big256.New("75073240733186868800511008"), LiquidityNet: big256.SNew("75069209791393607104894022")},
					{Index: 1, LiquidityGross: big256.New("83241819334822484100192596"), LiquidityNet: big256.SNew("-83192567035131723978469846")},
					{Index: 2, LiquidityGross: big256.New("162588558528412634500134760"), LiquidityNet: big256.SNew("-162588558528412634500134758")},
					{Index: 3, LiquidityGross: big256.New("152567777055602292189505423"), LiquidityNet: big256.SNew("-152567777055602292189505421")},
					{Index: 4, LiquidityGross: big256.New("77538285429336504129293072"), LiquidityNet: big256.SNew("-77538285429336504129293072")},
					{Index: 5, LiquidityGross: big256.New("31818212823423986151521"), LiquidityNet: big256.SNew("-31372341792552736420491")},
					{Index: 6, LiquidityGross: big256.New("9993790858398056443329559"), LiquidityNet: big256.SNew("-9993790858398056443329559")},
					{Index: 7, LiquidityGross: big256.New("24337590319140954542360"), LiquidityNet: big256.SNew("-24337590319140954542360")},
					{Index: 8, LiquidityGross: big256.New("169638302034094845502185"), LiquidityNet: big256.SNew("-169638302034094845502185")},
					{Index: 10, LiquidityGross: big256.New("268167229439953474248"), LiquidityNet: big256.SNew("-268167229439953474248")},
					{Index: 13, LiquidityGross: big256.New("1999941289275562102148"), LiquidityNet: big256.SNew("-1999941289275562102148")},
					{Index: 15, LiquidityGross: big256.New("222935515435624865515"), LiquidityNet: big256.SNew("-222935515435624865515")},
					{Index: 183, LiquidityGross: big256.New("771584970321333610037"), LiquidityNet: big256.SNew("-771584970321333610037")},
					{Index: 3568, LiquidityGross: big256.New("742805861479521102"), LiquidityNet: big256.SNew("-742805861479521102")},
				},
			},
			big256.New("14129812181867435130880"),
			big256.New("144328535395462486687744"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAmt0, gotAmt1 := EstimateReservesFromTicksU256(tt.args.sqrtPriceX96, tt.args.ticks)
			assert.Equalf(t, tt.wantAmt0, gotAmt0, "EstimateReservesFromTicksU256(%v, %v)", tt.args.sqrtPriceX96,
				tt.args.ticks)
			assert.Equalf(t, tt.wantAmt1, gotAmt1, "EstimateReservesFromTicksU256(%v, %v)", tt.args.sqrtPriceX96,
				tt.args.ticks)
		})
	}
}
