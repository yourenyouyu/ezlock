package router

import (
	"ezlock/controller"
	"ezlock/middleware"
	"github.com/gin-gonic/gin"
)

// 账户相关的接口
func Account(router *gin.Engine) {
	// 用户的登录
	router.POST("/login", controller.Login)
	account := router.Group("/account")
	account.Use(middleware.AuthMiddlerware.MiddlewareFunc())
	{
		// 获取用户手机号码
		account.POST("/get_phone_number", controller.GetPhone)
		// 获取用户信息
		account.POST("/get_user_info", controller.GetUserInfo)
	}

}
