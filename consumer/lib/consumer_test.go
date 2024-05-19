package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charkpep/usd_rate_api/shared/model"
	"github.com/dranikpg/gtrs"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"strings"
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

func TestCheckAndCreateGroup(t *testing.T) {
	opt, err := redis.ParseURL(fmt.Sprintf("redis://%s", SpinUpRedis(t)))
	if err != nil {
		t.Fatalf("failed to parse connection options")
	}

	rdb := redis.NewClient(opt)
	rdb.XAdd(context.Background(), &redis.XAddArgs{
		Stream: "test",
		ID:     "*",
		Values: map[string]interface{}{
			"f": "v",
		},
	})

	// Create group
	if err := CheckAndCreateGroup(context.Background(), rdb, "test", "test", "0"); err != nil {
		t.Fatal(err)
	}

	AssertGroupExists(t, rdb, "test", "test", "0-0")

	// Check if error returns on the subsequent request
	if err := CheckAndCreateGroup(context.Background(), rdb, "test", "test", "0"); err != nil {
		t.Fatal(err)
	}

}

func AssertGroupExists(t *testing.T, rdb *redis.Client, stream, group, offset string) {
	info := rdb.XInfoGroups(context.Background(), stream)
	if info.Err() != nil {
		t.Error(info.Err())
	}

	res, _ := info.Result()
	if len(res) != 1 {
		t.Errorf("expected 1 group, got %d\n", len(res))
	}

	if res[0].Name != group || res[0].LastDeliveredID != offset {
		t.Errorf("expected group to have name %s and offset %s, got %s and %s\n", group, offset, res[0].Name, res[0].LastDeliveredID)
	}
}

func TestUnmarshalMessage(t *testing.T) {
	type tt struct {
		i map[string]any
		e BankRateMessage
	}

	updateTime, _ := time.Parse(time.DateTime, "2024-01-01 00:00:00")
	ts := []tt{
		{
			i: map[string]any{
				"bank":      "bank",
				"buy":       "1.000",
				"sell":      "1.000",
				"update_at": updateTime.Format(time.RFC3339),
				"site_url":  "/bank.com",
			},
			e: BankRateMessage{
				Bank:       "bank",
				Buy:        1.000,
				BuyOnline:  0,
				Sell:       1.000,
				SellOnline: 0,
				UpdateAt:   updateTime,
				SiteUrl:    "/bank.com",
			},
		},
		{
			i: map[string]any{
				"bank":        "bank",
				"buy_online":  "1",
				"sell_online": "1.050",
				"update_at":   updateTime.Format(time.RFC3339),
				"site_url":    "/bank.com",
			},
			e: BankRateMessage{
				Bank:       "bank",
				BuyOnline:  1,
				SellOnline: 1.050,
				UpdateAt:   updateTime,
				SiteUrl:    "/bank.com",
			},
		},
	}

	for i, test := range ts {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			b := BankRateMessage{}
			if err := b.Unmarshal(test.i); err != nil {
				t.Error(err)
			}

			if b != test.e {
				t.Errorf("expected %#v, got %#v\n", test.e, b)
			}
		})
	}

}

func TestUnmarshalRequiredFields(t *testing.T) {
	type tt struct {
		i map[string]any
		e error
	}

	updateTime, _ := time.Parse(time.DateTime, "2024-01-01 00:00:00")
	ts := []tt{
		{
			i: map[string]any{
				"bank":      "",
				"update_at": updateTime,
			},
			e: fmt.Errorf("missing required field Bank"),
		},
	}

	for i, test := range ts {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			b := BankRateMessage{}
			err := b.Unmarshal(test.i)
			if err == nil {
				t.Fatalf("expected error, got nil\n")
			}

			if !errors.As(err, &gtrs.ParseError{}) {
				t.Errorf("expected error to be as %T, got %T\n", gtrs.ParseError{}, err)
			}

			if strings.Compare(err.(gtrs.ParseError).Err.Error(), test.e.Error()) != 0 {
				t.Errorf("expected error to be %q, got %q\n", test.e, err.(gtrs.ParseError).Err)
			}

		})
	}

}

func MessageToMap(b BankRateMessage) map[string]any {
	m := make(map[string]any)

	// Note: if message interface changes frequently, good idea to rewrite using reflect
	m["bank"] = b.Bank
	m["buy"] = b.Buy
	m["sell"] = b.Sell
	m["buy_online"] = b.BuyOnline
	m["sell_online"] = b.SellOnline
	m["site_url"] = b.SiteUrl
	m["update_at"] = b.UpdateAt
	m["source_url"] = b.SourceUrl
	return m
}

func TestConsume(t *testing.T) {
	type tt struct {
		db  []model.BankRate
		msg BankRateMessage
		key string
		exp model.BankRate
	}

	now := time.Now().Round(time.Millisecond)
	ts := []tt{
		{
			db: []model.BankRate{},
			msg: BankRateMessage{
				Bank:       "bank",
				Buy:        10,
				BuyOnline:  9.50,
				Sell:       12.0,
				SellOnline: 12.0,
				UpdateAt:   now,
				SiteUrl:    "/bank.com",
				SourceUrl:  "aggregator.com",
			},
			key: "rate:usd:bank",
			exp: model.BankRate{
				Bank:        "bank",
				Buy:         10.0,
				BuyOnline:   9.50,
				Sell:        12.0,
				SellOnline:  12.0,
				LastUpdated: now,
				SiteUrl:     "/bank.com",
				Source:      "aggregator.com",
			},
		},
		{
			db: []model.BankRate{
				{
					Bank:        "bank",
					Buy:         9.0,
					BuyOnline:   9.50,
					Sell:        12.0,
					SellOnline:  12.0,
					LastUpdated: now,
					SiteUrl:     "/bank.com",
					Source:      "aggregator.com",
				},
			},
			msg: BankRateMessage{
				Bank:       "bank",
				Buy:        11.0,
				BuyOnline:  9.50,
				Sell:       12.0,
				SellOnline: 12.0,
				UpdateAt:   now.Add(time.Second),
				SiteUrl:    "/bank.com",
				SourceUrl:  "aggregator.com",
			},
			key: "rate:usd:bank",
			exp: model.BankRate{
				Bank:        "bank",
				Buy:         11.0,
				BuyOnline:   9.50,
				Sell:        12.0,
				SellOnline:  12.0,
				LastUpdated: now.Add(time.Second),
				SiteUrl:     "/bank.com",
				Source:      "aggregator.com",
			},
		},
	}
	opt, err := redis.ParseURL(fmt.Sprintf("redis://%s", SpinUpRedis(t)))
	if err != nil {
		t.Fatal(err)
	}

	rdb := redis.NewClient(opt)
	for i, test := range ts {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			t.Cleanup(func() {
				t.Log("cleanup")
				rdb.FlushAll(context.Background())
			})
			for _, rate := range test.db {
				rateBuff, _ := json.Marshal(rate)
				rdb.Set(context.Background(), fmt.Sprintf("rate:usd:%s", rate.Bank), string(rateBuff), 0)
			}
			rdb.XAdd(context.Background(), &redis.XAddArgs{
				Stream: "rate:usd",
				ID:     "*",
				Values: MessageToMap(test.msg),
			})

			rdb.XGroupCreate(context.Background(), "rate:usd", "group", "0")
			// Note: db not mocked
			c, err := NewConsumer(rdb, Config{
				Name:   "consumer",
				Group:  "group",
				Stream: "rate:usd",
				Start:  ">",
			})

			if err != nil {
				t.Fatal(err)
			}

			go func() {
				if err = c.Consume(); err != nil {
					t.Log(err)
				}
			}()

			AssertLoop(t, rdb, test.key, test.exp)
			// wait for mux to unlock
			c.db.Mux.TryLock()
			c.Close()
		})
	}

}

func AssertLoop(t *testing.T, rdb *redis.Client, key string, expected model.BankRate) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	price := model.BankRate{}
	for {
		select {
		case <-ctx.Done():
			t.Errorf("timeouted: expected %#v, got %#v\n", expected, price)
			return
		default:
			res, err := rdb.Get(ctx, "rate:usd:bank").Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				t.Error(err)
			}

			if err = json.Unmarshal([]byte(res), &price); err != nil {
				t.Fatal(err)
			}

			if price == expected {
				return
			}

		}
	}

}
