package lib

import (
	"context"
	"encoding/json"
	"github.com/charkpep/usd_rate_api/shared"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"net/mail"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "api: ", log.LstdFlags)

const DEFAULT_BANK = "Приватбанк"

type Api struct {
	handler http.Handler
	db      *shared.Database
	done    <-chan struct{}
}

func NewApi(rdb *redis.Client) *Api {
	h := http.NewServeMux()
	api := Api{
		handler: h,
		db:      shared.NewDb(rdb),
	}

	h.Handle("GET /rate", LoggerWrapper{
		h:      api.HandlerGetRate,
		logger: logger,
	})

	h.Handle("GET /rate/{bank}", LoggerWrapper{
		h:      api.HandlerGetRate,
		logger: logger,
	})

	h.Handle("POST /subscribe", LoggerWrapper{
		h:      api.HandleSubscribe,
		logger: logger,
	})

	return &api
}

func (api Api) ListenAndServer(addr string) error {
	if err := http.ListenAndServe(addr, api.handler); err != nil {
		return err
	}

	return nil
}

func (api Api) HandlerGetRate(w http.ResponseWriter, r *http.Request) {
	var bank string
	if bank = r.PathValue("bank"); bank == "" {
		bank = DEFAULT_BANK
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	price, err := api.db.GetBankPrice(ctx, bank)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "unexpected error occurred"})
		return
	}

	if price == nil {
		w.WriteHeader(http.StatusNoContent)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "bank not found"})
		return
	}

	if err := json.NewEncoder(w).Encode(price); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "unexpected error occurred"})
		logger.Println(err)
	}

	return
}

func (api Api) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "can not read request"})
		return
	}
	params, ok := r.Form["email"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "email not specified"})
		return
	}

	if len(params) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "email not specified"})
		return
	}

	email := params[0]
	logger.Println(email)
	if _, err := mail.ParseAddress(email); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "email is wrong"})
		return
	}

	isAdded, err := api.db.AddSubscriber(context.Background(), email, DEFAULT_BANK)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "email is wrong"})
		return
	}

	if !isAdded {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct{ Message string }{Message: "email already added"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct{ Message string }{Message: "ok"})
	return
}
