package liquiditybookv20

import (
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	rawPools := []string{
		`{
			"address": "0x18332988456c4bd9aba6698ec748b331516f5a14",
			"reserveUsd": 37820.100016332304,
			"exchange": "traderjoe-v20",
			"type": "liquiditybook-v20",
			"timestamp": 1705345192,
			"reserves": [
				"6797571623",
				"31062309407"
			],
			"tokens": [
				{
					"address": "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					"weight": 50,
					"swappable": true
				},
				{
					"address": "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
					"weight": 50,
					"swappable": true
				}
			],
			"extra": "{\"rpcBlockTimestamp\":1705345186,\"subgraphBlockTimestamp\":1705345184,\"feeParameters\":{\"binStep\":1,\"baseFactor\":20000,\"filterPeriod\":10,\"decayPeriod\":120,\"reductionFactor\":5000,\"variableFeeControl\":2000000,\"protocolShare\":0,\"maxVolatilityAccumulated\":100000,\"volatilityAccumulated\":2500,\"volatilityReference\":2500,\"indexRef\":8388610,\"time\":1705344799},\"activeBinId\":8388610,\"bins\":[{\"id\":8388508,\"reserveX\":0,\"reserveY\":1999,\"totalSupply\":2000},{\"id\":8388509,\"reserveX\":0,\"reserveY\":10,\"totalSupply\":10},{\"id\":8388516,\"reserveX\":0,\"reserveY\":999,\"totalSupply\":1000},{\"id\":8388527,\"reserveX\":0,\"reserveY\":1500,\"totalSupply\":1500},{\"id\":8388541,\"reserveX\":0,\"reserveY\":100,\"totalSupply\":100},{\"id\":8388559,\"reserveX\":0,\"reserveY\":1,\"totalSupply\":1},{\"id\":8388561,\"reserveX\":0,\"reserveY\":100,\"totalSupply\":100},{\"id\":8388573,\"reserveX\":0,\"reserveY\":100,\"totalSupply\":100},{\"id\":8388580,\"reserveX\":0,\"reserveY\":100000,\"totalSupply\":100000},{\"id\":8388581,\"reserveX\":0,\"reserveY\":103528,\"totalSupply\":103528},{\"id\":8388582,\"reserveX\":0,\"reserveY\":123547,\"totalSupply\":123547},{\"id\":8388583,\"reserveX\":0,\"reserveY\":2421428,\"totalSupply\":2421428},{\"id\":8388584,\"reserveX\":0,\"reserveY\":1481762,\"totalSupply\":1481762},{\"id\":8388585,\"reserveX\":0,\"reserveY\":2482197,\"totalSupply\":2482196},{\"id\":8388586,\"reserveX\":0,\"reserveY\":1510037,\"totalSupply\":1510036},{\"id\":8388587,\"reserveX\":0,\"reserveY\":2496177,\"totalSupply\":2496175},{\"id\":8388588,\"reserveX\":0,\"reserveY\":1495928,\"totalSupply\":1495926},{\"id\":8388589,\"reserveX\":0,\"reserveY\":1545936,\"totalSupply\":1545934},{\"id\":8388590,\"reserveX\":0,\"reserveY\":1557360,\"totalSupply\":1557358},{\"id\":8388591,\"reserveX\":0,\"reserveY\":1594342,\"totalSupply\":1594339},{\"id\":8388592,\"reserveX\":0,\"reserveY\":1949061,\"totalSupply\":1949056},{\"id\":8388593,\"reserveX\":0,\"reserveY\":8666515,\"totalSupply\":8666503},{\"id\":8388594,\"reserveX\":0,\"reserveY\":10481618,\"totalSupply\":10481600},{\"id\":8388595,\"reserveX\":0,\"reserveY\":12260742,\"totalSupply\":12260711},{\"id\":8388596,\"reserveX\":0,\"reserveY\":14057001,\"totalSupply\":14056966},{\"id\":8388597,\"reserveX\":0,\"reserveY\":16390596,\"totalSupply\":16390541},{\"id\":8388598,\"reserveX\":0,\"reserveY\":26517213,\"totalSupply\":26517140},{\"id\":8388599,\"reserveX\":0,\"reserveY\":28907754,\"totalSupply\":28907658},{\"id\":8388600,\"reserveX\":0,\"reserveY\":34720508,\"totalSupply\":34720382},{\"id\":8388601,\"reserveX\":0,\"reserveY\":407699018,\"totalSupply\":407698705},{\"id\":8388602,\"reserveX\":0,\"reserveY\":431800496,\"totalSupply\":431800111},{\"id\":8388603,\"reserveX\":0,\"reserveY\":1561105990,\"totalSupply\":1561105652},{\"id\":8388604,\"reserveX\":0,\"reserveY\":1798224572,\"totalSupply\":1798224299},{\"id\":8388605,\"reserveX\":0,\"reserveY\":2453644158,\"totalSupply\":2453643724},{\"id\":8388606,\"reserveX\":0,\"reserveY\":2881745276,\"totalSupply\":2881744448},{\"id\":8388607,\"reserveX\":0,\"reserveY\":4349035785,\"totalSupply\":4349034480},{\"id\":8388608,\"reserveX\":0,\"reserveY\":12746248394,\"totalSupply\":12746248383},{\"id\":8388609,\"reserveX\":0,\"reserveY\":4081379342,\"totalSupply\":4081378651},{\"id\":8388610,\"reserveX\":2450363891,\"reserveY\":180558315,\"totalSupply\":2631410640},{\"id\":8388611,\"reserveX\":2196518621,\"reserveY\":0,\"totalSupply\":2197177171},{\"id\":8388612,\"reserveX\":968153988,\"reserveY\":0,\"totalSupply\":968541108},{\"id\":8388613,\"reserveX\":938474633,\"reserveY\":0,\"totalSupply\":938943605},{\"id\":8388614,\"reserveX\":45851114,\"reserveY\":0,\"totalSupply\":45878578},{\"id\":8388615,\"reserveX\":38995156,\"reserveY\":0,\"totalSupply\":39022395},{\"id\":8388616,\"reserveX\":32943540,\"reserveY\":0,\"totalSupply\":32969837},{\"id\":8388617,\"reserveX\":27707684,\"reserveY\":0,\"totalSupply\":27732578},{\"id\":8388618,\"reserveX\":24592239,\"reserveY\":0,\"totalSupply\":24616794},{\"id\":8388619,\"reserveX\":15657104,\"reserveY\":0,\"totalSupply\":15674296},{\"id\":8388620,\"reserveX\":13769541,\"reserveY\":0,\"totalSupply\":13786034},{\"id\":8388621,\"reserveX\":11992950,\"reserveY\":0,\"totalSupply\":12008513},{\"id\":8388622,\"reserveX\":10181045,\"reserveY\":0,\"totalSupply\":10195272},{\"id\":8388623,\"reserveX\":8064829,\"reserveY\":0,\"totalSupply\":8076909},{\"id\":8388624,\"reserveX\":1708368,\"reserveY\":0,\"totalSupply\":1711099},{\"id\":8388625,\"reserveX\":1582991,\"reserveY\":0,\"totalSupply\":1585680},{\"id\":8388626,\"reserveX\":1524353,\"reserveY\":0,\"totalSupply\":1527094},{\"id\":8388627,\"reserveX\":1355889,\"reserveY\":0,\"totalSupply\":1358463},{\"id\":8388628,\"reserveX\":1355883,\"reserveY\":0,\"totalSupply\":1358593},{\"id\":8388629,\"reserveX\":1365898,\"reserveY\":0,\"totalSupply\":1368765},{\"id\":8388630,\"reserveX\":1344401,\"reserveY\":0,\"totalSupply\":1347359},{\"id\":8388631,\"reserveX\":1338644,\"reserveY\":0,\"totalSupply\":1341724},{\"id\":8388632,\"reserveX\":1337360,\"reserveY\":0,\"totalSupply\":1340571},{\"id\":8388633,\"reserveX\":1378629,\"reserveY\":0,\"totalSupply\":1382078},{\"id\":8388634,\"reserveX\":3920,\"reserveY\":0,\"totalSupply\":3930},{\"id\":8388635,\"reserveX\":3920,\"reserveY\":0,\"totalSupply\":3930},{\"id\":8388636,\"reserveX\":3920,\"reserveY\":0,\"totalSupply\":3930},{\"id\":8388637,\"reserveX\":2,\"reserveY\":0,\"totalSupply\":2},{\"id\":8388640,\"reserveX\":10,\"reserveY\":0,\"totalSupply\":10},{\"id\":8388653,\"reserveX\":999,\"reserveY\":0,\"totalSupply\":1004},{\"id\":8388667,\"reserveX\":100,\"reserveY\":0,\"totalSupply\":100}]}"
		}`,
	}
	poolEntities := make([]entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		err := json.Unmarshal([]byte(rawPool), &poolEntities[i])
		require.NoError(t, err)
	}
	var err error
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(poolEntity)
		require.NoError(t, err)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolSimulator{})...))
	}
}
