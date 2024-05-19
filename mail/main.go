package main

import (
	"crypto/tls"
	"github.com/charkpep/mail-consumer/lib"
	"github.com/charkpep/usd_rate_api/shared"
	"github.com/redis/go-redis/v9"
	"gopkg.in/gomail.v2"
	"log"
	"os"
	"time"
)

func main() {
	var (
		password = os.Getenv("SMTP_PASS")
		from     = os.Getenv("SMTP_USER")
	)
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	opt.DialTimeout = 240 * time.Second
	opt.MaxRetries = 10
	rdb := redis.NewClient(opt)
	d := gomail.NewDialer("smtp.gmail.com", 587, from, password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	db := shared.NewDb(rdb)
	c := lib.NewMailConsumer(db, d)
	c.Consume()
}
