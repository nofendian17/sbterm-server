package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
)

func TestFindataFinancialUsecaseGetFindataFinancial(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns financial report"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.FindataFinancial{
				DefaultCurrency: "IDR",
				DataTables: domain.FindataDataTables{
					Periods:  []string{"12M 2025"},
					Accounts: []domain.FindataAccount{{ID: 190, Name: "Arus Kas Dari Aktivitas Operasi"}},
				},
			}
			repo := mocks.NewMockFindataFinancialRepository(ctrl)
			repo.EXPECT().GetFindataFinancial(gomock.Any(), "BRPT", 1, 0, 1, 3, 2).Return(want, tt.repoErr)

			uc := NewFindataFinancialUsecase(repo)
			got, err := uc.GetFindataFinancial(context.Background(), "BRPT", 1, 0, 1, 3, 2)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
