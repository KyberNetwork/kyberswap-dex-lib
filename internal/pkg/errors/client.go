package errors

import (
	"net/http"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"

	"github.com/KyberNetwork/kyberswap-error/pkg/errors"
)

func NewRestAPIErrTokensAreIdentical(rootCause error, entities ...string) *errors.RestAPIError {
	return errors.NewRestAPIError(
		http.StatusBadRequest,
		constant.ClientErrCodeTokensAreIdentical,
		constant.ClientErrMsgTokensAreIdentical,
		entities,
		rootCause,
	)
}

func NewRestAPIErrDeadlineIsInThePast(rootCause error, entities ...string) *errors.RestAPIError {
	return errors.NewRestAPIError(
		http.StatusBadRequest,
		constant.ClientErrCodeDeadlineIsInThePast,
		constant.ClientErrMsgDeadlineIsInThePast,
		entities,
		rootCause,
	)
}

func NewRestAPIErrCouldNotFindRoute(rootCause error) *errors.RestAPIError {
	return errors.NewRestAPIError(
		http.StatusInternalServerError,
		constant.ClientErrCodeCouldNotFindRoute,
		constant.ClientErrMsgCouldNotFindRoute,
		nil,
		rootCause,
	)
}
