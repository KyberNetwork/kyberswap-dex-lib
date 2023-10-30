package liquiditybookv20

import (
	"fmt"
	"math/big"
	"strconv"
)

func transformSubgraphBins(
	subgraphBins []binSubgraphResp,
	unitX *big.Float,
	unitY *big.Float,
) ([]bin, error) {
	ret := make([]bin, 0, len(subgraphBins))
	for _, b := range subgraphBins {
		id, err := strconv.ParseUint(b.BinID, 10, 32)
		if err != nil {
			return nil, err
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

		totalSupply, _ := new(big.Int).SetString(b.TotalSupply, 10)

		ret = append(ret, bin{
			ID:          uint32(id),
			ReserveX:    reserveXInt,
			ReserveY:    reserveYInt,
			TotalSupply: totalSupply,
		})
	}

	return ret, nil
}

func buildQueryGetBins(pairAddress string, binIDGT int64) string {
	q := fmt.Sprintf(`{
	lbpair(id: "%s") {
		tokenX { decimals }
		tokenY { decimals }
		bins(where: {totalSupply_not: "0", binId_gt: "%d"}, orderBy: binId, orderDirection: asc, first: %d) {
		  binId
		  reserveX
		  reserveY
		  totalSupply
		}
	}
	_meta { block { timestamp } }
	}`, pairAddress, binIDGT, graphFirstLimit)

	return q
}
