package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

type HTTPTestCase struct {
	ReqMethod         string
	ReqURL            string
	ReqParams         url.Values
	ReqBody           io.Reader
	PathIncludeParams string
	ReqHandler        gin.HandlerFunc

	RespHTTPStatus int
	RespBody       interface{}
}

func (tc *HTTPTestCase) setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	if len(tc.PathIncludeParams) > 0 {
		router.Handle(tc.ReqMethod, tc.PathIncludeParams, tc.ReqHandler)
	} else {
		router.Handle(tc.ReqMethod, tc.ReqURL, tc.ReqHandler)
	}

	return router
}

func (tc *HTTPTestCase) Run(t *testing.T) {
	router := tc.setupRouter()

	w := httptest.NewRecorder()

	req, _ := http.NewRequest(
		tc.ReqMethod,
		tc.ReqURL,
		tc.ReqBody,
	)
	req.URL.RawQuery = tc.ReqParams.Encode()
	router.ServeHTTP(w, req)

	respBytes, err := json.Marshal(tc.RespBody)
	if err != nil {
		t.Fatal("failed to marshal expected response" + err.Error())
	}

	assert.Equal(t, tc.RespHTTPStatus, w.Code)
	assert.EqualValues(t, string(respBytes), w.Body.String())
}
