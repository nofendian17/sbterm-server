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

func TestCompanyProfileUsecaseGetProfile(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns profile"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.CompanyProfile{Background: "PT Dian Swastatika", History: &domain.ProfileHistory{Date: "10 Dec 2009"}}
			repo := mocks.NewMockCompanyProfileRepository(ctrl)
			repo.EXPECT().GetProfile(gomock.Any(), "DSSA").Return(want, tt.repoErr)

			uc := NewCompanyProfileUsecase(repo)
			got, err := uc.GetProfile(context.Background(), "DSSA")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
