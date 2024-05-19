package lib

import (
	"context"
	"fmt"
	"github.com/charkpep/usd_rate_api/shared"
	"github.com/charkpep/usd_rate_api/shared/model"
	"gopkg.in/gomail.v2"
	"log"
	"os"
	"strings"
	"time"
)

var logger = log.New(os.Stdout, "email-sender: ", log.LstdFlags)

type Mail struct {
	To   string
	Data *model.BankRate
}

type MailConsumer struct {
	cache  map[string]*model.BankRate
	db     *shared.Database
	dialer *gomail.Dialer
}

func NewMailConsumer(db *shared.Database, dialer *gomail.Dialer) *MailConsumer {
	m := &MailConsumer{
		db:     db,
		dialer: dialer,
		cache:  make(map[string]*model.BankRate),
	}

	return m
}

func (m MailConsumer) Consume() error {
	iter, err := m.db.GetSubscriberMails(context.TODO())
	if err != nil {
		return err
	}

	for iter.Next(context.TODO()) {
		fmt.Println(iter.Val())
		to := strings.Split(iter.Val(), ":")
		if len(to) != 2 {
			logger.Printf("elements %s does not have 2 elements\n", to)
			continue
		}

		if data, ok := m.cache[to[1]]; ok {
			if err := m.sendMail(to[0], data); err != nil {
				logger.Println(err)
			}
			continue
		}

		data, err := m.db.GetBankPrice(context.TODO(), to[1])
		if err != nil {
			logger.Println(err)
			continue
		}

		if data == nil {
			logger.Printf("bank rate for %s, not found\n", to[1])
			continue
		}

		m.cache[to[1]] = data
		if err := m.sendMail(to[0], data); err != nil {
			logger.Println(err)
		}
	}

	if iter.Err() != nil {
		return iter.Err()
	}

	return nil
}

func (m MailConsumer) sendMail(to string, data *model.BankRate) error {
	message := gomail.NewMessage()
	start := time.Now()
	message.SetHeader("Subject", "USD Price update")
	message.SetHeader("From", m.dialer.Username)
	message.SetHeader("To", to)
	message.SetBody("text/html", fmt.Sprintf("%s, Buy: %v; Sell: %v", data.Bank, data.Buy, data.Sell))
	if err := m.dialer.DialAndSend(message); err != nil {
		return err
	}

	logger.Printf("send to %s in %s\n", to, time.Now().Sub(start).String())
	return nil
}
