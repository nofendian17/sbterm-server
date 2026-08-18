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

func TestSectorsUsecaseGetSectors(t *testing.T) {
	sectors := []domain.Sector{{Symbol: "IDXCYCLIC", ID: "1000003293"}}
	companies := []domain.SubsectorCompany{{Symbol: "MNCN", Name: "Media Nusantara Citra Tbk."}}

	tests := []struct {
		name            string
		sectorsErr      error
		companiesErr    error
		wantErr         bool
		wantNestedCount int
	}{
		{name: "returns sectors with nested companies", wantNestedCount: 1},
		{name: "propagates sectors repository error", sectorsErr: errors.New("boom"), wantErr: true},
		{name: "propagates subsector repository error", companiesErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sectorsRepo := mocks.NewMockSectorsRepository(ctrl)
			sectorsRepo.EXPECT().GetSectors(gomock.Any()).Return(sectors, tt.sectorsErr)

			subsectorRepo := mocks.NewMockSubsectorRepository(ctrl)
			if tt.sectorsErr == nil {
				subsectorRepo.EXPECT().GetCompanies(gomock.Any(), "70", "1000003293").Return(companies, tt.companiesErr)
			}

			uc := NewSectorsUsecase(sectorsRepo, subsectorRepo)
			got, err := uc.GetSectors(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "IDXCYCLIC", got[0].Symbol)
			require.Len(t, got[0].Companies, tt.wantNestedCount)
			assert.Equal(t, "MNCN", got[0].Companies[0].Symbol)
		})
	}
}
