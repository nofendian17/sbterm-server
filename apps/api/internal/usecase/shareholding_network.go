package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=shareholding_network.go -destination=../mocks/mock_shareholding_network_usecase.go -package=mocks -typed
type ShareholdingNetworkUsecase interface {
	GetShareholdingNetwork(ctx context.Context, rootID, rootType string, maxDepth, maxEdgePerNode int) (*domain.ShareholdingNetwork, error)
}

type shareholdingNetworkUsecase struct {
	repo repository.ShareholdingNetworkRepository
}

func NewShareholdingNetworkUsecase(repo repository.ShareholdingNetworkRepository) *shareholdingNetworkUsecase {
	return &shareholdingNetworkUsecase{repo: repo}
}

func (u *shareholdingNetworkUsecase) GetShareholdingNetwork(ctx context.Context, rootID, rootType string, maxDepth, maxEdgePerNode int) (*domain.ShareholdingNetwork, error) {
	return u.repo.GetShareholdingNetwork(ctx, rootID, rootType, maxDepth, maxEdgePerNode)
}
