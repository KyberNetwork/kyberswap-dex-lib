package platypus

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

var GetPoolAddressesQueryTemplate = `{pools (first: {{.First}} skip: {{.Skip}}) { id }}`

type PoolSubgraphReaderConfig struct {
	GetPoolAddressesBulk int
}

type PoolSubgraphReader struct {
	client *graphql.Client
	config PoolSubgraphReaderConfig
}

func NewPoolSubgraphReader(
	endpoint string,
	config PoolSubgraphReaderConfig,
) *PoolSubgraphReader {
	return &PoolSubgraphReader{
		client: graphql.NewClient(endpoint),
		config: config,
	}
}

func (r *PoolSubgraphReader) GetPoolAddresses(
	ctx context.Context,
) ([]string, error) {
	queryTemplate, err := template.New("getPoolAddressesQueryTemplate").
		Parse(GetPoolAddressesQueryTemplate)
	if err != nil {
		return nil, err
	}

	var poolAddresses []string
	var runCount int

	for {
		param := GetPoolsQueryParam{
			First: r.config.GetPoolAddressesBulk,
			Skip:  runCount * r.config.GetPoolAddressesBulk,
		}

		var query bytes.Buffer

		if err = queryTemplate.Execute(&query, param); err != nil {
			return nil, err
		}

		req := graphql.NewRequest(query.String())

		var resp GetPoolsResponse

		if err = r.client.Run(ctx, req, &resp); err != nil {
			return nil, err
		}

		if len(resp.Pools) == 0 {
			break
		}

		for _, pool := range resp.Pools {
			if strings.EqualFold(pool.Id, constant.AddressZero) {
				continue
			}

			poolAddresses = append(poolAddresses, pool.Id)
		}

		runCount += 1
	}

	return poolAddresses, nil
}
