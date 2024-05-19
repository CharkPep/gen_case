package main

import (
	"fmt"
	"github.com/charkpep/usd_rate_api/api/lib"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
)

func main() {
	url, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		log.Fatalf("missing REDIS URL")
	}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		log.Fatalf("missing PORT")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	rdb := redis.NewClient(opt)
	api := lib.NewApi(rdb)

	if err = api.ListenAndServer(fmt.Sprintf("0.0.0.0:%s", port)); err != nil {
		log.Println(err)
	}

}
