//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

func TestCarpoolModeGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restricted := []string{
		"/payment/orders", "/payment/webhook/stripe", "/admin/payment/config",
		"/redeem", "/admin/redeem-codes/create-and-redeem",
		"/admin/promo-codes", "/admin/affiliates/users", "/user/aff/transfer",
		"/pages", "/model-plaza", "/auth/register", "/v1/images/batches",
		"/auth/validate-promo-code", "/auth/oauth/wechat/payment/callback",
	}
	allowed := []string{
		"/admin/users", "/admin/groups/1/subscriptions/reset-usage",
		"/admin/subscriptions", "/subscriptions", "/keys", "/usage",
		"/settings", "/auth/validate-invitation-code", "/payment-extra",
	}
	for _, mode := range []string{config.RunModeStandard, config.RunModeSimple, config.RunModeCarpool} {
		for _, blocked := range []bool{true, false} {
			paths := allowed
			if blocked {
				paths = restricted
			}
			for _, path := range paths {
				for _, method := range []string{http.MethodGet, http.MethodPost} {
					t.Run(mode+method+path, func(t *testing.T) {
						router := gin.New()
						group := router.Group("/api/v1")
						group.Use(CarpoolModeGuard(&config.Config{RunMode: mode}))
						called := false
						group.Handle(method, path, func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) })
						recorder := httptest.NewRecorder()
						router.ServeHTTP(recorder, httptest.NewRequest(method, "/api/v1"+path, nil))
						want := http.StatusNoContent
						if mode == config.RunModeCarpool && blocked {
							want = http.StatusForbidden
						}
						if recorder.Code != want {
							t.Fatalf("status = %d, want %d", recorder.Code, want)
						}
						if called != (want == http.StatusNoContent) {
							t.Fatal("incorrect downstream handler execution")
						}
					})
				}
			}
		}
	}
}

func TestCarpoolBatchImageGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{config.RunModeStandard, config.RunModeCarpool} {
		router := gin.New()
		router.POST("/v1/images/batches", CarpoolModeGuard(&config.Config{RunMode: mode}), func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/images/batches", nil))
		want := http.StatusNoContent
		if mode == config.RunModeCarpool {
			want = http.StatusForbidden
		}
		if recorder.Code != want {
			t.Fatalf("%s: got %d, want %d", mode, recorder.Code, want)
		}
	}
}
