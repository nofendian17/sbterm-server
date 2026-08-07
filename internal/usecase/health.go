package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

const statusOK = "ok"

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
	return &domain.HealthStatus{
		Status:      statusOK,
		DBConnected: dbConnected,
	}, nil
}
