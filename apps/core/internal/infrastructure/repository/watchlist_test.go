package repository

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

func TestWatchlistRepository_ListByUser(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		setup  func(mock pgxmock.PgxPoolIface)
		want   []domain.Watchlist
	}{
		{
			name:   "returns items",
			userID: "u1",
			setup: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{"id", "user_id", "symbol", "label", "created_at"}).
					AddRow("w1", "u1", "BBCA", "Bank", now).
					AddRow("w2", "u1", "TLKM", "Telco", now)
				mock.ExpectQuery(`SELECT id, user_id, symbol, label, created_at`).WithArgs("u1").WillReturnRows(rows)
			},
			want: nil, // we check length and fields separately
		},
		{
			name:   "empty",
			userID: "u2",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "user_id", "symbol", "label", "created_at"})
				mock.ExpectQuery(`SELECT id, user_id, symbol, label, created_at`).WithArgs("u2").WillReturnRows(rows)
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewWatchlistRepository(mock)
			got, err := repo.ListByUser(context.Background(), tt.userID)
			require.NoError(t, err)
			if tt.name == "empty" {
				assert.Nil(t, got)
			} else {
				require.Len(t, got, 2)
				assert.Equal(t, "w1", got[0].ID)
				assert.Equal(t, "u1", got[0].UserID)
				assert.Equal(t, "BBCA", got[0].Symbol)
				assert.Equal(t, "Bank", got[0].Label)
				assert.Equal(t, "w2", got[1].ID)
				assert.Equal(t, "TLKM", got[1].Symbol)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWatchlistRepository_Add(t *testing.T) {
	tests := []struct {
		name    string
		w       domain.Watchlist
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr error
	}{
		{
			name: "success",
			w:    domain.Watchlist{UserID: "u1", Symbol: "BBCA", Label: "Bank"},
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO watchlists`).
					WithArgs("u1", "BBCA", "Bank").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
		},
		{
			name: "duplicate symbol",
			w:    domain.Watchlist{UserID: "u1", Symbol: "BBCA", Label: "Bank"},
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO watchlists`).
					WithArgs("u1", "BBCA", "Bank").
					WillReturnError(pgUniqueViolation())
			},
			wantErr: domain.ErrDuplicateWatchlist,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewWatchlistRepository(mock)
			err := repo.Add(context.Background(), tt.w)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWatchlistRepository_RemoveBySymbol(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`DELETE FROM watchlists`).
		WithArgs("u1", "BBCA").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewWatchlistRepository(mock)
	assert.NoError(t, repo.RemoveBySymbol(context.Background(), "u1", "BBCA"))
	assert.NoError(t, mock.ExpectationsWereMet())
}
