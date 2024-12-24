package pancakev3

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var poolEncoded = `{
  "address": "0xd4dca84e1808da3354924cd243c66828cf775470",
  "reserveUsd": 8366665.950394863,
  "amplifiedTvl": 5.160057377140917E+54,
  "swapFee": 2500,
  "exchange": "pancake-v3",
  "type": "pancake-v3",
  "timestamp": 1730449645,
  "reserves": [
    "2240679306918813463602",
    "40017763330367824175"
  ],
  "tokens": [
    {
      "address": "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
      "name": "Ethereum Token",
      "symbol": "ETH",
      "decimals": 18,
      "weight": 50,
      "swappable": true
    },
    {
      "address": "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
      "name": "BTCB Token",
      "symbol": "BTCB",
      "decimals": 18,
      "weight": 50,
      "swappable": true
    }
  ],
  "extra": "{\"liquidity\":4967388667821564946603,\"sqrtPriceX96\":15051825994443620570236299050,\"tick\":-33219,\"ticks\":[{\"index\":-887250,\"liquidityGross\":66053528479507347108,\"liquidityNet\":66053528479507347108},{\"index\":-39900,\"liquidityGross\":70508896873622922,\"liquidityNet\":70508896873622922},{\"index\":-39400,\"liquidityGross\":48740047716641799,\"liquidityNet\":48740047716641799},{\"index\":-39300,\"liquidityGross\":23659350773676994,\"liquidityNet\":23659350773676994},{\"index\":-39250,\"liquidityGross\":1070403859429138319,\"liquidityNet\":1070403859429138319},{\"index\":-39150,\"liquidityGross\":79685265646310494,\"liquidityNet\":79685265646310494},{\"index\":-39050,\"liquidityGross\":180322259085125862,\"liquidityNet\":180322259085125862},{\"index\":-39000,\"liquidityGross\":5143776120734644,\"liquidityNet\":5143776120734644},{\"index\":-38600,\"liquidityGross\":654003707535727983,\"liquidityNet\":654003707535727983},{\"index\":-38500,\"liquidityGross\":7712216739938003,\"liquidityNet\":7712216739938003},{\"index\":-38350,\"liquidityGross\":2010949834182953415,\"liquidityNet\":2010949834182953415},{\"index\":-38300,\"liquidityGross\":313070763005502157,\"liquidityNet\":313070763005502157},{\"index\":-38050,\"liquidityGross\":751504450314768756,\"liquidityNet\":751504450314768756},{\"index\":-38000,\"liquidityGross\":7096251082163236,\"liquidityNet\":7096251082163236},{\"index\":-37950,\"liquidityGross\":376984490216644370,\"liquidityNet\":376984490216644370},{\"index\":-37700,\"liquidityGross\":261555374264484119,\"liquidityNet\":261555374264484119},{\"index\":-37450,\"liquidityGross\":2341950946990431848,\"liquidityNet\":2341950946990431848},{\"index\":-37250,\"liquidityGross\":1958557830789732,\"liquidityNet\":1958557830789732},{\"index\":-37100,\"liquidityGross\":3012592186226944078,\"liquidityNet\":3012592186226944078},{\"index\":-36950,\"liquidityGross\":34724366629024078,\"liquidityNet\":34724366629024078},{\"index\":-36900,\"liquidityGross\":359635869229096763,\"liquidityNet\":359635869229096763},{\"index\":-36800,\"liquidityGross\":2129821827137791908,\"liquidityNet\":2129821827137791908},{\"index\":-36600,\"liquidityGross\":71811518042567314,\"liquidityNet\":71811518042567314},{\"index\":-36550,\"liquidityGross\":357391934023365169,\"liquidityNet\":357391934023365169},{\"index\":-36500,\"liquidityGross\":8120306524201149818,\"liquidityNet\":8120306524201149818},{\"index\":-36400,\"liquidityGross\":391689475637218890,\"liquidityNet\":391689475637218890},{\"index\":-36200,\"liquidityGross\":1057700039981467654,\"liquidityNet\":1057700039981467654},{\"index\":-36100,\"liquidityGross\":120658034563339440391,\"liquidityNet\":120658034563339440391},{\"index\":-36000,\"liquidityGross\":2189398961178606507,\"liquidityNet\":2189398961178606507},{\"index\":-35950,\"liquidityGross\":3204943908486862625,\"liquidityNet\":3204943908486862625},{\"index\":-35900,\"liquidityGross\":2082254671900874500,\"liquidityNet\":2082254671900874500},{\"index\":-35850,\"liquidityGross\":2189055673132301427,\"liquidityNet\":2189055673132301427},{\"index\":-35800,\"liquidityGross\":562133037136579063,\"liquidityNet\":562133037136579063},{\"index\":-35400,\"liquidityGross\":325599427483001936,\"liquidityNet\":325599427483001936},{\"index\":-35250,\"liquidityGross\":2681382700770850151,\"liquidityNet\":2681382700770850151},{\"index\":-35100,\"liquidityGross\":359203498191368257,\"liquidityNet\":359203498191368257},{\"index\":-35050,\"liquidityGross\":139015399798904475,\"liquidityNet\":139015399798904475},{\"index\":-34950,\"liquidityGross\":1560599854262717957,\"liquidityNet\":1560599854262717957},{\"index\":-34800,\"liquidityGross\":742078212575651285,\"liquidityNet\":742078212575651285},{\"index\":-34750,\"liquidityGross\":369328196340005770,\"liquidityNet\":369328196340005770},{\"index\":-34650,\"liquidityGross\":11159441939292152996,\"liquidityNet\":11159441939292152996},{\"index\":-34600,\"liquidityGross\":13251999985035936105,\"liquidityNet\":13251999985035936105},{\"index\":-34550,\"liquidityGross\":1733246326815422038,\"liquidityNet\":1733246326815422038},{\"index\":-34500,\"liquidityGross\":23270792554446629919,\"liquidityNet\":23270792554446629919},{\"index\":-34450,\"liquidityGross\":7938413972098821699,\"liquidityNet\":7938413972098821699},{\"index\":-34400,\"liquidityGross\":1061333418426388596,\"liquidityNet\":1061333418426388596},{\"index\":-34350,\"liquidityGross\":10830963848694202977,\"liquidityNet\":10830963848694202977},{\"index\":-34300,\"liquidityGross\":142982183882240432,\"liquidityNet\":142982183882240432},{\"index\":-34250,\"liquidityGross\":3856345364525466519,\"liquidityNet\":3856345364525466519},{\"index\":-34200,\"liquidityGross\":732673489436891573,\"liquidityNet\":732673489436891573},{\"index\":-34150,\"liquidityGross\":20314140289507416,\"liquidityNet\":20314140289507416},{\"index\":-34100,\"liquidityGross\":957459039268850926,\"liquidityNet\":957459039268850926},{\"index\":-34050,\"liquidityGross\":4592196813373927857,\"liquidityNet\":4592196813373927857},{\"index\":-34000,\"liquidityGross\":12602525788451291460,\"liquidityNet\":12602525788451291460},{\"index\":-33950,\"liquidityGross\":571894497401295581,\"liquidityNet\":571894497401295581},{\"index\":-33900,\"liquidityGross\":306933585440654111,\"liquidityNet\":306933585440654111},{\"index\":-33850,\"liquidityGross\":26204925501242428398,\"liquidityNet\":26204925501242428398},{\"index\":-33800,\"liquidityGross\":3550849895193204179167,\"liquidityNet\":3550849895193204179167},{\"index\":-33750,\"liquidityGross\":116317722897312231,\"liquidityNet\":116317722897312231},{\"index\":-33700,\"liquidityGross\":1791607215593244417,\"liquidityNet\":1791607215593244417},{\"index\":-33650,\"liquidityGross\":4553978044616841218,\"liquidityNet\":4553978044616841218},{\"index\":-33600,\"liquidityGross\":21030327465323757394,\"liquidityNet\":21030327465323757394},{\"index\":-33550,\"liquidityGross\":54047275512765298888,\"liquidityNet\":54047275512765298888},{\"index\":-33500,\"liquidityGross\":8505610862923789096,\"liquidityNet\":8505610862923789096},{\"index\":-33450,\"liquidityGross\":4521328538950080245,\"liquidityNet\":4521328538950080245},{\"index\":-33400,\"liquidityGross\":971620788422850179183,\"liquidityNet\":971620788422850179183},{\"index\":-33350,\"liquidityGross\":7222624591740603,\"liquidityNet\":7222624591740603},{\"index\":-33300,\"liquidityGross\":3341563221484753203,\"liquidityNet\":3341563221484753203},{\"index\":-33250,\"liquidityGross\":910830175219870578,\"liquidityNet\":910830175219870578},{\"index\":-33200,\"liquidityGross\":11164381103589905992,\"liquidityNet\":11164381103589905992},{\"index\":-33100,\"liquidityGross\":13117533733813945838,\"liquidityNet\":13117533733813945838},{\"index\":-33050,\"liquidityGross\":43899197920548461302,\"liquidityNet\":43899197920548461302},{\"index\":-33000,\"liquidityGross\":23679782504822625273,\"liquidityNet\":23679782504822625273},{\"index\":-32950,\"liquidityGross\":17170545424397660023,\"liquidityNet\":17170545424397660023},{\"index\":-32850,\"liquidityGross\":1174840440937019695,\"liquidityNet\":1174840440937019695},{\"index\":-32800,\"liquidityGross\":14,\"liquidityNet\":14},{\"index\":-32750,\"liquidityGross\":3526176524052314999588,\"liquidityNet\":-3511542101523701418890},{\"index\":-32700,\"liquidityGross\":53965746948312936,\"liquidityNet\":53965746948312936},{\"index\":-32650,\"liquidityGross\":971620788422850179183,\"liquidityNet\":-971620788422850179183},{\"index\":-32600,\"liquidityGross\":62617484683365574415,\"liquidityNet\":33326580361086787841},{\"index\":-32550,\"liquidityGross\":7332839330711618141,\"liquidityNet\":-7332839330711618141},{\"index\":-32500,\"liquidityGross\":1694361199337064915,\"liquidityNet\":1694361199337064915},{\"index\":-32450,\"liquidityGross\":12565790636334492594,\"liquidityNet\":9604323116490170548},{\"index\":-32400,\"liquidityGross\":11203577989420255137,\"liquidityNet\":-10072321526172626391},{\"index\":-32300,\"liquidityGross\":921632048958158391,\"liquidityNet\":-207464131399237085},{\"index\":-32250,\"liquidityGross\":41637415395577105963,\"liquidityNet\":41637415395577105963},{\"index\":-32200,\"liquidityGross\":13763545011294905711,\"liquidityNet\":-8455530969921671525},{\"index\":-32150,\"liquidityGross\":4917526152576056917,\"liquidityNet\":-4055246477907253939},{\"index\":-32100,\"liquidityGross\":2105518836188422913,\"liquidityNet\":2105518836188422913},{\"index\":-32050,\"liquidityGross\":6369976836238769350,\"liquidityNet\":6369976836238769350},{\"index\":-32000,\"liquidityGross\":2083670328205660698,\"liquidityNet\":464692431267919154},{\"index\":-31950,\"liquidityGross\":1889785650130462300,\"liquidityNet\":1889785650130462300},{\"index\":-31900,\"liquidityGross\":152390982303868870,\"liquidityNet\":152390982303868870},{\"index\":-31850,\"liquidityGross\":50644232576124406372,\"liquidityNet\":50644232576124406372},{\"index\":-31800,\"liquidityGross\":187965289554157276,\"liquidityNet\":187965289554157276},{\"index\":-31750,\"liquidityGross\":703066654701225860,\"liquidityNet\":703066654701225860},{\"index\":-31700,\"liquidityGross\":37318598945981629490,\"liquidityNet\":37318598945981629490},{\"index\":-31650,\"liquidityGross\":525708490704409,\"liquidityNet\":525708490704409},{\"index\":-31600,\"liquidityGross\":6117899258390635894,\"liquidityNet\":-673158678475496384},{\"index\":-31550,\"liquidityGross\":562649739463143807,\"liquidityNet\":562649739463143807},{\"index\":-31500,\"liquidityGross\":7987983321937385361,\"liquidityNet\":-7903138970613974087},{\"index\":-31450,\"liquidityGross\":10107403045799387129,\"liquidityNet\":4798671651210257729},{\"index\":-31400,\"liquidityGross\":963940382516812982,\"liquidityNet\":-146939211103007294},{\"index\":-31350,\"liquidityGross\":5420295026534386429,\"liquidityNet\":2411043293540162667},{\"index\":-31300,\"liquidityGross\":995356704226199,\"liquidityNet\":995356704226199},{\"index\":-31250,\"liquidityGross\":2508851946450891012,\"liquidityNet\":-2450316005584119136},{\"index\":-31200,\"liquidityGross\":59432930814455061189,\"liquidityNet\":-41855534337793751555},{\"index\":-31150,\"liquidityGross\":1580705756314240047,\"liquidityNet\":1580705756314240047},{\"index\":-31100,\"liquidityGross\":39795603888584015184,\"liquidityNet\":14849196480630027940},{\"index\":-31050,\"liquidityGross\":9788881074524276539,\"liquidityNet\":-833024172157684075},{\"index\":-31000,\"liquidityGross\":315785133863122299,\"liquidityNet\":315785133863122299},{\"index\":-30950,\"liquidityGross\":325599427483001936,\"liquidityNet\":-325599427483001936},{\"index\":-30900,\"liquidityGross\":32447427720415326295,\"liquidityNet\":-14912137289229924251},{\"index\":-30850,\"liquidityGross\":17177821892025339653,\"liquidityNet\":-17163268956769980393},{\"index\":-30800,\"liquidityGross\":8061675048505493159,\"liquidityNet\":-7977912537661609543},{\"index\":-30750,\"liquidityGross\":481640308228436027,\"liquidityNet\":363726928265655209},{\"index\":-30700,\"liquidityGross\":28676075687278411627,\"liquidityNet\":-3761840392659341853},{\"index\":-30650,\"liquidityGross\":19114149660380520149,\"liquidityNet\":4479727131766939451},{\"index\":-30600,\"liquidityGross\":8932783097294097185,\"liquidityNet\":8214376100911360671},{\"index\":-30550,\"liquidityGross\":116899904225082887,\"liquidityNet\":116899904225082887},{\"index\":-30400,\"liquidityGross\":61894037026616831,\"liquidityNet\":61894037026616831},{\"index\":-30350,\"liquidityGross\":17076024044963216216,\"liquidityNet\":15320514060298390910},{\"index\":-30300,\"liquidityGross\":187283912526647442,\"liquidityNet\":187283912526647442},{\"index\":-30250,\"liquidityGross\":15782737162,\"liquidityNet\":-15782737162},{\"index\":-30200,\"liquidityGross\":73498670992647884329,\"liquidityNet\":73361354835099772581},{\"index\":-30150,\"liquidityGross\":4639086137412637272,\"liquidityNet\":4639086137412637272},{\"index\":-30100,\"liquidityGross\":6945918621750340803,\"liquidityNet\":3493322980668611441},{\"index\":-30050,\"liquidityGross\":296129038934931964,\"liquidityNet\":296129038934931964},{\"index\":-30000,\"liquidityGross\":2323201096309621677,\"liquidityNet\":-1861188267139062709},{\"index\":-29950,\"liquidityGross\":8873407410074751520,\"liquidityNet\":8873407410074751520},{\"index\":-29900,\"liquidityGross\":114654438633898,\"liquidityNet\":114654438633898},{\"index\":-29850,\"liquidityGross\":15396895305261260099,\"liquidityNet\":11630489708717075047},{\"index\":-29800,\"liquidityGross\":118217467711147,\"liquidityNet\":118217467711147},{\"index\":-29750,\"liquidityGross\":6644589131935143311,\"liquidityNet\":6644589131935143311},{\"index\":-29700,\"liquidityGross\":155357858500522,\"liquidityNet\":155357858500522},{\"index\":-29650,\"liquidityGross\":2778184458296887992,\"liquidityNet\":-2778184458296887992},{\"index\":-29600,\"liquidityGross\":9959505463262375085,\"liquidityNet\":871807908995459379},{\"index\":-29550,\"liquidityGross\":201474533231757895,\"liquidityNet\":201474533231757895},{\"index\":-29500,\"liquidityGross\":53889975838266306223,\"liquidityNet\":52824397592122564573},{\"index\":-29450,\"liquidityGross\":394925624921296344,\"liquidityNet\":-394169317964606640},{\"index\":-29400,\"liquidityGross\":213120534271548,\"liquidityNet\":213120534271548},{\"index\":-29350,\"liquidityGross\":421586262280425,\"liquidityNet\":421586262280425},{\"index\":-29300,\"liquidityGross\":171612792743346912873,\"liquidityNet\":171004539079672758423},{\"index\":-29250,\"liquidityGross\":39749341510462103,\"liquidityNet\":-39749341510462103},{\"index\":-29200,\"liquidityGross\":10150706097899325710,\"liquidityNet\":-294987835489257134},{\"index\":-29150,\"liquidityGross\":2865514762504479637,\"liquidityNet\":-2865514762504479637},{\"index\":-29050,\"liquidityGross\":6731501394975631782,\"liquidityNet\":-6390392858808168296},{\"index\":-29000,\"liquidityGross\":26870951229685633740,\"liquidityNet\":-26870951229685633740},{\"index\":-28950,\"liquidityGross\":1999232617955679381,\"liquidityNet\":1998389445431118531},{\"index\":-28900,\"liquidityGross\":11119387207375868781,\"liquidityNet\":10587775339027271245},{\"index\":-28850,\"liquidityGross\":20905140093587917979,\"liquidityNet\":-20905140093587917979},{\"index\":-28800,\"liquidityGross\":66830358525880846447,\"liquidityNet\":-66830358525880846447},{\"index\":-28750,\"liquidityGross\":172018997057403345801,\"liquidityNet\":-172018997057403345801},{\"index\":-28700,\"liquidityGross\":6649832388820834132,\"liquidityNet\":-6639097326256626876},{\"index\":-28600,\"liquidityGross\":11797093753932230322,\"liquidityNet\":-11797093753932230322},{\"index\":-28550,\"liquidityGross\":6217462127251953478,\"liquidityNet\":5948168995183606346},{\"index\":-28500,\"liquidityGross\":118217467711147,\"liquidityNet\":-118217467711147},{\"index\":-28450,\"liquidityGross\":320195738897277,\"liquidityNet\":-320195738897277},{\"index\":-28400,\"liquidityGross\":1211878073440495853,\"liquidityNet\":1043779180010612307},{\"index\":-28350,\"liquidityGross\":1250123295692838489,\"liquidityNet\":-1250123295692838489},{\"index\":-28300,\"liquidityGross\":5579366043730861710,\"liquidityNet\":-5316832117921134234},{\"index\":-28250,\"liquidityGross\":16443448409018972743,\"liquidityNet\":-16443448409018972743},{\"index\":-28150,\"liquidityGross\":249553603219959269,\"liquidityNet\":-249553603219959269},{\"index\":-28100,\"liquidityGross\":76573332628405683180,\"liquidityNet\":-76573332628405683180},{\"index\":-28050,\"liquidityGross\":247062279676168634643,\"liquidityNet\":245569142562483865919},{\"index\":-28000,\"liquidityGross\":11381886856279044937,\"liquidityNet\":-10788226896545618205},{\"index\":-27950,\"liquidityGross\":695999526056770141,\"liquidityNet\":-695999526056770141},{\"index\":-27800,\"liquidityGross\":41637414711935053804,\"liquidityNet\":-41637414711935053804},{\"index\":-27700,\"liquidityGross\":7263268694521872073,\"liquidityNet\":-6912099949529570661},{\"index\":-27650,\"liquidityGross\":73952794275826312820,\"liquidityNet\":34526291653644184646},{\"index\":-27600,\"liquidityGross\":23265828222207870133,\"liquidityNet\":-23265828222207870133},{\"index\":-27550,\"liquidityGross\":184009368390604568,\"liquidityNet\":118771493595272744},{\"index\":-27500,\"liquidityGross\":1727433476583725786,\"liquidityNet\":1507180301662519088},{\"index\":-27450,\"liquidityGross\":158973834162238644,\"liquidityNet\":-158973834162238644},{\"index\":-27400,\"liquidityGross\":23513514108007271558,\"liquidityNet\":23513514108007271558},{\"index\":-27350,\"liquidityGross\":55011004654890060589,\"liquidityNet\":-55011004654890060589},{\"index\":-27250,\"liquidityGross\":37233797031505356076,\"liquidityNet\":-36881935449482623072},{\"index\":-27200,\"liquidityGross\":253890349171053360589,\"liquidityNet\":-253890349171053360589},{\"index\":-27150,\"liquidityGross\":175930791011366502,\"liquidityNet\":-175930791011366502},{\"index\":-27100,\"liquidityGross\":150760211472637267,\"liquidityNet\":-150760211472637267},{\"index\":-27050,\"liquidityGross\":327505352104042813,\"liquidityNet\":-327505352104042813},{\"index\":-27000,\"liquidityGross\":325586113732920274371,\"liquidityNet\":325501269381596863097},{\"index\":-26950,\"liquidityGross\":2312639040743774037,\"liquidityNet\":-2312639040743774037},{\"index\":-26900,\"liquidityGross\":46345514464890381038,\"liquidityNet\":-46345514464890381038},{\"index\":-26850,\"liquidityGross\":995356704226199,\"liquidityNet\":-995356704226199},{\"index\":-26700,\"liquidityGross\":213120534271548,\"liquidityNet\":-213120534271548},{\"index\":-26650,\"liquidityGross\":165399324414424110,\"liquidityNet\":-165399324414424110},{\"index\":-26600,\"liquidityGross\":325567701076744503656,\"liquidityNet\":-325567701076744503656},{\"index\":-26550,\"liquidityGross\":23516622785184778611,\"liquidityNet\":-23516622785184778611},{\"index\":-26500,\"liquidityGross\":1744274794511845110,\"liquidityNet\":-1744274794511845110},{\"index\":-26400,\"liquidityGross\":185579852400283114,\"liquidityNet\":-185579852400283114},{\"index\":-26250,\"liquidityGross\":7919545458431110614,\"liquidityNet\":-7919545458431110614},{\"index\":-26200,\"liquidityGross\":131266962904863738,\"liquidityNet\":-131266962904863738},{\"index\":-26150,\"liquidityGross\":3033250330002482,\"liquidityNet\":-3033250330002482},{\"index\":-26000,\"liquidityGross\":751504450314768756,\"liquidityNet\":-751504450314768756},{\"index\":-25900,\"liquidityGross\":2451346885044458144,\"liquidityNet\":-2451346885044458144},{\"index\":-25850,\"liquidityGross\":38271016446769585,\"liquidityNet\":-38271016446769585},{\"index\":-25750,\"liquidityGross\":1217688492803422024,\"liquidityNet\":-1217688492803422024},{\"index\":-25650,\"liquidityGross\":1265182246674609833,\"liquidityNet\":-1265182246674609833},{\"index\":-25600,\"liquidityGross\":175584372496150706,\"liquidityNet\":-175584372496150706},{\"index\":-25500,\"liquidityGross\":643159108816570223,\"liquidityNet\":-643159108816570223},{\"index\":-25450,\"liquidityGross\":23659350773676994,\"liquidityNet\":-23659350773676994},{\"index\":-25400,\"liquidityGross\":1070403859429138319,\"liquidityNet\":-1070403859429138319},{\"index\":-25350,\"liquidityGross\":6082815561217779912,\"liquidityNet\":-6082815561217779912},{\"index\":-25300,\"liquidityGross\":212473941643375311,\"liquidityNet\":-212473941643375311},{\"index\":-25200,\"liquidityGross\":1508876918856122135,\"liquidityNet\":-1508876918856122135},{\"index\":-24750,\"liquidityGross\":160285217544872870,\"liquidityNet\":-160285217544872870},{\"index\":-24700,\"liquidityGross\":654003707535727983,\"liquidityNet\":-654003707535727983},{\"index\":-24600,\"liquidityGross\":273556939280535818,\"liquidityNet\":-273556939280535818},{\"index\":-24500,\"liquidityGross\":2010949834182953415,\"liquidityNet\":-2010949834182953415},{\"index\":-24250,\"liquidityGross\":715605637914733404,\"liquidityNet\":-715605637914733404},{\"index\":-24200,\"liquidityGross\":5143776120734644,\"liquidityNet\":-5143776120734644},{\"index\":-24100,\"liquidityGross\":376984490216644370,\"liquidityNet\":-376984490216644370},{\"index\":-23850,\"liquidityGross\":259827097639630760,\"liquidityNet\":-259827097639630760},{\"index\":-23550,\"liquidityGross\":562649739463143807,\"liquidityNet\":-562649739463143807},{\"index\":-23500,\"liquidityGross\":33075182887457925,\"liquidityNet\":-33075182887457925},{\"index\":-23450,\"liquidityGross\":428112238292409455,\"liquidityNet\":-428112238292409455},{\"index\":-23400,\"liquidityGross\":1958557830789732,\"liquidityNet\":-1958557830789732},{\"index\":-23250,\"liquidityGross\":904982620118830741,\"liquidityNet\":-904982620118830741},{\"index\":-23200,\"liquidityGross\":2640528875938648607,\"liquidityNet\":-2640528875938648607},{\"index\":-23100,\"liquidityGross\":4846998876633794747,\"liquidityNet\":-4846998876633794747},{\"index\":-23050,\"liquidityGross\":906960911607084363,\"liquidityNet\":-906960911607084363},{\"index\":-22950,\"liquidityGross\":2129821827137791908,\"liquidityNet\":-2129821827137791908},{\"index\":-22750,\"liquidityGross\":33540501595797729,\"liquidityNet\":-33540501595797729},{\"index\":-22650,\"liquidityGross\":387185612947455516,\"liquidityNet\":-387185612947455516},{\"index\":-22600,\"liquidityGross\":5376814869087283,\"liquidityNet\":-5376814869087283},{\"index\":-22550,\"liquidityGross\":4869617926820515122,\"liquidityNet\":-4869617926820515122},{\"index\":-22250,\"liquidityGross\":119574160305151552158,\"liquidityNet\":-119574160305151552158},{\"index\":-22150,\"liquidityGross\":2963316903385571661,\"liquidityNet\":-2963316903385571661},{\"index\":-22100,\"liquidityGross\":3204079322304961387,\"liquidityNet\":-3204079322304961387},{\"index\":-22050,\"liquidityGross\":6337357898841103438,\"liquidityNet\":-6337357898841103438},{\"index\":-22000,\"liquidityGross\":4271310345033175927,\"liquidityNet\":-4271310345033175927},{\"index\":-21950,\"liquidityGross\":4964332238759786,\"liquidityNet\":-4964332238759786},{\"index\":-21900,\"liquidityGross\":10843505163488926545,\"liquidityNet\":-10843505163488926545},{\"index\":-21850,\"liquidityGross\":8156518337731,\"liquidityNet\":-8156518337731},{\"index\":-21800,\"liquidityGross\":1495680014348672304,\"liquidityNet\":-1495680014348672304},{\"index\":-21750,\"liquidityGross\":8014126668950211424,\"liquidityNet\":-8014126668950211424},{\"index\":-21650,\"liquidityGross\":143658667301642849,\"liquidityNet\":-143658667301642849},{\"index\":-21500,\"liquidityGross\":8573579599102728928,\"liquidityNet\":-8573579599102728928},{\"index\":-21400,\"liquidityGross\":47634334996056764329,\"liquidityNet\":-47634334996056764329},{\"index\":-21250,\"liquidityGross\":13324106967880548,\"liquidityNet\":-13324106967880548},{\"index\":-21150,\"liquidityGross\":112618319456361721,\"liquidityNet\":-112618319456361721},{\"index\":-21000,\"liquidityGross\":1560599854262717957,\"liquidityNet\":-1560599854262717957},{\"index\":-20950,\"liquidityGross\":713679699198812956,\"liquidityNet\":-713679699198812956},{\"index\":-20900,\"liquidityGross\":369328196340005770,\"liquidityNet\":-369328196340005770},{\"index\":-20800,\"liquidityGross\":10805805305857734421,\"liquidityNet\":-10805805305857734421},{\"index\":-20750,\"liquidityGross\":1214826249023199602,\"liquidityNet\":-1214826249023199602},{\"index\":-20700,\"liquidityGross\":1535190057107029446,\"liquidityNet\":-1535190057107029446},{\"index\":-20650,\"liquidityGross\":334751012055967852,\"liquidityNet\":-334751012055967852},{\"index\":-20550,\"liquidityGross\":8314466382584420068,\"liquidityNet\":-8314466382584420068},{\"index\":-20450,\"liquidityGross\":42865368055396611,\"liquidityNet\":-42865368055396611},{\"index\":-20400,\"liquidityGross\":3603440472259039738,\"liquidityNet\":-3603440472259039738},{\"index\":-20350,\"liquidityGross\":190083310699919826,\"liquidityNet\":-190083310699919826},{\"index\":-20300,\"liquidityGross\":20314140289507416,\"liquidityNet\":-20314140289507416},{\"index\":-20250,\"liquidityGross\":147970774442032313,\"liquidityNet\":-147970774442032313},{\"index\":-20150,\"liquidityGross\":958591259756037231,\"liquidityNet\":-958591259756037231},{\"index\":-20100,\"liquidityGross\":6349266909280116288,\"liquidityNet\":-6349266909280116288},{\"index\":-20050,\"liquidityGross\":29667790786063010,\"liquidityNet\":-29667790786063010},{\"index\":-20000,\"liquidityGross\":2474299609856413411,\"liquidityNet\":-2474299609856413411},{\"index\":-19950,\"liquidityGross\":56285484178377411775,\"liquidityNet\":-56285484178377411775},{\"index\":-19900,\"liquidityGross\":116317722897312231,\"liquidityNet\":-116317722897312231},{\"index\":-19800,\"liquidityGross\":1141820532929424407,\"liquidityNet\":-1141820532929424407},{\"index\":-19700,\"liquidityGross\":19278787910878833969,\"liquidityNet\":-19278787910878833969},{\"index\":-19650,\"liquidityGross\":4609904055462072,\"liquidityNet\":-4609904055462072},{\"index\":-19350,\"liquidityGross\":69017329775279556,\"liquidityNet\":-69017329775279556},{\"index\":-17750,\"liquidityGross\":26719985602880751,\"liquidityNet\":-26719985602880751},{\"index\":-17400,\"liquidityGross\":79685265646310494,\"liquidityNet\":-79685265646310494},{\"index\":-15100,\"liquidityGross\":108429172951543,\"liquidityNet\":-108429172951543},{\"index\":138150,\"liquidityGross\":14808467822101239,\"liquidityNet\":-14808467822101239},{\"index\":887250,\"liquidityGross\":66053528479507347108,\"liquidityNet\":-66053528479507347108}]}",
  "staticExtra": "{\"poolId\":\"0xd4dca84e1808da3354924cd243c66828cf775470\"}"
}`

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	type testcase struct {
		name     string
		tokenIn  string
		amountIn string
		tokenOut string
	}
	testcases := []testcase{
		{
			name:     "swap ETH for BTCB",
			tokenIn:  "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
			amountIn: "1000000000000000000", // 1
			tokenOut: "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulatorBigInt(*poolEntity, valueobject.ChainIDEthereum)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})
			require.NoError(t, err)
			_ = result
		})

		t.Run(tc.name+"new sim", func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})
			require.NoError(t, err)
			_ = result
		})
	}
}

func TestComparePoolSimulatorV2(t *testing.T) {
	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(poolEncoded), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulatorBigInt(*poolEntity, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	poolSimV2, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	for i := 0; i < 500; i++ {
		amt := RandNumberString(24)

		t.Run(fmt.Sprintf("test %s ETH -> BTCB %d", amt, i), func(t *testing.T) {
			in := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
					Amount: bignumber.NewBig10(amt),
				},
				TokenOut: "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
			}
			result, err := poolSim.CalcAmountOut(in)
			resultV2, errV2 := poolSimV2.CalcAmountOut(in)

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountOut, resultV2.TokenAmountOut)
				assert.Equal(t, result.Fee, resultV2.Fee)
				assert.Equal(t, result.RemainingTokenAmountIn.Amount.String(), resultV2.RemainingTokenAmountIn.Amount.String())

				poolSim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				})
				poolSimV2.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *resultV2.TokenAmountOut,
					Fee:            *resultV2.Fee,
					SwapInfo:       resultV2.SwapInfo,
				})
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s ETH -> BTCB (reversed) %d", amt, i), func(t *testing.T) {
			result, err := poolSim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
				Limit:   nil,
			})

			resultV2, errV2 := poolSimV2.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
				Limit:   nil,
			})

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountIn.Amount, resultV2.TokenAmountIn.Amount)
				assert.Equal(t, result.Fee.Amount, resultV2.Fee.Amount)
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s BTCB -> ETH %d", amt, i), func(t *testing.T) {
			in := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
					Amount: bignumber.NewBig10(amt),
				},
				TokenOut: "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
			}
			result, err := poolSim.CalcAmountOut(in)
			resultV2, errV2 := poolSimV2.CalcAmountOut(in)

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountOut, resultV2.TokenAmountOut)
				assert.Equal(t, result.Fee, resultV2.Fee)
				assert.Equal(t, result.RemainingTokenAmountIn.Amount.String(), resultV2.RemainingTokenAmountIn.Amount.String())

				poolSim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				})
				poolSimV2.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *resultV2.TokenAmountOut,
					Fee:            *resultV2.Fee,
					SwapInfo:       resultV2.SwapInfo,
				})
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s BTCB -> ETH (reversed) %d", amt, i), func(t *testing.T) {
			result, err := poolSim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
				Limit:   nil,
			})
			resultV2, errV2 := poolSimV2.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
				Limit:   nil,
			})

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountIn.Amount, resultV2.TokenAmountIn.Amount)
				assert.Equal(t, result.Fee.Amount, resultV2.Fee.Amount)
			} else {
				fmt.Println(err)
			}
		})
	}
}

// not really random but should be enough for testing
func RandNumberString(maxLen int) string {
	sLen := rand.Intn(maxLen-1) + 1
	var s string
	for i := 0; i < sLen; i++ {
		var c int
		if i == 0 {
			c = rand.Intn(9) + 1
		} else {
			c = rand.Intn(10)
		}
		s = fmt.Sprintf("%s%d", s, c)
	}
	return s
}
