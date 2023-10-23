package liquiditybookv21

import (
	"fmt"
	"math/big"
)

func transformSubgraphBins(
	subgraphBins []binSubgraphResp,
	unitX *big.Float,
	unitY *big.Float,
) ([]bin, error) {
	ret := make([]bin, 0, len(subgraphBins))
	for _, b := range subgraphBins {
		id, ok := new(big.Int).SetString(b.BinID, 10)
		if !ok {
			return nil, ErrInvalidBinID
		}

		reserveX, ok := new(big.Float).SetString(b.ReserveX)
		if !ok {
			return nil, ErrInvalidReserve
		}
		reserveXInt, _ := new(big.Float).Mul(reserveX, unitX).Int(nil)

		reserveY, ok := new(big.Float).SetString(b.ReserveY)
		if !ok {
			return nil, ErrInvalidReserve
		}
		reserveYInt, _ := new(big.Float).Mul(reserveY, unitY).Int(nil)

		ret = append(ret, bin{
			ID:       id,
			ReserveX: reserveXInt,
			ReserveY: reserveYInt,
		})
	}

	return ret, nil
}

func buildQueryGetBins(pairAddress string, skip int) string {
	q := fmt.Sprintf(`
	lbpair(id: "%s") {
		tokenX { decimals }
		tokenY { decimals }
		bins(orderBy: binId, orderDirection: asc, first: 1000, skip: %d) {
		  binId
		  reserveX
		  reserveY
		}
	  }
	  _meta { block { timestamp } }
	`, pairAddress, skip)

	return q
}
