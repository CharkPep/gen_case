package main

import (
	"context"
	consumer "github.com/charkpep/usd_rate_api/consumer/lib"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"time"
)

func getEnvs() (redis, stream, group string) {
	var ok bool
	redis, ok = os.LookupEnv("REDIS_URL")
	if !ok {
		log.Fatalf("missing REDIS URL")
	}

	stream, ok = os.LookupEnv("REDIS_STEAM")
	if !ok {
		log.Fatalf("missing REDIS STEAM")
	}

	group, ok = os.LookupEnv("CONSUMPTION_GROUP")
	if !ok {
		log.Fatalf("missing CONSUMPTION GROUP")
	}

	return
}

func main() {
	url, steam, group := getEnvs()

	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	opt.DialTimeout = 240 * time.Second
	opt.MaxRetries = 10
	rdb := redis.NewClient(opt)
	defer rdb.Close()
	if err := consumer.CheckAndCreateGroup(context.Background(), rdb, steam, group, "0"); err != nil {
		log.Println(err)
		rdb.Close()
		os.Exit(1)
	}

	c, err := consumer.NewConsumer(rdb, consumer.Config{Name: "consumer_go", Group: group, Stream: steam, Start: ">"})
	if err != nil {
		log.Println(err)
		rdb.Close()
		os.Exit(1)
	}

	defer c.Close()
	c.Consume()
}
