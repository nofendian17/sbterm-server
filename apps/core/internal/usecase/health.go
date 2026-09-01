// Package usecase implements the business logic for the core domain.

package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

const (
	statusOK       = "ok"
	statusDegraded = "degraded"
)

//go:generate go run go.uber.org/mock/mockgen -source=health.go -destination=../mocks/mock_health_usecase.go -package=mocks -typed

// HealthUsecase checks the health of service dependencies.
type HealthUsecase interface {
	GetHealth(ctx context.Context) (*domain.HealthStatus, error)
}

type healthUsecase struct {
	repo repository.HealthRepository
}

// NewHealthUsecase wires up the health usecase.
func NewHealthUsecase(repo repository.HealthRepository) HealthUsecase {
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
