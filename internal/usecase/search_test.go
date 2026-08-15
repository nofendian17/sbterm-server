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

func TestSearchUsecaseGetSearch(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns search result"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.SearchResult{
				Company: []domain.SearchCompany{{ID: "59", Name: "BBRI"}},
			}
			repo := mocks.NewMockSearchRepository(ctrl)
			repo.EXPECT().GetSearch(gomock.Any(), "BBRI", 1, "company").Return(want, tt.repoErr)

			uc := NewSearchUsecase(repo)
			got, err := uc.GetSearch(context.Background(), "BBRI", 1, "company")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
