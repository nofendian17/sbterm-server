package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=shareholding_network.go -destination=../mocks/mock_shareholding_network_repository.go -package=mocks -typed
type ShareholdingNetworkRepository interface {
	GetShareholdingNetwork(ctx context.Context, rootID, rootType string, maxDepth, maxEdgePerNode int) (*domain.ShareholdingNetwork, error)
}
