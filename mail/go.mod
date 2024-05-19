module github.com/charkpep/mail-consumer

go 1.22.3

require (
	github.com/charkpep/usd_rate_api/shared v0.0.0-00010101000000-000000000000
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redsync/redsync/v4 v4.13.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/redis/go-redis/v9 v9.5.1 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
)

replace github.com/charkpep/usd_rate_api/shared => ../shared
