package controllers

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/gin-contrib/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"login/core"
	"math/rand"
	"strconv"
	"time"
)
import "github.com/gin-gonic/gin"

type PostLoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AutoLoginBody struct {
	SessionId string `json:"session_id"`
}

type ForgetPasswordBody struct {
	Email string `json:"email"`
}

type ModifyPasswordBody struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Code            string `json:"code"`
}

type RegisterBody struct {
	Email           string `json:"email"`
	Name            string `json:"name"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type NormalResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

type RegisterModel struct {
	ID         uint `gorm:"primaryKey"`
	Email      string
	Password   string
	Name       string
	EmailToken string
}

func PostLogin(c *gin.Context) {
	var body PostLoginBody
	var db = core.GetDb()

	if c.ShouldBindJSON(&body) != nil {
		c.JSON(200, ErrorResponse{Code: 1, Message: "missing params"})
		return
	}

	var email = body.Email
	var password = body.Password

	var result = map[string]interface{}{}
	db.Table("users").Where(map[string]interface{}{"email": email}).Select("id", "password").Take(&result)
	if len(result) == 0 {
		c.JSON(200, ErrorResponse{Code: 1, Message: "invalid email or password"})
		return
	}
	var match = bcrypt.CompareHashAndPassword([]byte(result["password"].(string)), []byte(password))
	if match != nil {
		c.JSON(200, ErrorResponse{Code: 1, Message: "invalid email or password"})
	} else {
		session := sessions.Default(c)
		session.Set("uuid", result["id"].(int32))
		session.Save()
		c.JSON(200, NormalResponse{Code: 0, Data: session.ID()})
	}
	return
}

func Register(c *gin.Context) {
	var body RegisterBody
	var db = core.GetDb()

	if c.ShouldBindJSON(&body) != nil {
		c.JSON(200, ErrorResponse{Code: 1, Message: "missing params"})
		return
	}
	var hashed, _ = bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	var token = generateRandomToken()
	var m = RegisterModel{Email: body.Email, Name: body.Name, Password: string(hashed), EmailToken: token}
	db.Table("pending_users").Create(&m)
	var link = core.Config.ServerAddress + "/confirmRegister?email_token=" + token + "&id=" + strconv.Itoa(int(m.ID))
	var subject = "完成注册链接"
	var content = "请<a href=\"" + link + "\" target=\"_blank\">点击链接</a>完成注册"
	core.SendEmailTo(body.Email, subject, content)
	c.JSON(200, NormalResponse{Code: 0})
}

func ConfirmRegister(c *gin.Context) {
	var token = c.Query("email_token")
	var id = c.Query("id")
	var db = core.GetDb()

	var result = map[string]interface{}{}
	db.Table("pending_users").Where(map[string]interface{}{"id": id, "email_token": token}).Take(&result)
	if len(result) == 0 {
		//todo 改为返回html页面
		c.JSON(200, gin.H{"message": "link expired or not exist"})
	} else {
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Table("users").Create(map[string]interface{}{"email": result["email"], "name": result["name"], "password": result["password"]}).Error; err != nil {
				return err
			}

			var res = map[string]interface{}{}
			if err := tx.Table("pending_users").Where(map[string]interface{}{"id": id}).Delete(&res).Error; err != nil {
				return err
			}

			return nil
		}); err != nil {
			panic(err)
		} else {
			c.JSON(200, gin.H{"message": "activition succeed"})
		}
	}
}

func AutoLogin(c *gin.Context) {
	var body AutoLoginBody

	if c.ShouldBindJSON(&body) != nil {
		c.JSON(200, ErrorResponse{Code: 1, Message: "missing params"})
		return
	}

	var seid = "session_" + body.SessionId
	fmt.Println(seid)
	var rdb = core.GetRedis()
	var ctx = context.Background()

	val, err := rdb.Get(ctx, seid).Result()
	if err != nil {
		c.JSON(200, ErrorResponse{Code: 1, Message: "session expired, need relogin"})
		return
	}
	fmt.Println(val)

	var result = map[interface{}]interface{}{}
	dec := gob.NewDecoder(bytes.NewBuffer([]byte(val)))
	err = dec.Decode(&result)
	if err != nil {
		c.JSON(200, ErrorResponse{Code: 1, Message: "session expired, need relogin"})
		return
	}
	c.JSON(200, NormalResponse{Code: 0})
	return
}

func ForgetPassword(c *gin.Context) {
	var body ForgetPasswordBody
	var db = core.GetDb()

	if c.ShouldBindJSON(&body) != nil {
		c.JSON(200, ErrorResponse{Code: 1, Message: "missing params"})
		return
	}

	var email = body.Email
	var result = map[string]interface{}{}
	db.Table("users").Where(map[string]string{"email": email}).Take(&result)
	if len(result) == 0 {
		c.JSON(200, ErrorResponse{Code: 1, Message: "user email not exist"})
		return
	}
	var code = generateRandomCode()
	var content = "您的验证码为<strong>" + code + "</strong>"
	var subject = "验证码"
	core.SendEmailTo(email, subject, content)
	db.Table("forget_password").Create(map[string]interface{}{"email": email, "code": code})
	c.JSON(200, NormalResponse{Code: 0})
}

func ModifyPassword(c *gin.Context) {
	var body ModifyPasswordBody
	var db = core.GetDb()

	if c.ShouldBindJSON(&body) != nil {
		c.JSON(200, ErrorResponse{Code: 1, Message: "missing params"})
		return
	}

	var result = map[string]interface{}{}
	db.Table("forget_password").Where(map[string]interface{}{"email": body.Email, "code": body.Code}).Take(&result)
	if len(result) == 0 {
		c.JSON(200, ErrorResponse{Code: 1, Message: "code error"})
		return
	}

	var hashed, _ = bcrypt.GenerateFromPassword([]byte(body.Password), 10)

	if err := db.Transaction(func(tx *gorm.DB) error {
		var res = map[string]interface{}{}
		if err := tx.Table("forget_password").Where(map[string]interface{}{"email": body.Email, "code": body.Code}).Delete(&res).Error; err != nil {
			return err
		}

		if err := tx.Table("users").Where(map[string]interface{}{"email": body.Email}).Updates(map[string]interface{}{"password": string(hashed)}).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		panic(err)
	} else {
		c.JSON(200, NormalResponse{Code: 0})
	}
}

func generateRandomToken() string {
	rand.Seed(time.Now().UnixNano())
	var letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	b := make([]byte, 10)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func generateRandomCode() string {
	rand.Seed(time.Now().UnixNano())
	var letterBytes = "1234567890"

	b := make([]byte, 4)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
