package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AUTHENTICATED_KEY is set to true on the context when the request carries a valid login token.
const AUTHENTICATED_KEY = "AUTHENTICATED"

// tokenFromRequest reads the login token from the cookie, falling back to the Authorization header.
func tokenFromRequest(ctx *gin.Context) string {
	if token, err := ctx.Cookie(TOKEN_COOKIE); err == nil && token != "" {
		return token
	}
	header := ctx.GetHeader("Authorization")
	if len(header) > 7 && header[:7] == "Bearer " {
		return header[7:]
	}
	return ""
}

// tokenIsValid reports whether the token is signed with our secret and not expired.
func tokenIsValid(tokenString, secret string) bool {
	if tokenString == "" {
		return false
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}
	expired, hasExpiry := claims["expired"].(float64)
	return hasExpiry && float64(time.Now().Unix()) <= expired
}

// OptionalAuth never rejects a request. It only records whether the caller is logged in, for
// endpoints that are public but show more to a logged-in admin.
func OptionalAuth(secret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set(AUTHENTICATED_KEY, tokenIsValid(tokenFromRequest(ctx), secret))
		ctx.Next()
	}
}

// IsAuthenticated reads the flag set by OptionalAuth.
func IsAuthenticated(ctx *gin.Context) bool {
	return ctx.GetBool(AUTHENTICATED_KEY)
}
