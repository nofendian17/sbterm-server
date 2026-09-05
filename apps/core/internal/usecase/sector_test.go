package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func TestSectorUsecase_Create_TrimsAndRejectsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockSectorRepository(ctrl)
	uc := NewSectorUsecase(repo)

	_, err := uc.Create(context.Background(), "   ")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	repo.EXPECT().Create(gomock.Any(), "Financials").
		Return(domain.Sector{ID: "s1", Name: "Financials"}, nil)
	got, err := uc.Create(context.Background(), "  Financials ")
	require.NoError(t, err)
	assert.Equal(t, "s1", got.ID)
}

func TestSectorUsecase_SoftDelete_BlockedByStocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockSectorRepository(ctrl)
	uc := NewSectorUsecase(repo)

	repo.EXPECT().SoftDelete(gomock.Any(), "s1").
		Return(domain.ErrSectorHasStocks)
	err := uc.SoftDelete(context.Background(), "s1")
	assert.ErrorIs(t, err, domain.ErrSectorHasStocks)
}
