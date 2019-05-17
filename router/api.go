package router

import (
	"ezlock/controller"
	"ezlock/middleware"
	"github.com/gin-gonic/gin"
)

// v1版本的api
func Api(router *gin.Engine) {

	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddlerware.MiddlewareFunc())
	{
		// 获取默认门锁信息
		api.GET("/lock/default", controller.GetDefaultLock)
		// 设置默认门锁
		api.POST("/lock/default", controller.SetDefaultLock)

		// 获取锁列表 showValid 为false 显示所有表列表，showValid 为true显示所有可用锁列表
		api.GET("/lock/list", controller.GetLockList)
		// 生成开锁密钥
		api.POST("/lock/open", controller.GetOpenLockKey)
		// 绑定新锁，添加设备
		api.POST("/lock/info", controller.AddLock)
		// 修改门锁信息 只可以修改属于自己的并且没有被删除的锁
		api.PUT("/lock/info", controller.UpdateLock)
		// 删除门锁信息 只可以删除属于自己的门锁，逻辑删除
		api.DELETE("/lock/info", controller.DeleteLock)

		// 查看某一把锁对应的授权详细信息
		api.GET("/lock/auth/list", controller.GetLockAuthList)

		// 分享门锁的授权
		api.POST("/lock/auth", controller.CreateLockAuth)
		// 使用门锁的授权
		api.PUT("/lock/auth", controller.UseLockAuth)
		// 撤销自己发出的门锁的授权信息
		api.POST("/auth/revoke", controller.RevokeAuth)

		// 生成添加门卡的密钥
		api.POST("/lock/card/add", controller.GetAddCardKey)
		// 生成删除门卡的密钥
		api.POST("/lock/card/del", controller.GetDelCardKey)

		// 添加门卡
		api.POST("/lock/card", controller.SetLockCard)
		// 更新门卡信息
		api.PUT("/lock/card", controller.UpdateCard)
		// 查看此锁绑定的门卡
		api.GET("/lock/card", controller.GetLockCardList)
		// 删除门卡
		api.DELETE("/lock/card", controller.DelCard)

		// 生成获取日志的密钥
		api.PUT("/lock/log", controller.GetLogKey)
		// 查看某一把锁对应的操作日志
		api.GET("/lock/log", controller.GetLockOperateLog)
		// 添加某一把锁对应的操作日志，将硬件给的日志信息解密，写入数据库
		api.POST("/lock/log", controller.SetLockOperateLog)

	}

}
