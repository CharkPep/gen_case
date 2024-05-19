package consumer

import (
	"context"
	"github.com/charkpep/usd_rate_api/shared"
	"github.com/charkpep/usd_rate_api/shared/model"
	"github.com/dranikpg/gtrs"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
)

var errLogger = log.New(os.Stdout, "consumer error: ", log.LstdFlags)

type Config struct {
	Name   string
	Group  string
	Stream string
	Start  string
}

type Consumer struct {
	db   *shared.Database
	rdb  *redis.Client
	cs   *gtrs.GroupConsumer[BankRateMessage]
	done chan struct{}
}

func NewConsumer(rdb *redis.Client, conf Config) (*Consumer, error) {
	cs := gtrs.NewGroupConsumer[BankRateMessage](context.Background(), rdb, conf.Group, conf.Name, conf.Stream, conf.Start)
	return &Consumer{
		db:   shared.NewDb(rdb),
		cs:   cs,
		done: make(chan struct{}),
	}, nil
}

func (c *Consumer) Close() {
	errLogger.Println("Closing")
	c.done <- struct{}{}
	c.cs.AwaitAcks()
	c.cs.Close()
}

func (c *Consumer) Consume() error {
	for {
		select {
		case delivery := <-c.cs.Chan():
			switch delivery.Err.(type) {
			case nil:
				//TODO: write tests for race conditions, though redsync guarantees mut execution
				go c.processMessage(context.TODO(), delivery.Data)
				c.cs.Ack(delivery)
			case gtrs.ParseError:
				// Data loss is acceptable here
				errLogger.Println(delivery.Err)
				c.cs.Ack(delivery)
			default:
				return delivery.Err
			}
		case <-c.done:
			return nil
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg BankRateMessage) {
	if err := c.db.Mux.LockContext(ctx); err != nil {
		errLogger.Println(err)
		return
	}

	curPrice, err := c.db.GetBankPrice(ctx, msg.Bank)
	if err != nil {
		errLogger.Println(err)
		return
	}

	if curPrice == nil || msg.UpdateAt.Sub(curPrice.LastUpdated) >= 0 {
		price := model.BankRate{}
		mapToBankRateModel(msg, &price)
		if err := c.db.SetBankPrice(ctx, &price); err != nil {
			errLogger.Println(err)
		}
	}

	if _, err = c.db.Mux.Unlock(); err != nil {
		errLogger.Println(err)
	}

}

func mapToBankRateModel(msg BankRateMessage, cur *model.BankRate) {
	cur.Source = msg.SourceUrl
	cur.LastUpdated = msg.UpdateAt
	cur.Bank = msg.Bank
	cur.Buy = msg.Buy
	cur.Sell = msg.Sell
	cur.BuyOnline = msg.BuyOnline
	cur.SellOnline = msg.SellOnline
	cur.SiteUrl = msg.SiteUrl
}
