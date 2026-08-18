package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

const (
	statusOK       = "ok"
	statusDegraded = "degraded"
)

//go:generate go run go.uber.org/mock/mockgen -source=health.go -destination=../mocks/mock_health_usecase.go -package=mocks -typed
type HealthUsecase interface {
	GetHealth(ctx context.Context) (*domain.HealthStatus, error)
}

type healthUsecase struct {
	repo repository.HealthRepository
}

func NewHealthUsecase(repo repository.HealthRepository) *healthUsecase {
	return &healthUsecase{repo: repo}
}

func (u *healthUsecase) GetHealth(ctx context.Context) (*domain.HealthStatus, error) {
	dbConnected := u.repo.Ping(ctx) == nil
	redisConnected := u.repo.PingRedis(ctx) == nil

	status := statusOK
	if !dbConnected || !redisConnected {
		status = statusDegraded
	}

	return &domain.HealthStatus{
		Status:         status,
		DBConnected:    dbConnected,
		RedisConnected: redisConnected,
	}, nil
}
