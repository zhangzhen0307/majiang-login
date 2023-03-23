package main

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"login/core"
)
import "login/controllers"

func main() {
	r := gin.Default()
	store, _ := redis.NewStore(10, "tcp", "localhost:6379", "")
	r.Use(sessions.Sessions("mysession", store))
	r.POST("/postLogin", controllers.PostLogin)
	r.POST("/register", controllers.Register)
	r.GET("/confirmRegister", controllers.ConfirmRegister)
	r.POST("/autoLogin", controllers.AutoLogin)
	r.POST("/forgetPassword", controllers.ForgetPassword)
	r.POST("/modifyPassword", controllers.ModifyPassword)
	r.Run(core.Config.ServerRunAddress) // listen and serve on 0.0.0.0:8080
}
