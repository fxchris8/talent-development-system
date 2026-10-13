package middlewares

import (
	"net/http"
	"time"

	"backend/internal/models/web"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	TOKEN_COOKIE = "Authorization"
	TOKEN_KEY    = "TOKEN"
	USER_KEY     = "USER"
)

func DeleteToken(ctx *gin.Context) {
	ctx.SetCookie(TOKEN_COOKIE, "delete", -1, "", "", false, true)
}

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		var tokenString string
		var err error

		// First try to get token from cookie
		tokenString, err = ctx.Cookie(TOKEN_COOKIE)

		// If no cookie, try Authorization header
		if err != nil {
			authHeader := ctx.GetHeader("Authorization")

			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString = authHeader[7:] // Remove "Bearer " prefix
			} else {
				ctx.JSON(http.StatusUnauthorized, web.ErrorResponse{
					Code:   http.StatusUnauthorized,
					Status: "Unauthorized",
					Error:  "Missing Token",
				})
				ctx.Abort()
				return
			}
		}


		// TODO: Decode and Validate it
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		//! IF the signature is different
		if err != nil {
			//! Delete Token
			// DeleteToken(ctx)
			ctx.JSON(http.StatusUnauthorized, web.ErrorResponse{
				Code:   http.StatusUnauthorized,
				Status: "Unauthorized",
				Error:  "Signature Failed",
				// Error: err.Error(),
			})
			ctx.Abort()
			return
		}

		// TODO: Claim the JWT
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// TODO: Check the expired time
			// A token without a numeric "expired" claim is rejected instead of panicking.
			expired, hasExpiry := claims["expired"].(float64)
			if !hasExpiry || float64(time.Now().Unix()) > expired {
				//! Delete the Cookie
				DeleteToken(ctx)
				ctx.JSON(http.StatusUnauthorized, web.ErrorResponse{
					Code:   http.StatusUnauthorized,
					Status: "Unauthorized",
					Error:  "Token Expired",
				})
				ctx.Abort()
				return
			}

			// TODO: Find the user with Token Sub
			// var user domain.User

			// TODO: Set the User

			// TODO: Set the Token (for logout)
			ctx.Set(TOKEN_KEY, tokenString)

			// Continue
			ctx.Next()

		} else {
			ctx.JSON(http.StatusUnauthorized, web.ErrorResponse{
				Code:   http.StatusUnauthorized,
				Status: "Unauthorized",
				Error:  "Failed to Claim",
			})
			ctx.Abort()
			return
		}
		// fmt.Println(tokenString)
		// ctx.Next()
	}
}

// RequireAuthForWrites lets anyone read (GET/HEAD/OPTIONS) but requires a login for every
// request that changes data. Used for the question bank, which the public quiz pages read.
func RequireAuthForWrites(auth gin.HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		switch ctx.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			ctx.Next()
		default:
			auth(ctx)
		}
	}
}
