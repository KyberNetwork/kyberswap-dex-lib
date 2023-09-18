package iziswap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
)

// params="chain_id=324&type=10&version=v2&time_start=2023-06-02T13:53:13&page=1&page_size=10&order_by=time"
var urlRoot = "https://api.izumi.finance/api/v1/izi_swap/meta_record"

type PoolsListQueryParams struct {
	chainId int
	// v1 or v2
	version string
	// timestamp in second
	timeStart int
	// response size
	limit int
}

type PoolInfo struct {
	Fee            int    `json:"fee"`
	TokenX         string `json:"tokenX"`
	TokenY         string `json:"tokenY"`
	Address        string `json:"address"`
	Timestamp      int    `json:"timestamp"`
	TokenXAddress  string `json:"tokenX_address"`
	TokenYAddress  string `json:"tokenY_address"`
	TokenXDecimals int    `json:"tokenX_decimals"`
	TokenYDecimals int    `json:"tokenY_decimals"`
	Version        string `json:"version"`
}

type PoolsListQueryResponse struct {
	Data  []PoolInfo `json:"data,omitempty"`
	Total int        `json:"total"`
}

func getPoolsList(
	ctx context.Context,
	client *http.Client,
	params *PoolsListQueryParams,
) ([]PoolInfo, error) {
	req, err := http.NewRequest(http.MethodGet, urlRoot, nil)
	if err != nil {
		return nil, err
	}

	limit := params.limit
	if limit < 0 || limit > POOL_LIST_LIMIT {
		limit = POOL_LIST_LIMIT
	}

	q := req.URL.Query()
	q.Add("chain_id", strconv.Itoa(params.chainId))
	q.Add("type", POOL_TYPE_VALUE)
	q.Add("version", params.version)
	q.Add("time_start", time.Unix(int64(params.timeStart), 0).Format("2006-01-02T15:04:05"))
	q.Add("format", "json")
	q.Add("order_by", "time")
	q.Add("page_size", strconv.Itoa(limit))

	req.URL.RawQuery = q.Encode()

	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var response PoolsListQueryResponse
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}
