package route

import (
	"net/http"
	"strings"

	"github.com/asim/go-micro/cmd/dashboard/v4/config"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Method == "OPTIONS" {
			ctx.Next()
			return
		}
		tokenString := ctx.GetHeader("Authorization")
		if len(tokenString) == 0 || !strings.HasPrefix(tokenString, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, "")
			return
		}
		tokenString = tokenString[7:]
		claims := jwt.StandardClaims{}
		token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(config.GetAuthConfig().TokenSecret), nil
		})
		if err != nil {
			ctx.AbortWithError(http.StatusUnauthorized, err)
		}
		if !token.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)
		}
		ctx.Set("username", claims.Subject)
		ctx.Next()
	}
}

func CorsHandler(allowOrigin string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", allowOrigin)
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, token")
		ctx.Header("Access-Control-Allow-Methods", "POST, GET, DELETE, PUT, OPTIONS")
		ctx.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		ctx.Header("Access-Control-Allow-Credentials", "true")
		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
		}
		ctx.Next()
	}
}
