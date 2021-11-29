package route

import "github.com/gin-gonic/gin"

type Registrar interface {
	RegisterAuthRoute(gin.IRoutes)
	RegisterNonAuthRoute(gin.IRoutes)
}
