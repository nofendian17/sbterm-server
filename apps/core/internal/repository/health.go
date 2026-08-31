package repository

import "context"

//go:generate go run go.uber.org/mock/mockgen -source=health.go -destination=../mocks/mock_health_repository.go -package=mocks -typed

// HealthRepository checks connectivity to external dependencies.
type HealthRepository interface {
	Ping(ctx context.Context) error
	PingRedis(ctx context.Context) error
}
