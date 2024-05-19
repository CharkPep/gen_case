
### About

This repository contains Practical case for [Software Engineering School 4.0](https://www.genesis-for-univ.com/genesis-software-engineering-school-4).
The task was to create a simple API to query USD rate and be able to subscribe to updates (1 per/day) on email.

### Quickstart

To start application go ahead and export or update in docker-compose your SMTP credentials. 

```bash

$ docker compose up

# Get current rate from default bank (for now ПриватБанк)
$ curl --request GET \
     --url 'http://localhost:8000/rate'

# Get current rate from monobank
$ curl --request GET \
     --url 'http://localhost:8000/rate/monobank'

# Subscribe for updated from default bank (as updates send ones per day,
# the simplest way to trigger update is to restart docker compose without downing it)
$ curl --request GET \
     --url 'http://localhost:8000/subscribe

```

### Documentations

## Endpoints

`/rate/{bank}`

**params**

- *bank* - list can be found on https://minfin.com.ua/ua/currency/banks/usd/, list is updated every 15 min.
 
Return bank current online and cash exchange rates:

```go
type BankRate struct {
    Bank        string    `json:"bank"`
    Buy         float64   `json:"buy"`
    BuyOnline   float64   `json:"buy_online"`
    Sell        float64   `json:"sell"`
    SellOnline  float64   `json:"sell_online"`
    LastUpdated time.Time `json:"update_at"`
    // last update source url
    Source  string `json:"source"`
    SiteUrl string `json:"site_url"`
}
```

Application is split into separate services (lambdas): **API, Scraper, Consumer, Mailer**. From the beginning I was looking to deploy the application, 
which in turn reflected on the architecture. Lets look at each service:

- **API:** Simple API to application
- **Scraper** - (lambda) Source of rates for the application, the main idea behind scraper is that it can be distributed to get rates from different sources and put them into queue to be processed by **Consumer**
- **Consumer** - Consumes updates from scraper and updated relevant information in **Redis**. I wanted to decouple Scraper from data store, so that was the simples way. Also can be extended to analyze data, store timestamped data etc.
- **Mailer** - (lambda) Simple service to synchronously send emails

Also application uses Redis to store and publish data between services. 

### Design solutions so far:

- Decouple scraper from DB by introducing queue and a consumer, also gives the ability to manipulate data without scraper knowing it.
- Use Redis as data storage as it gives ability to store timestamp data(gives extensibility) and task requires only key-value data.


