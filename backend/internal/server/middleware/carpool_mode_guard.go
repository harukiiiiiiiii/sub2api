package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// CarpoolModeGuard blocks both user and admin commercial APIs, including
// payment callbacks, regardless of any previously persisted feature switches.
func CarpoolModeGuard(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg == nil || cfg.RunMode != config.RunModeCarpool {
			c.Next()
			return
		}
		routePath := c.FullPath()
		if routePath == "" {
			routePath = c.Request.URL.Path
		}
		path := strings.TrimPrefix(routePath, "/api/v1")
		for _, prefix := range []string{
			"/payment", "/admin/payment", "/redeem", "/admin/redeem-codes",
			"/admin/promo-codes", "/admin/affiliates", "/user/aff",
			"/auth/validate-promo-code", "/auth/oauth/wechat/payment",
			"/pages", "/model-plaza", "/auth/register", "/v1/images/batches",
		} {
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				response.Forbidden(c, "This feature is disabled in carpool mode.")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
