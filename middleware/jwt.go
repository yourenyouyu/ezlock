package middleware

import (
	"ezlock/config"
	"ezlock/utils"
	"github.com/appleboy/gin-jwt"
	"github.com/gin-gonic/gin"
	"time"
)

var AuthMiddlerware jwt.GinJWTMiddleware

func CreateToken(userId string) (string, time.Time, error) {
	AuthMiddlerware.MiddlewareInit()
	// 默认id字段存放userid，如果要加自定义的payload则在下面的data字段加入
	return AuthMiddlerware.TokenGenerator(userId, nil)
}

func init() {

	AuthMiddlerware = jwt.GinJWTMiddleware{
		Realm:      config.ProjName,
		Key:        []byte(config.JwtSecretKey),
		Timeout:    15 * time.Minute,
		MaxRefresh: 15 * time.Minute,
		Authenticator: func(c *gin.Context) (interface{}, error) {
			return nil, nil
		},
		//Authorizator: authorizator,
		Unauthorized: func(c *gin.Context, code int, message string) {
			utils.ResponseError(utils.UNAUTH, message, c)
			return
		},

		TokenLookup:   "header: Authorization",
		TokenHeadName: config.ProjName,
		TimeFunc:      time.Now,
	}
}
