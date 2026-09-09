package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResetGroupQuota_RejectsInvalidRequests(t *testing.T) {
	for _, tc := range []struct{ id, body string }{
		{"0", `{"daily":true}`},
		{"-1", `{"daily":true}`},
		{"invalid", `{"daily":true}`},
		{"7", `{}`},
		{"7", `{"daily":false,"weekly":false,"monthly":false}`},
		{"7", `{"weekly":"yes"}`},
	} {
		t.Run(tc.id+tc.body, func(t *testing.T) {
			router := gin.New()
			h := NewSubscriptionHandler(nil)
			router.POST("/groups/:id/subscriptions/reset-quota", h.ResetGroupQuota)
			req := httptest.NewRequest(http.MethodPost, "/groups/"+tc.id+"/subscriptions/reset-quota", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}
