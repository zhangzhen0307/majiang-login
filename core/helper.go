package core

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"net/smtp"
	"strings"
)

var db = newDb()

func newDb() *gorm.DB {
	dsn := Config.MysqlConfig.Dsn
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("mysql open failed")
	}
	return db
}

func GetDb() *gorm.DB {
	return db
}

var red = newRedis()

func newRedis() *redis.Client {
	var red = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return red
}

func GetRedis() *redis.Client {
	return red
}

type mailMessage struct {
	sender  string
	to      []string
	subject string
	body    string
}

func SendEmailTo(receiver string, subject string, content string) {
	var m = mailMessage{sender: EmailConfig.Email, to: []string{receiver}, subject: subject, body: content}
	var auth = smtp.PlainAuth("", EmailConfig.Email, EmailConfig.Password, EmailConfig.Server)
	var e = smtp.SendMail(EmailConfig.Server+":"+EmailConfig.Port, auth, EmailConfig.Email, []string{receiver}, []byte(buildMailMessage(m)))
	if e != nil {
		fmt.Println(e)
	}
}

func buildMailMessage(m mailMessage) []byte {
	msg := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg += fmt.Sprintf("From: %s\r\n", m.sender)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(m.to, ";"))
	msg += fmt.Sprintf("Subject: %s\r\n", m.subject)
	msg += fmt.Sprintf("\r\n%s\r\n", m.body)

	return []byte(msg)
}
