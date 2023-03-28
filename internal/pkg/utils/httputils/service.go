package httputils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"context"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

type response struct {
	Code    int             `json:"response_code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func Process(ctx context.Context, req *http.Request, dest interface{}) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf("failed to post request, err: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("failed to read response body, err: %v", err)
		return err
	}
	logger.Debugf("price http response: %v", string(respBody))

	var r response
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		logger.Errorf("failed to unmarshal response body, err: %v", err)
		return err
	}

	if r.Code != http.StatusOK {
		return fmt.Errorf("response code: %d", r.Code)
	}

	err = json.Unmarshal(r.Data, dest)
	if err != nil {
		logger.Errorf("failed to unmarshal response data, err: %v", err)
		return err
	}

	return nil
}
