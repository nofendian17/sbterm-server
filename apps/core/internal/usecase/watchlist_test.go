package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func TestWatchlistUsecase(t *testing.T) {
	tests := []struct {
		name    string
		call    func(uc WatchlistUsecase, repo *mocks.MockWatchlistRepository) error
		setup   func(repo *mocks.MockWatchlistRepository)
		wantErr bool
	}{
		{
			name: "list success",
			call: func(uc WatchlistUsecase, repo *mocks.MockWatchlistRepository) error {
				_, err := uc.List(context.Background(), "u1")
				return err
			},
			setup: func(repo *mocks.MockWatchlistRepository) {
				repo.EXPECT().ListByUser(gomock.Any(), "u1").Return([]domain.Watchlist{{Symbol: "BBCA"}}, nil)
			},
		},
		{
			name: "list error",
			call: func(uc WatchlistUsecase, repo *mocks.MockWatchlistRepository) error {
				_, err := uc.List(context.Background(), "u1")
				return err
			},
			setup: func(repo *mocks.MockWatchlistRepository) {
				repo.EXPECT().ListByUser(gomock.Any(), "u1").Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "add success",
			call: func(uc WatchlistUsecase, repo *mocks.MockWatchlistRepository) error {
				return uc.Add(context.Background(), "u1", domain.AddWatchlistInput{Symbol: "BBCA", Label: "Bank"})
			},
			setup: func(repo *mocks.MockWatchlistRepository) {
				repo.EXPECT().Add(gomock.Any(), domain.Watchlist{UserID: "u1", Symbol: "BBCA", Label: "Bank"}).Return(nil)
			},
		},
		{
			name: "add duplicate",
			call: func(uc WatchlistUsecase, repo *mocks.MockWatchlistRepository) error {
				return uc.Add(context.Background(), "u1", domain.AddWatchlistInput{Symbol: "BBCA"})
			},
			setup: func(repo *mocks.MockWatchlistRepository) {
				repo.EXPECT().Add(gomock.Any(), domain.Watchlist{UserID: "u1", Symbol: "BBCA"}).Return(domain.ErrDuplicateWatchlist)
			},
			wantErr: true,
		},
		{
			name: "remove success",
			call: func(uc WatchlistUsecase, repo *mocks.MockWatchlistRepository) error {
				return uc.Remove(context.Background(), "u1", "BBCA")
			},
			setup: func(repo *mocks.MockWatchlistRepository) {
				repo.EXPECT().RemoveBySymbol(gomock.Any(), "u1", "BBCA").Return(nil)
			},
		},
		{
			name: "remove error",
			call: func(uc WatchlistUsecase, repo *mocks.MockWatchlistRepository) error {
				return uc.Remove(context.Background(), "u1", "BBCA")
			},
			setup: func(repo *mocks.MockWatchlistRepository) {
				repo.EXPECT().RemoveBySymbol(gomock.Any(), "u1", "BBCA").Return(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mocks.NewMockWatchlistRepository(ctrl)
			tt.setup(repo)

			uc := NewWatchlistUsecase(repo)
			err := tt.call(uc, repo)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWatchlistUsecase_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockWatchlistRepository(ctrl)
	repo.EXPECT().ListByUser(gomock.Any(), "u1").Return([]domain.Watchlist{
		{Symbol: "BBCA", Label: "Bank"},
		{Symbol: "TLKM", Label: "Telco"},
	}, nil)

	uc := NewWatchlistUsecase(repo)
	items, err := uc.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "BBCA", items[0].Symbol)
}
