package liquiditybookv21

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

var entityStr = "{\"address\":\"0x576b3179e58de0c91cb15aeaadeb4ecfee3620af\",\"amplifiedTvl\":187534265519277700000,\"exchange\":\"e3\",\"type\":\"liquiditybook-v21\",\"timestamp\":1736346251,\"reserves\":[\"49524428824877076\",\"4885626\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"weight\":50,\"swappable\":true},{\"address\":\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\",\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"rpcBlockTimestamp\\\":1736346251,\\\"subgraphBlockTimestamp\\\":1715847100,\\\"staticFeeParams\\\":{\\\"baseFactor\\\":8000,\\\"filterPeriod\\\":30,\\\"decayPeriod\\\":600,\\\"reductionFactor\\\":5000,\\\"variableFeeControl\\\":120000,\\\"protocolShare\\\":2023,\\\"maxVolatilityAccumulator\\\":300000},\\\"variableFeeParams\\\":{\\\"volatilityAccumulator\\\":300000,\\\"volatilityReference\\\":0,\\\"idReference\\\":8349252,\\\"timeOfLastUpdate\\\":1736346251},\\\"activeBinId\\\":8349755,\\\"binStep\\\":5,\\\"bins\\\":[{\\\"id\\\":8349152,\\\"reserveX\\\":0,\\\"reserveY\\\":39389},{\\\"id\\\":8349153,\\\"reserveX\\\":0,\\\"reserveY\\\":39454},{\\\"id\\\":8349154,\\\"reserveX\\\":0,\\\"reserveY\\\":39482},{\\\"id\\\":8349155,\\\"reserveX\\\":0,\\\"reserveY\\\":39481},{\\\"id\\\":8349156,\\\"reserveX\\\":0,\\\"reserveY\\\":39500},{\\\"id\\\":8349157,\\\"reserveX\\\":0,\\\"reserveY\\\":39551},{\\\"id\\\":8349158,\\\"reserveX\\\":0,\\\"reserveY\\\":39545},{\\\"id\\\":8349159,\\\"reserveX\\\":0,\\\"reserveY\\\":39542},{\\\"id\\\":8349160,\\\"reserveX\\\":0,\\\"reserveY\\\":39543},{\\\"id\\\":8349161,\\\"reserveX\\\":0,\\\"reserveY\\\":39588},{\\\"id\\\":8349162,\\\"reserveX\\\":0,\\\"reserveY\\\":39614},{\\\"id\\\":8349163,\\\"reserveX\\\":0,\\\"reserveY\\\":39614},{\\\"id\\\":8349164,\\\"reserveX\\\":0,\\\"reserveY\\\":39679},{\\\"id\\\":8349165,\\\"reserveX\\\":0,\\\"reserveY\\\":39724},{\\\"id\\\":8349166,\\\"reserveX\\\":0,\\\"reserveY\\\":39736},{\\\"id\\\":8349167,\\\"reserveX\\\":0,\\\"reserveY\\\":39713},{\\\"id\\\":8349168,\\\"reserveX\\\":0,\\\"reserveY\\\":39674},{\\\"id\\\":8349169,\\\"reserveX\\\":0,\\\"reserveY\\\":39666},{\\\"id\\\":8349170,\\\"reserveX\\\":0,\\\"reserveY\\\":39668},{\\\"id\\\":8349171,\\\"reserveX\\\":0,\\\"reserveY\\\":39712},{\\\"id\\\":8349172,\\\"reserveX\\\":0,\\\"reserveY\\\":39756},{\\\"id\\\":8349173,\\\"reserveX\\\":0,\\\"reserveY\\\":39752},{\\\"id\\\":8349174,\\\"reserveX\\\":0,\\\"reserveY\\\":39750},{\\\"id\\\":8349175,\\\"reserveX\\\":0,\\\"reserveY\\\":39754},{\\\"id\\\":8349176,\\\"reserveX\\\":0,\\\"reserveY\\\":39794},{\\\"id\\\":8349177,\\\"reserveX\\\":0,\\\"reserveY\\\":39714},{\\\"id\\\":8349178,\\\"reserveX\\\":0,\\\"reserveY\\\":39768},{\\\"id\\\":8349179,\\\"reserveX\\\":0,\\\"reserveY\\\":39845},{\\\"id\\\":8349180,\\\"reserveX\\\":0,\\\"reserveY\\\":39750},{\\\"id\\\":8349181,\\\"reserveX\\\":0,\\\"reserveY\\\":39769},{\\\"id\\\":8349182,\\\"reserveX\\\":0,\\\"reserveY\\\":39764},{\\\"id\\\":8349183,\\\"reserveX\\\":0,\\\"reserveY\\\":39766},{\\\"id\\\":8349184,\\\"reserveX\\\":0,\\\"reserveY\\\":39754},{\\\"id\\\":8349185,\\\"reserveX\\\":0,\\\"reserveY\\\":39725},{\\\"id\\\":8349186,\\\"reserveX\\\":0,\\\"reserveY\\\":39728},{\\\"id\\\":8349187,\\\"reserveX\\\":0,\\\"reserveY\\\":39734},{\\\"id\\\":8349188,\\\"reserveX\\\":0,\\\"reserveY\\\":39768},{\\\"id\\\":8349189,\\\"reserveX\\\":0,\\\"reserveY\\\":39766},{\\\"id\\\":8349190,\\\"reserveX\\\":0,\\\"reserveY\\\":39763},{\\\"id\\\":8349191,\\\"reserveX\\\":0,\\\"reserveY\\\":39747},{\\\"id\\\":8349192,\\\"reserveX\\\":0,\\\"reserveY\\\":39746},{\\\"id\\\":8349193,\\\"reserveX\\\":0,\\\"reserveY\\\":39753},{\\\"id\\\":8349194,\\\"reserveX\\\":0,\\\"reserveY\\\":39733},{\\\"id\\\":8349195,\\\"reserveX\\\":0,\\\"reserveY\\\":39735},{\\\"id\\\":8349196,\\\"reserveX\\\":0,\\\"reserveY\\\":39710},{\\\"id\\\":8349197,\\\"reserveX\\\":0,\\\"reserveY\\\":39608},{\\\"id\\\":8349198,\\\"reserveX\\\":0,\\\"reserveY\\\":39598},{\\\"id\\\":8349199,\\\"reserveX\\\":0,\\\"reserveY\\\":39597},{\\\"id\\\":8349200,\\\"reserveX\\\":0,\\\"reserveY\\\":39598},{\\\"id\\\":8349201,\\\"reserveX\\\":0,\\\"reserveY\\\":39599},{\\\"id\\\":8349202,\\\"reserveX\\\":0,\\\"reserveY\\\":94948},{\\\"id\\\":8349203,\\\"reserveX\\\":0,\\\"reserveY\\\":55345},{\\\"id\\\":8349204,\\\"reserveX\\\":0,\\\"reserveY\\\":55376},{\\\"id\\\":8349205,\\\"reserveX\\\":0,\\\"reserveY\\\":55396},{\\\"id\\\":8349206,\\\"reserveX\\\":0,\\\"reserveY\\\":55410},{\\\"id\\\":8349207,\\\"reserveX\\\":0,\\\"reserveY\\\":55447},{\\\"id\\\":8349208,\\\"reserveX\\\":0,\\\"reserveY\\\":55616},{\\\"id\\\":8349209,\\\"reserveX\\\":0,\\\"reserveY\\\":55658},{\\\"id\\\":8349210,\\\"reserveX\\\":0,\\\"reserveY\\\":55682},{\\\"id\\\":8349211,\\\"reserveX\\\":0,\\\"reserveY\\\":55711},{\\\"id\\\":8349212,\\\"reserveX\\\":0,\\\"reserveY\\\":55738},{\\\"id\\\":8349213,\\\"reserveX\\\":0,\\\"reserveY\\\":55764},{\\\"id\\\":8349214,\\\"reserveX\\\":0,\\\"reserveY\\\":55795},{\\\"id\\\":8349215,\\\"reserveX\\\":0,\\\"reserveY\\\":55828},{\\\"id\\\":8349216,\\\"reserveX\\\":0,\\\"reserveY\\\":55861},{\\\"id\\\":8349217,\\\"reserveX\\\":0,\\\"reserveY\\\":55893},{\\\"id\\\":8349218,\\\"reserveX\\\":0,\\\"reserveY\\\":55832},{\\\"id\\\":8349219,\\\"reserveX\\\":0,\\\"reserveY\\\":55792},{\\\"id\\\":8349220,\\\"reserveX\\\":0,\\\"reserveY\\\":55815},{\\\"id\\\":8349221,\\\"reserveX\\\":0,\\\"reserveY\\\":55837},{\\\"id\\\":8349222,\\\"reserveX\\\":0,\\\"reserveY\\\":55860},{\\\"id\\\":8349223,\\\"reserveX\\\":0,\\\"reserveY\\\":55884},{\\\"id\\\":8349224,\\\"reserveX\\\":0,\\\"reserveY\\\":55908},{\\\"id\\\":8349225,\\\"reserveX\\\":0,\\\"reserveY\\\":55936},{\\\"id\\\":8349226,\\\"reserveX\\\":0,\\\"reserveY\\\":55998},{\\\"id\\\":8349227,\\\"reserveX\\\":0,\\\"reserveY\\\":56128},{\\\"id\\\":8349228,\\\"reserveX\\\":0,\\\"reserveY\\\":56138},{\\\"id\\\":8349229,\\\"reserveX\\\":0,\\\"reserveY\\\":56086},{\\\"id\\\":8349230,\\\"reserveX\\\":0,\\\"reserveY\\\":56115},{\\\"id\\\":8349231,\\\"reserveX\\\":0,\\\"reserveY\\\":56129},{\\\"id\\\":8349232,\\\"reserveX\\\":0,\\\"reserveY\\\":56106},{\\\"id\\\":8349233,\\\"reserveX\\\":0,\\\"reserveY\\\":56137},{\\\"id\\\":8349234,\\\"reserveX\\\":0,\\\"reserveY\\\":56167},{\\\"id\\\":8349235,\\\"reserveX\\\":0,\\\"reserveY\\\":56197},{\\\"id\\\":8349236,\\\"reserveX\\\":0,\\\"reserveY\\\":56229},{\\\"id\\\":8349237,\\\"reserveX\\\":0,\\\"reserveY\\\":56267},{\\\"id\\\":8349238,\\\"reserveX\\\":0,\\\"reserveY\\\":56278},{\\\"id\\\":8349239,\\\"reserveX\\\":0,\\\"reserveY\\\":56257},{\\\"id\\\":8349240,\\\"reserveX\\\":0,\\\"reserveY\\\":56285},{\\\"id\\\":8349241,\\\"reserveX\\\":0,\\\"reserveY\\\":56314},{\\\"id\\\":8349242,\\\"reserveX\\\":0,\\\"reserveY\\\":56344},{\\\"id\\\":8349243,\\\"reserveX\\\":0,\\\"reserveY\\\":56435},{\\\"id\\\":8349244,\\\"reserveX\\\":0,\\\"reserveY\\\":56597},{\\\"id\\\":8349245,\\\"reserveX\\\":0,\\\"reserveY\\\":56632},{\\\"id\\\":8349246,\\\"reserveX\\\":0,\\\"reserveY\\\":56667},{\\\"id\\\":8349247,\\\"reserveX\\\":0,\\\"reserveY\\\":56700},{\\\"id\\\":8349248,\\\"reserveX\\\":0,\\\"reserveY\\\":56766},{\\\"id\\\":8349249,\\\"reserveX\\\":0,\\\"reserveY\\\":56828},{\\\"id\\\":8349250,\\\"reserveX\\\":0,\\\"reserveY\\\":57211},{\\\"id\\\":8349251,\\\"reserveX\\\":0,\\\"reserveY\\\":57431},{\\\"id\\\":8349252,\\\"reserveX\\\":0,\\\"reserveY\\\":58023},{\\\"id\\\":8349755,\\\"reserveX\\\":564812756876094,\\\"reserveY\\\":1110},{\\\"id\\\":8349756,\\\"reserveX\\\":566222032937713,\\\"reserveY\\\":0},{\\\"id\\\":8349757,\\\"reserveX\\\":566383069164906,\\\"reserveY\\\":0},{\\\"id\\\":8349758,\\\"reserveX\\\":568771894986579,\\\"reserveY\\\":0},{\\\"id\\\":8349759,\\\"reserveX\\\":569694995258293,\\\"reserveY\\\":0},{\\\"id\\\":8349760,\\\"reserveX\\\":572794501425422,\\\"reserveY\\\":0},{\\\"id\\\":8349761,\\\"reserveX\\\":576208352104446,\\\"reserveY\\\":0},{\\\"id\\\":8349762,\\\"reserveX\\\":576901076731006,\\\"reserveY\\\":0},{\\\"id\\\":8349763,\\\"reserveX\\\":573330192792410,\\\"reserveY\\\":0},{\\\"id\\\":8349764,\\\"reserveX\\\":570613860918593,\\\"reserveY\\\":0},{\\\"id\\\":8349765,\\\"reserveX\\\":569613381375836,\\\"reserveY\\\":0},{\\\"id\\\":8349766,\\\"reserveX\\\":568819246721987,\\\"reserveY\\\":0},{\\\"id\\\":8349767,\\\"reserveX\\\":567977262192847,\\\"reserveY\\\":0},{\\\"id\\\":8349768,\\\"reserveX\\\":566763620573865,\\\"reserveY\\\":0},{\\\"id\\\":8349769,\\\"reserveX\\\":565121423125266,\\\"reserveY\\\":0},{\\\"id\\\":8349770,\\\"reserveX\\\":565262518302035,\\\"reserveY\\\":0},{\\\"id\\\":8349771,\\\"reserveX\\\":563318304513920,\\\"reserveY\\\":0},{\\\"id\\\":8349772,\\\"reserveX\\\":564497655431440,\\\"reserveY\\\":0},{\\\"id\\\":8349773,\\\"reserveX\\\":566142572546915,\\\"reserveY\\\":0},{\\\"id\\\":8349774,\\\"reserveX\\\":565262447014418,\\\"reserveY\\\":0},{\\\"id\\\":8349775,\\\"reserveX\\\":566262923474300,\\\"reserveY\\\":0},{\\\"id\\\":8349776,\\\"reserveX\\\":564517283024390,\\\"reserveY\\\":0},{\\\"id\\\":8349777,\\\"reserveX\\\":561589241338238,\\\"reserveY\\\":0},{\\\"id\\\":8349778,\\\"reserveX\\\":560936445049174,\\\"reserveY\\\":0},{\\\"id\\\":8349779,\\\"reserveX\\\":559501523740768,\\\"reserveY\\\":0},{\\\"id\\\":8349780,\\\"reserveX\\\":557675217048338,\\\"reserveY\\\":0},{\\\"id\\\":8349781,\\\"reserveX\\\":559717126377203,\\\"reserveY\\\":0},{\\\"id\\\":8349782,\\\"reserveX\\\":560044854494589,\\\"reserveY\\\":0},{\\\"id\\\":8349783,\\\"reserveX\\\":561535057946771,\\\"reserveY\\\":0},{\\\"id\\\":8349784,\\\"reserveX\\\":562714156048319,\\\"reserveY\\\":0},{\\\"id\\\":8349785,\\\"reserveX\\\":561496247167158,\\\"reserveY\\\":0},{\\\"id\\\":8349786,\\\"reserveX\\\":560963361181229,\\\"reserveY\\\":0},{\\\"id\\\":8349787,\\\"reserveX\\\":561422674781980,\\\"reserveY\\\":0},{\\\"id\\\":8349788,\\\"reserveX\\\":558519596026790,\\\"reserveY\\\":0},{\\\"id\\\":8349789,\\\"reserveX\\\":556874438270049,\\\"reserveY\\\":0},{\\\"id\\\":8349790,\\\"reserveX\\\":557380113795493,\\\"reserveY\\\":0},{\\\"id\\\":8349791,\\\"reserveX\\\":556781906213626,\\\"reserveY\\\":0},{\\\"id\\\":8349792,\\\"reserveX\\\":557320566778277,\\\"reserveY\\\":0},{\\\"id\\\":8349793,\\\"reserveX\\\":558114305271853,\\\"reserveY\\\":0},{\\\"id\\\":8349794,\\\"reserveX\\\":558844850578295,\\\"reserveY\\\":0},{\\\"id\\\":8349795,\\\"reserveX\\\":561496289299344,\\\"reserveY\\\":0},{\\\"id\\\":8349796,\\\"reserveX\\\":560249294801984,\\\"reserveY\\\":0},{\\\"id\\\":8349797,\\\"reserveX\\\":559554048564326,\\\"reserveY\\\":0},{\\\"id\\\":8349798,\\\"reserveX\\\":561382985609186,\\\"reserveY\\\":0},{\\\"id\\\":8349799,\\\"reserveX\\\":561694869021416,\\\"reserveY\\\":0},{\\\"id\\\":8349800,\\\"reserveX\\\":559377814447303,\\\"reserveY\\\":0},{\\\"id\\\":8349801,\\\"reserveX\\\":557732469627899,\\\"reserveY\\\":0},{\\\"id\\\":8349802,\\\"reserveX\\\":555401203197451,\\\"reserveY\\\":0},{\\\"id\\\":8349803,\\\"reserveX\\\":553325019888790,\\\"reserveY\\\":0},{\\\"id\\\":8349804,\\\"reserveX\\\":552664105207866,\\\"reserveY\\\":0},{\\\"id\\\":8349805,\\\"reserveX\\\":871832442990572,\\\"reserveY\\\":0},{\\\"id\\\":8349806,\\\"reserveX\\\":320918934725615,\\\"reserveY\\\":0},{\\\"id\\\":8349807,\\\"reserveX\\\":320611962587757,\\\"reserveY\\\":0},{\\\"id\\\":8349808,\\\"reserveX\\\":320080881980074,\\\"reserveY\\\":0},{\\\"id\\\":8349809,\\\"reserveX\\\":320869300918878,\\\"reserveY\\\":0},{\\\"id\\\":8349810,\\\"reserveX\\\":320663314788781,\\\"reserveY\\\":0},{\\\"id\\\":8349811,\\\"reserveX\\\":319201427602292,\\\"reserveY\\\":0},{\\\"id\\\":8349812,\\\"reserveX\\\":319244147647179,\\\"reserveY\\\":0},{\\\"id\\\":8349813,\\\"reserveX\\\":319730099731390,\\\"reserveY\\\":0},{\\\"id\\\":8349814,\\\"reserveX\\\":319201103651368,\\\"reserveY\\\":0},{\\\"id\\\":8349815,\\\"reserveX\\\":318495010918056,\\\"reserveY\\\":0},{\\\"id\\\":8349816,\\\"reserveX\\\":318009945064367,\\\"reserveY\\\":0},{\\\"id\\\":8349817,\\\"reserveX\\\":317431027288969,\\\"reserveY\\\":0},{\\\"id\\\":8349818,\\\"reserveX\\\":316836843428652,\\\"reserveY\\\":0},{\\\"id\\\":8349819,\\\"reserveX\\\":317913064267435,\\\"reserveY\\\":0},{\\\"id\\\":8349820,\\\"reserveX\\\":319595230182170,\\\"reserveY\\\":0},{\\\"id\\\":8349821,\\\"reserveX\\\":320672300263753,\\\"reserveY\\\":0},{\\\"id\\\":8349822,\\\"reserveX\\\":320593694504787,\\\"reserveY\\\":0},{\\\"id\\\":8349823,\\\"reserveX\\\":321522137435740,\\\"reserveY\\\":0},{\\\"id\\\":8349824,\\\"reserveX\\\":321878945275429,\\\"reserveY\\\":0},{\\\"id\\\":8349825,\\\"reserveX\\\":319805794333927,\\\"reserveY\\\":0},{\\\"id\\\":8349826,\\\"reserveX\\\":318971936598309,\\\"reserveY\\\":0},{\\\"id\\\":8349827,\\\"reserveX\\\":316794656269427,\\\"reserveY\\\":0},{\\\"id\\\":8349828,\\\"reserveX\\\":315698206539213,\\\"reserveY\\\":0},{\\\"id\\\":8349829,\\\"reserveX\\\":315214993779182,\\\"reserveY\\\":0},{\\\"id\\\":8349830,\\\"reserveX\\\":316022023418652,\\\"reserveY\\\":0},{\\\"id\\\":8349831,\\\"reserveX\\\":315848839977066,\\\"reserveY\\\":0},{\\\"id\\\":8349832,\\\"reserveX\\\":315767619303043,\\\"reserveY\\\":0},{\\\"id\\\":8349833,\\\"reserveX\\\":315455982685053,\\\"reserveY\\\":0},{\\\"id\\\":8349834,\\\"reserveX\\\":315781191309378,\\\"reserveY\\\":0},{\\\"id\\\":8349835,\\\"reserveX\\\":315972214083905,\\\"reserveY\\\":0},{\\\"id\\\":8349836,\\\"reserveX\\\":316625958793772,\\\"reserveY\\\":0},{\\\"id\\\":8349837,\\\"reserveX\\\":316307168098843,\\\"reserveY\\\":0},{\\\"id\\\":8349838,\\\"reserveX\\\":316819517249341,\\\"reserveY\\\":0},{\\\"id\\\":8349839,\\\"reserveX\\\":316639795970518,\\\"reserveY\\\":0},{\\\"id\\\":8349840,\\\"reserveX\\\":316966726273721,\\\"reserveY\\\":0},{\\\"id\\\":8349841,\\\"reserveX\\\":316882534975972,\\\"reserveY\\\":0},{\\\"id\\\":8349842,\\\"reserveX\\\":317599634536951,\\\"reserveY\\\":0},{\\\"id\\\":8349843,\\\"reserveX\\\":318370315863316,\\\"reserveY\\\":0},{\\\"id\\\":8349844,\\\"reserveX\\\":319374281717007,\\\"reserveY\\\":0},{\\\"id\\\":8349845,\\\"reserveX\\\":319333532164987,\\\"reserveY\\\":0},{\\\"id\\\":8349846,\\\"reserveX\\\":318833035124957,\\\"reserveY\\\":0},{\\\"id\\\":8349847,\\\"reserveX\\\":317578784399709,\\\"reserveY\\\":0},{\\\"id\\\":8349848,\\\"reserveX\\\":316590674089892,\\\"reserveY\\\":0},{\\\"id\\\":8349849,\\\"reserveX\\\":315622237094855,\\\"reserveY\\\":0},{\\\"id\\\":8349850,\\\"reserveX\\\":313460440911039,\\\"reserveY\\\":0},{\\\"id\\\":8349851,\\\"reserveX\\\":313340446734398,\\\"reserveY\\\":0},{\\\"id\\\":8349852,\\\"reserveX\\\":323943059038112,\\\"reserveY\\\":0},{\\\"id\\\":8349853,\\\"reserveX\\\":325021501815535,\\\"reserveY\\\":0},{\\\"id\\\":8349854,\\\"reserveX\\\":324257250590595,\\\"reserveY\\\":0},{\\\"id\\\":8349855,\\\"reserveX\\\":363745464201157,\\\"reserveY\\\":0},{\\\"id\\\":8349856,\\\"reserveX\\\":51627007055885,\\\"reserveY\\\":0},{\\\"id\\\":8349857,\\\"reserveX\\\":51652343273441,\\\"reserveY\\\":0},{\\\"id\\\":8349858,\\\"reserveX\\\":51679179168520,\\\"reserveY\\\":0},{\\\"id\\\":8349859,\\\"reserveX\\\":51553253369757,\\\"reserveY\\\":0},{\\\"id\\\":8349860,\\\"reserveX\\\":51517047350708,\\\"reserveY\\\":0},{\\\"id\\\":8349861,\\\"reserveX\\\":51537713717090,\\\"reserveY\\\":0},{\\\"id\\\":8349862,\\\"reserveX\\\":51365577533599,\\\"reserveY\\\":0},{\\\"id\\\":8349863,\\\"reserveX\\\":51404045584150,\\\"reserveY\\\":0},{\\\"id\\\":8349864,\\\"reserveX\\\":51520546340623,\\\"reserveY\\\":0},{\\\"id\\\":8349865,\\\"reserveX\\\":51543902699535,\\\"reserveY\\\":0},{\\\"id\\\":8349866,\\\"reserveX\\\":51420443436890,\\\"reserveY\\\":0},{\\\"id\\\":8349867,\\\"reserveX\\\":51431791507870,\\\"reserveY\\\":0},{\\\"id\\\":8349868,\\\"reserveX\\\":51362072656777,\\\"reserveY\\\":0},{\\\"id\\\":8349869,\\\"reserveX\\\":51311067596201,\\\"reserveY\\\":0},{\\\"id\\\":8349870,\\\"reserveX\\\":51319023848881,\\\"reserveY\\\":0},{\\\"id\\\":8349871,\\\"reserveX\\\":51285536568871,\\\"reserveY\\\":0},{\\\"id\\\":8349872,\\\"reserveX\\\":51294211065402,\\\"reserveY\\\":0},{\\\"id\\\":8349873,\\\"reserveX\\\":51371170521700,\\\"reserveY\\\":0},{\\\"id\\\":8349874,\\\"reserveX\\\":51376645447772,\\\"reserveY\\\":0},{\\\"id\\\":8349875,\\\"reserveX\\\":51347451421128,\\\"reserveY\\\":0},{\\\"id\\\":8349876,\\\"reserveX\\\":51344417212311,\\\"reserveY\\\":0},{\\\"id\\\":8349877,\\\"reserveX\\\":51346484297007,\\\"reserveY\\\":0},{\\\"id\\\":8349878,\\\"reserveX\\\":51336229057913,\\\"reserveY\\\":0},{\\\"id\\\":8349879,\\\"reserveX\\\":51256448991420,\\\"reserveY\\\":0},{\\\"id\\\":8349880,\\\"reserveX\\\":51125884727075,\\\"reserveY\\\":0},{\\\"id\\\":8349881,\\\"reserveX\\\":51119068541686,\\\"reserveY\\\":0},{\\\"id\\\":8349882,\\\"reserveX\\\":51165624359026,\\\"reserveY\\\":0},{\\\"id\\\":8349883,\\\"reserveX\\\":51174319308121,\\\"reserveY\\\":0},{\\\"id\\\":8349884,\\\"reserveX\\\":51185022576589,\\\"reserveY\\\":0},{\\\"id\\\":8349885,\\\"reserveX\\\":50996236960866,\\\"reserveY\\\":0},{\\\"id\\\":8349886,\\\"reserveX\\\":50853925223483,\\\"reserveY\\\":0},{\\\"id\\\":8349887,\\\"reserveX\\\":50465409505707,\\\"reserveY\\\":0},{\\\"id\\\":8349888,\\\"reserveX\\\":50510890139408,\\\"reserveY\\\":0},{\\\"id\\\":8349889,\\\"reserveX\\\":50519915672910,\\\"reserveY\\\":0},{\\\"id\\\":8349890,\\\"reserveX\\\":50473443489484,\\\"reserveY\\\":0},{\\\"id\\\":8349891,\\\"reserveX\\\":50534950534878,\\\"reserveY\\\":0},{\\\"id\\\":8349892,\\\"reserveX\\\":50351363296583,\\\"reserveY\\\":0},{\\\"id\\\":8349893,\\\"reserveX\\\":50126697301370,\\\"reserveY\\\":0},{\\\"id\\\":8349894,\\\"reserveX\\\":49941591101969,\\\"reserveY\\\":0},{\\\"id\\\":8349895,\\\"reserveX\\\":49902570046199,\\\"reserveY\\\":0},{\\\"id\\\":8349896,\\\"reserveX\\\":49851121813115,\\\"reserveY\\\":0},{\\\"id\\\":8349897,\\\"reserveX\\\":49902221540152,\\\"reserveY\\\":0},{\\\"id\\\":8349898,\\\"reserveX\\\":49905603652574,\\\"reserveY\\\":0},{\\\"id\\\":8349899,\\\"reserveX\\\":49962902937801,\\\"reserveY\\\":0},{\\\"id\\\":8349900,\\\"reserveX\\\":50029229506887,\\\"reserveY\\\":0},{\\\"id\\\":8349901,\\\"reserveX\\\":49998554772920,\\\"reserveY\\\":0},{\\\"id\\\":8349902,\\\"reserveX\\\":1086153607398897,\\\"reserveY\\\":0},{\\\"id\\\":8349903,\\\"reserveX\\\":59561262645599,\\\"reserveY\\\":0},{\\\"id\\\":8349904,\\\"reserveX\\\":59619083442183,\\\"reserveY\\\":0},{\\\"id\\\":8349905,\\\"reserveX\\\":59487861575707,\\\"reserveY\\\":0},{\\\"id\\\":8349906,\\\"reserveX\\\":19804025289247,\\\"reserveY\\\":0},{\\\"id\\\":8349907,\\\"reserveX\\\":19798053169895,\\\"reserveY\\\":0},{\\\"id\\\":8349908,\\\"reserveX\\\":19789655100748,\\\"reserveY\\\":0},{\\\"id\\\":8349909,\\\"reserveX\\\":19791183943560,\\\"reserveY\\\":0},{\\\"id\\\":8349910,\\\"reserveX\\\":19788014689299,\\\"reserveY\\\":0},{\\\"id\\\":8349911,\\\"reserveX\\\":19770839528144,\\\"reserveY\\\":0},{\\\"id\\\":8349912,\\\"reserveX\\\":19767016414980,\\\"reserveY\\\":0},{\\\"id\\\":8349913,\\\"reserveX\\\":19767038165915,\\\"reserveY\\\":0},{\\\"id\\\":8349914,\\\"reserveX\\\":19760249312848,\\\"reserveY\\\":0},{\\\"id\\\":8349915,\\\"reserveX\\\":19780099390298,\\\"reserveY\\\":0},{\\\"id\\\":8349916,\\\"reserveX\\\":19793050106807,\\\"reserveY\\\":0},{\\\"id\\\":8349917,\\\"reserveX\\\":19803566441375,\\\"reserveY\\\":0},{\\\"id\\\":8349918,\\\"reserveX\\\":19843013129075,\\\"reserveY\\\":0},{\\\"id\\\":8349919,\\\"reserveX\\\":19837372772473,\\\"reserveY\\\":0},{\\\"id\\\":8349920,\\\"reserveX\\\":19835086147532,\\\"reserveY\\\":0},{\\\"id\\\":8349921,\\\"reserveX\\\":19846246146986,\\\"reserveY\\\":0},{\\\"id\\\":8349922,\\\"reserveX\\\":19812221779346,\\\"reserveY\\\":0},{\\\"id\\\":8349923,\\\"reserveX\\\":19759945223139,\\\"reserveY\\\":0},{\\\"id\\\":8349924,\\\"reserveX\\\":19719317783813,\\\"reserveY\\\":0},{\\\"id\\\":8349925,\\\"reserveX\\\":19721571683108,\\\"reserveY\\\":0},{\\\"id\\\":8349926,\\\"reserveX\\\":19724268969842,\\\"reserveY\\\":0},{\\\"id\\\":8349927,\\\"reserveX\\\":19726840318867,\\\"reserveY\\\":0},{\\\"id\\\":8349928,\\\"reserveX\\\":19732033984398,\\\"reserveY\\\":0},{\\\"id\\\":8349929,\\\"reserveX\\\":19728835172207,\\\"reserveY\\\":0},{\\\"id\\\":8349930,\\\"reserveX\\\":19703695748179,\\\"reserveY\\\":0},{\\\"id\\\":8349931,\\\"reserveX\\\":19679842832208,\\\"reserveY\\\":0},{\\\"id\\\":8349932,\\\"reserveX\\\":19674566742194,\\\"reserveY\\\":0},{\\\"id\\\":8349933,\\\"reserveX\\\":19676357318402,\\\"reserveY\\\":0},{\\\"id\\\":8349934,\\\"reserveX\\\":19681470522181,\\\"reserveY\\\":0},{\\\"id\\\":8349935,\\\"reserveX\\\":19697364186317,\\\"reserveY\\\":0},{\\\"id\\\":8349936,\\\"reserveX\\\":19699984457762,\\\"reserveY\\\":0},{\\\"id\\\":8349937,\\\"reserveX\\\":19702425486553,\\\"reserveY\\\":0},{\\\"id\\\":8349938,\\\"reserveX\\\":19705602139646,\\\"reserveY\\\":0},{\\\"id\\\":8349939,\\\"reserveX\\\":19712058140103,\\\"reserveY\\\":0},{\\\"id\\\":8349940,\\\"reserveX\\\":19721355104139,\\\"reserveY\\\":0},{\\\"id\\\":8349941,\\\"reserveX\\\":19714571351158,\\\"reserveY\\\":0},{\\\"id\\\":8349942,\\\"reserveX\\\":19698214473887,\\\"reserveY\\\":0},{\\\"id\\\":8349943,\\\"reserveX\\\":19697574800527,\\\"reserveY\\\":0},{\\\"id\\\":8349944,\\\"reserveX\\\":19697272191434,\\\"reserveY\\\":0},{\\\"id\\\":8349945,\\\"reserveX\\\":19699038959354,\\\"reserveY\\\":0},{\\\"id\\\":8349946,\\\"reserveX\\\":19700397975672,\\\"reserveY\\\":0},{\\\"id\\\":8349947,\\\"reserveX\\\":19701102757113,\\\"reserveY\\\":0},{\\\"id\\\":8349948,\\\"reserveX\\\":19702143054652,\\\"reserveY\\\":0},{\\\"id\\\":8349949,\\\"reserveX\\\":19703518504625,\\\"reserveY\\\":0},{\\\"id\\\":8349950,\\\"reserveX\\\":19705203806808,\\\"reserveY\\\":0},{\\\"id\\\":8349951,\\\"reserveX\\\":19707520236542,\\\"reserveY\\\":0},{\\\"id\\\":8349952,\\\"reserveX\\\":19677241167586,\\\"reserveY\\\":0}],\\\"liquidity\\\":186900225,\\\"priceX128\\\":1250622376476148626307841371099}\",\"blockNumber\":293298842}"

func initPoolSimulator() *PoolSimulator {
	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(entityStr), &poolEntity)
	if err != nil {
		panic(err)
	}
	simulator, _ := NewPoolSimulator(poolEntity)
	return simulator
}

// Before optimized
// BenchmarkOptimizePoolSimulator-10    	    4346	    284726 ns/op	  336061 B/op	    8675 allocs/op
// After optimized
// BenchmarkOptimizePoolSimulator-10    	  362793	      3157 ns/op	    2531 B/op	      65 allocs/op
func BenchmarkOptimizePoolSimulator(b *testing.B) {
	simulator := initPoolSimulator()
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", // WETH
			Amount: big.NewInt(1000000000000000),
		},
		TokenOut: "0xaf88d065e77c8cc2239327c5edb3a432268e5831", // USDC
	}

	for i := 0; i < b.N; i++ {
		_, err := simulator.CalcAmountOut(params) // 2817957 USDC
		assert.Nil(b, err)
	}
}

func TestOptimizeGetPriceFromIDWorkCorrectly(t *testing.T) {
	simulator := initPoolSimulator()

	for _, bin := range simulator.bins {
		priceBackup, errBackup := getPriceFromIDBackup(bin.ID, simulator.binStep)
		price, err := getPriceFromID(bin.ID, simulator.binStep)
		assert.Equal(t, priceBackup, price)
		assert.Equal(t, errBackup, err)
	}
}

// Before optimized (getPriceFromIDBackup)
// BenchmarkOptimizeGetPriceFromID-10    	  366819	      3094 ns/op	    3987 B/op	     105 allocs/op
// After optimized
// BenchmarkOptimizeGetPriceFromID-10    	  809637	      1594 ns/op	     728 B/op	      20 allocs/op
func BenchmarkOptimizeGetPriceFromID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := getPriceFromID(8349152, 5)
		assert.Nil(b, err)
	}
}
