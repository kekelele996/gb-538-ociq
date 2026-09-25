package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"industrial-noise-source-attribution/backend/internal/model"
	"industrial-noise-source-attribution/backend/internal/util"
)

type Authenticator struct{ secret []byte }

func NewAuthenticator(secret string) *Authenticator { return &Authenticator{secret: []byte(secret)} }

func (a *Authenticator) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			util.RespondError(c, util.NewError(http.StatusUnauthorized, "unauthorized", "需要 Bearer 访问令牌", nil))
			c.Abort()
			return
		}
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return a.secret, nil
		}, jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			util.RespondError(c, util.NewError(http.StatusUnauthorized, "unauthorized", "访问令牌无效或已过期", nil))
			c.Abort()
			return
		}
		idValue, idErr := strconv.ParseUint(stringClaim(claims, "sub"), 10, 64)
		if idErr != nil || idValue == 0 || stringClaim(claims, "role") == "" {
			util.RespondError(c, util.NewError(http.StatusUnauthorized, "unauthorized", "访问令牌声明不完整", nil))
			c.Abort()
			return
		}
		c.Set("actor", model.Actor{
			ID: uint(idValue), Username: stringClaim(claims, "username"), DisplayName: stringClaim(claims, "display_name"),
			Role: stringClaim(claims, "role"), RequestID: c.GetString("request_id"),
		})
		c.Next()
	}
}

func stringClaim(claims jwt.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return value
}

func ActorFromContext(c *gin.Context) (model.Actor, bool) {
	value, exists := c.Get("actor")
	actor, ok := value.(model.Actor)
	return actor, exists && ok
}

type rateWindow struct {
	started time.Time
	count   int
}

type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	limit   int
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{windows: make(map[string]rateWindow), limit: limit}
}

func (r *RateLimiter) Middleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := scope + ":" + c.ClientIP()
		if actor, ok := ActorFromContext(c); ok {
			key = scope + ":" + actor.Username
		}
		now := time.Now()
		r.mu.Lock()
		window := r.windows[key]
		if window.started.IsZero() || now.Sub(window.started) >= time.Minute {
			window = rateWindow{started: now}
		}
		window.count++
		r.windows[key] = window
		allowed := window.count <= r.limit
		retryAfter := int(time.Minute.Seconds() - now.Sub(window.started).Seconds())
		if len(r.windows) > 10000 {
			for candidate, state := range r.windows {
				if now.Sub(state.started) > 2*time.Minute {
					delete(r.windows, candidate)
				}
			}
		}
		r.mu.Unlock()
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			util.RespondError(c, util.NewError(http.StatusTooManyRequests, "rate_limited", "请求过于频繁，请稍后重试", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}
