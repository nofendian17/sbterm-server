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

func TestSubsidiaryUsecaseGetSubsidiaries(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns subsidiaries"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.SubsidiaryData{
				Currency:          "CURRENCY_USD",
				LastUpdatedPeriod: "Q1 2026",
				Unit:              "UNIT_FULL",
				Subsidiaries:      []domain.Subsidiary{{CompanyName: "PT DSST Mas Gemilang", Percentage: "99.99"}},
			}
			repo := mocks.NewMockSubsidiaryRepository(ctrl)
			repo.EXPECT().GetSubsidiaries(gomock.Any(), "DSSA").Return(want, tt.repoErr)

			uc := NewSubsidiaryUsecase(repo)
			got, err := uc.GetSubsidiaries(context.Background(), "DSSA")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
