package shared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charkpep/usd_rate_api/shared/model"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"time"
)

type Database struct {
	db  *redis.Client
	Mux *redsync.Mutex
}

func NewDb(rdb *redis.Client) *Database {
	pool := goredis.NewPool(rdb)
	rs := redsync.New(pool)
	mux := rs.NewMutex("rate:usd:Mux", redsync.WithExpiry(2*time.Second))
	return &Database{
		db:  rdb,
		Mux: mux,
	}
}

func (db *Database) GetBanks(ctx context.Context) (*redis.ScanIterator, error) {
	res := db.db.Scan(ctx, 0, "bank:*", 0).Iterator()
	if res.Err() != nil {
		return nil, res.Err()
	}

	return res, nil
}

func (db *Database) AddSubscriber(ctx context.Context, email string, bank string) (bool, error) {
	res, err := db.db.SAdd(ctx, "rate:usd:subscribers", fmt.Sprintf("%s:%s", email, bank)).Result()
	if err != nil {
		return false, err
	}

	if res == 0 {
		return false, nil
	}

	return true, nil
}

func (db *Database) GetBankPrice(ctx context.Context, bank string) (*model.BankRate, error) {
	priceRaw, err := db.db.Get(ctx, fmt.Sprintf("rate:usd:%s", bank)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}

		return nil, err
	}

	price := model.BankRate{}
	if err := json.Unmarshal([]byte(priceRaw), &price); err != nil {
		return nil, err
	}

	return &price, nil
}

func (db *Database) SetBankPrice(ctx context.Context, price *model.BankRate) error {
	priceBuff, err := json.Marshal(price)
	if err != nil {
		return err
	}

	if err := db.db.Set(ctx, fmt.Sprintf("rate:usd:%s", price.Bank), string(priceBuff), 0).Err(); err != nil {
		return err
	}

	return nil
}

func (db *Database) GetSubscriberMails(ctx context.Context) (*redis.ScanIterator, error) {
	res := db.db.SScan(ctx, "rate:usd:subscribers", 0, "*:*", 0).Iterator()
	if res.Err() != nil {
		return nil, res.Err()
	}

	return res, nil
}
