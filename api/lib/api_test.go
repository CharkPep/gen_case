package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/charkpep/usd_rate_api/shared/model"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"net"
	"net/http"
	"testing"
	"time"
)

func SpinUpRedis(t *testing.T) string {
	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	redisC, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("Could not start redis: %s", err)
	}

	t.Cleanup(func() {
		if err := redisC.Terminate(context.Background()); err != nil {
			log.Fatalf("Could not container redis: %s", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	endpoint, err := redisC.Endpoint(ctx, "")
	if err != nil {
		log.Fatalf("Could get redis endpoint: %s", err)
	}

	t.Log(endpoint)
	return endpoint
}

func GetFreePort(t *testing.T) int {
	var (
		a   *net.TCPAddr
		err error
	)

	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port
		}
	}

	t.Fatalf("can not get a free port")
	return 0
}

func TestApi(t *testing.T) {
	type tt struct {
		addr   string
		status int
		res    string
	}
	opt, err := redis.ParseURL(fmt.Sprintf("redis://%s", SpinUpRedis(t)))
	if err != nil {
		t.Fatal(err)
	}

	rdb := redis.NewClient(opt)
	rate := model.BankRate{
		Bank:        "bank",
		Buy:         10,
		BuyOnline:   10,
		Sell:        11,
		SellOnline:  11,
		LastUpdated: time.Time{},
		Source:      "source.com",
		SiteUrl:     "/bank.com",
	}

	rateBuff, _ := json.Marshal(rate)
	rdb.Set(context.Background(), "rate:usd:bank", string(rateBuff), 0)

	api := NewApi(rdb)
	addr := fmt.Sprintf(":%d", GetFreePort(t))
	go api.ListenAndServer(addr)
	ts := []tt{
		{
			addr:   fmt.Sprintf("%s/%s", addr, "rate/bank"),
			status: 200,
			res:    string(rateBuff),
		},
		{
			addr:   fmt.Sprintf("%s/%s", addr, "rate/private"),
			status: 204,
			res:    "not found",
		},
	}

	for _, test := range ts {
		client := http.DefaultClient
		res, err := client.Get(test.addr)
		if err != nil {
			t.Fatal(err)
		}
		
	}

}
