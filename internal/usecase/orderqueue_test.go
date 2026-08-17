package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
)

func TestOrderQueueUsecaseGetOrderQueue(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns order queue"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.OrderQueueData{
				IsOpenMarket: false,
				Orders: []domain.OrderQueueOrder{{
					ID:          "3495619555",
					StockCode:   "SLIS",
					ActionType:  "ACTION_TYPE_BUY",
					Price:       101,
					Status:      "ORDER_STATUS_PARTIAL_MATCH",
					Open:        39,
					Lot:         50,
					BoardType:   "BOARD_TYPE_REGULAR",
					BrokerCode:  "YP",
					QueueLot:    0,
					BrokerGroup: "BROKER_GROUP_FOREIGN",
				}},
				Pagination: domain.OrderQueuePagination{HasNextPage: true},
			}
			repo := mocks.NewMockOrderQueueRepository(ctrl)
			repo.EXPECT().GetOrderQueue(gomock.Any(), "SLIS", "ACTION_TYPE_ALL", "BOARD_TYPE_REGULAR", "ORDER_STATUS_OPEN", "SORT_BY_QUEUE", "SORT_DIRECTION_ASC", 100, int64(101)).Return(want, tt.repoErr)

			uc := NewOrderQueueUsecase(repo)
			got, err := uc.GetOrderQueue(context.Background(), "SLIS", "ACTION_TYPE_ALL", "BOARD_TYPE_REGULAR", "ORDER_STATUS_OPEN", "SORT_BY_QUEUE", "SORT_DIRECTION_ASC", 100, 101)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
