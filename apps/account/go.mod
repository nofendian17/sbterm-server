module github.com/nofendian17/sbterm/apps/account

go 1.26.5

replace github.com/nofendian17/sbterm/libs/pkg => ../../libs/pkg

require (
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-playground/validator/v10 v10.26.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/golang-migrate/migrate/v4 v4.19.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/nofendian17/sbterm/libs/pkg v0.0.0
	github.com/pashagolub/pgxmock/v5 v5.1.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/samber/do/v2 v2.0.0
	github.com/spf13/viper v1.20.0
	github.com/stretchr/testify v1.10.0
	go.uber.org/mock v0.5.0
	golang.org/x/crypto v0.53.0
	golang.org/x/sync v0.16.0
)
