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

func newTestCompanyProfileUsecase(ctrl *gomock.Controller) (
	*mocks.MockCompanyProfileRepository,
	*mocks.MockCompanyProfileSyncClient,
	CompanyProfileUsecase,
) {
	repo := mocks.NewMockCompanyProfileRepository(ctrl)
	sync := mocks.NewMockCompanyProfileSyncClient(ctrl)
	uc := NewCompanyProfileUsecase(repo, sync)
	return repo, sync, uc
}

func TestCompanyProfileUsecase_SyncProfile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, sync, uc := newTestCompanyProfileUsecase(ctrl)

	upstream := domain.CompanyProfile{
		Symbol: "BBCA",
		Board:  strPtr("Papan Utama"),
		Executives: []domain.CompanyExecutive{
			{Kind: "commissioner", Name: "TONNY KUSNADI"},
		},
	}
	saved := upstream
	saved.Subsidiaries = []domain.CompanySubsidiary{{Name: "PT Bank Digital BCA"}}

	sync.EXPECT().FetchCompanyProfile(gomock.Any(), "BBCA").Return(upstream, nil)
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().GetBySymbol(gomock.Any(), "BBCA").Return(saved, nil)

	got, err := uc.SyncProfile(context.Background(), "BBCA")
	require.NoError(t, err)
	assert.Equal(t, "BBCA", got.Symbol)
	assert.Len(t, got.Subsidiaries, 1)
}

func TestCompanyProfileUsecase_SyncProfile_InvalidSymbol(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, _, uc := newTestCompanyProfileUsecase(ctrl)

	_, err := uc.SyncProfile(context.Background(), "  ")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestCompanyProfileUsecase_SyncProfile_UpstreamError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, sync, uc := newTestCompanyProfileUsecase(ctrl)

	sync.EXPECT().FetchCompanyProfile(gomock.Any(), "BBCA").
		Return(domain.CompanyProfile{}, errors.New("upstream boom"))

	_, err := uc.SyncProfile(context.Background(), "BBCA")
	assert.ErrorIs(t, err, domain.ErrCompanyProfileSyncFailed)
}

func TestCompanyProfileUsecase_SyncProfile_SaveStockNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, sync, uc := newTestCompanyProfileUsecase(ctrl)

	sync.EXPECT().FetchCompanyProfile(gomock.Any(), "ZZZZ").
		Return(domain.CompanyProfile{Symbol: "ZZZZ"}, nil)
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).
		Return(domain.ErrStockNotFound)

	_, err := uc.SyncProfile(context.Background(), "ZZZZ")
	assert.ErrorIs(t, err, domain.ErrStockNotFound)
}

func TestCompanyProfileUsecase_SyncProfile_NilSyncClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockCompanyProfileRepository(ctrl)
	uc := NewCompanyProfileUsecase(repo, nil)

	_, err := uc.SyncProfile(context.Background(), "BBCA")
	assert.ErrorIs(t, err, domain.ErrCompanyProfileSyncFailed)
}

func strPtr(s string) *string { return &s }
