package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var ErrorResponseByError = map[error]ErrorResponse{
	ErrBindQueryFailed: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4001,
		Message:    "unable to bind query parameters",
	},

	ErrBindRequestBodyFailed: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4002,
		Message:    "unable to bind request body",
	},

	ErrInvalidRoute: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4003,
		Message:    "invalid route",
	},

	ErrInvalidValue: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4004,
		Message:    "invalid value",
	},

	ErrFeeAmountGreaterThanAmountIn: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4005,
		Message:    "feeAmount is greater than amountIn",
	},

	ErrFeeAmountGreaterThanAmountOut: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4006,
		Message:    "feeAmount is greater than amountOut",
	},

	usecase.ErrFeeAmountIsGreaterThanAmountOut: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4007,
		Message:    "feeAmount is greater than amountOut",
	},

	usecase.ErrRouteNotFound: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4008,
		Message:    "route not found",
	},

	usecase.ErrAmountInIsGreaterThanMaxAllowed: {
		HTTPStatus: http.StatusBadRequest,
		Code:       4009,
		Message:    "amountIn is greater than max allowed",
	},

	usecase.ErrPublicKeyNotFound: {
		HTTPStatus: http.StatusNotFound,
		Code:       4040,
		Message:    "public key can not be found",
	},

	eth.ErrWETHNotFound: {
		HTTPStatus: http.StatusUnprocessableEntity,
		Code:       4221,
		Message:    "weth not found",
	},
}

type SuccessResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"requestId"`
}

type ErrorResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details"`

	HTTPStatus int    `json:"-"`
	RequestID  string `json:"requestId"`
}

type DetailsBadRequest struct {
	FieldViolations []*DetailBadRequestFieldViolation `json:"fieldViolations"`
}

type DetailBadRequestFieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
}

var DefaultErrorResponse = ErrorResponse{
	HTTPStatus: http.StatusInternalServerError,
	Code:       500,
	Message:    "internal server error",
}

func RespondSuccess(c *gin.Context, data interface{}) {
	successResponse := SuccessResponse{
		Code:      0,
		Message:   "successfully",
		Data:      data,
		RequestID: requestid.ExtractRequestID(c),
	}

	c.JSON(
		http.StatusOK,
		successResponse,
	)
}

func RespondFailure(c *gin.Context, err error) {
	var validationErr *validator.ValidationError
	if errors.As(err, &validationErr) {
		respondValidationError(c, validationErr)
		return
	}

	requestID := requestid.ExtractRequestID(c)
	response := responseFromErr(err)
	response.RequestID = requestID

	logger.
		WithFields(logger.Fields{"request.id": requestID, "error": err}).
		Warn("respond failure")

	c.JSON(
		response.HTTPStatus,
		response,
	)
}

func responseFromErr(err error) ErrorResponse {
	for {
		if err == nil {
			return DefaultErrorResponse
		}

		if resp, ok := ErrorResponseByError[err]; ok {
			return resp
		}

		err = errors.Unwrap(err)
	}
}

func respondValidationError(c *gin.Context, err *validator.ValidationError) {

	errorResponse := ErrorResponse{
		Code:    4000,
		Message: "bad request",
		Details: []interface{}{
			&DetailsBadRequest{
				FieldViolations: []*DetailBadRequestFieldViolation{
					{
						Field:       err.Field,
						Description: err.Description,
					},
				},
			},
		},
		RequestID: requestid.ExtractRequestID(c),
	}

	c.JSON(
		http.StatusBadRequest,
		errorResponse,
	)
}
