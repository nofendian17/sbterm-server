package repository

import "context"

//go:generate go run go.uber.org/mock/mockgen -source=health.go -destination=../mocks/mock_health_repository.go -package=mocks -typed
type HealthRepository interface {
	PingCache(ctx context.Context) error
}
