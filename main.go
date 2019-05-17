package main

import (
	"ezlock/config"
	"ezlock/router"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"os"
	"time"
)

func main() {
	// 运行 job
	//go controller.CronCountCapInfo()
	server := gin.New()
	// product 模式运行
	gin.SetMode(gin.ReleaseMode)
	// v1 版本的api
	// 支持前端跨域，方便前后端分离调试
	server.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "HEAD", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}))
	router.Account(server)
	router.Api(server)
	if err := server.Run(fmt.Sprintf(":%d", config.ListenPort)); err != nil {
		os.Exit(1)
		fmt.Println("http server start error: ", err.Error())
	}

}
