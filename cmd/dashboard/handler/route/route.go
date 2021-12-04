package route

import "github.com/gin-gonic/gin"

type Registrar interface {
	RegisterRoute(gin.IRoutes)
}
