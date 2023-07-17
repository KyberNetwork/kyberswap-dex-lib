package clientid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const testXClientID = "test-client-id"

func emptySuccessResponse(c *gin.Context) {
	c.String(http.StatusOK, "")
}

func Test_RequestID_CreateNew(t *testing.T) {
	r := gin.New()
	r.Use(New())
	r.GET("/", emptySuccessResponse)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get(headerXClientID))
}

func Test_RequestID_PassThru(t *testing.T) {
	r := gin.New()
	r.Use(New())
	r.GET("/", emptySuccessResponse)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set(headerXClientID, testXClientID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, testXClientID, w.Header().Get(headerXClientID))
}

func TestClientIDWithCustomHeaderKey(t *testing.T) {
	r := gin.New()
	r.Use(
		New(
			WithCustomHeaderStrKey("customKey"),
		),
	)
	r.GET("/", emptySuccessResponse)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("customKey", testXClientID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, testXClientID, w.Header().Get("customKey"))
}

func TestClientIDWithHandler(t *testing.T) {
	r := gin.New()
	called := false
	r.Use(
		New(
			WithHandler(func(c *gin.Context, requestID string) {
				called = true
				assert.Equal(t, testXClientID, requestID)
			}),
		),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("X-Client-ID", testXClientID)
	r.ServeHTTP(w, req)

	assert.True(t, called)
}
