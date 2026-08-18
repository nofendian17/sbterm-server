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

func TestShareholdingCompositionUsecaseGetShareholdingComposition(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns composition periods"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := []domain.ShareholdingCompositionPeriod{{
				ReportDate:   "2026-07-31",
				Compositions: []domain.ShareholdingComposition{{Label: "SINAR MAS TUNGGAL"}},
			}}
			repo := mocks.NewMockShareholdingCompositionRepository(ctrl)
			repo.EXPECT().GetShareholdingComposition(gomock.Any(), "DSSA", "", "").Return(want, tt.repoErr)

			uc := NewShareholdingCompositionUsecase(repo)
			got, err := uc.GetShareholdingComposition(context.Background(), "DSSA", "", "")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
