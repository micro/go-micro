package account

import (
	"time"

	"github.com/asim/go-micro/cmd/dashboard/v4/config"
	"github.com/asim/go-micro/cmd/dashboard/v4/handler/route"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

type service struct{}

func NewRouteRegistrar() route.Registrar {
	return service{}
}

func (s service) RegisterRoute(router gin.IRoutes) {
	router.POST("/api/account/login", s.Login)
	router.Use(route.AuthRequired()).GET("/api/account/profile", s.Profile)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token string `json:"token" binding:"required"`
}

// @Tags Account
// @ID account_login
// @Param	input	body		loginRequest	true		"request"
// @Success 200 	{object}	loginResponse	"success"
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/account/login [post]
func (s *service) Login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); nil != err {
		ctx.Render(400, render.String{Format: err.Error()})
		return
	}
	if req.Username != config.GetServerConfig().Auth.Username ||
		req.Password != config.GetServerConfig().Auth.Password {
		ctx.Render(400, render.String{Format: "incorrect username or password"})
		return
	}
	claims := jwt.StandardClaims{
		Subject:   req.Username,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(config.GetAuthConfig().TokenExpiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.GetAuthConfig().TokenSecret))
	if err != nil {
		ctx.Render(400, render.String{Format: err.Error()})
		return
	}
	ctx.JSON(200, loginResponse{Token: signedToken})
}

type profileResponse struct {
	Name string `json:"name"`
}

// @Security ApiKeyAuth
// @Tags Account
// @ID account_profile
// @Success 200 	{object}	profileResponse	"success"
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/account/profile [get]
func (s *service) Profile(ctx *gin.Context) {
	ctx.JSON(200, profileResponse{Name: config.GetAuthConfig().Username})
}
