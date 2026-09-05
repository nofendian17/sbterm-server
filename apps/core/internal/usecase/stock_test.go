package usecase

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// mockLogger is a minimal no-op log.Logger for usecase tests.
type mockLogger struct{}

func (mockLogger) Debug(msg string, args ...any)                          {}
func (mockLogger) DebugContext(ctx context.Context, msg string, a ...any) {}
func (mockLogger) Info(msg string, args ...any)                           {}
func (mockLogger) InfoContext(ctx context.Context, msg string, a ...any)  {}
func (mockLogger) Warn(msg string, args ...any)                           {}
func (mockLogger) WarnContext(ctx context.Context, msg string, a ...any)  {}
func (mockLogger) Error(msg string, args ...any)                          {}
func (mockLogger) ErrorContext(ctx context.Context, msg string, a ...any) {}
func (mockLogger) With(args ...any) log.Logger                            { return mockLogger{} }
func (mockLogger) WithGroup(name string) log.Logger                       { return mockLogger{} }
func (mockLogger) Enabled(ctx context.Context, level log.Level) bool      { return true }
func (mockLogger) Slog() *slog.Logger                                     { return slog.New(slog.DiscardHandler) }

func newTestStockUsecase(ctrl *gomock.Controller) (
	*mocks.MockStockRepository,
	*mocks.MockSectorRepository,
	*mocks.MockStockSyncClient,
	StockUsecase,
) {
	repo := mocks.NewMockStockRepository(ctrl)
	sectors := mocks.NewMockSectorRepository(ctrl)
	sync := mocks.NewMockStockSyncClient(ctrl)
	uc := NewStockUsecase(repo, sectors, sync, mockLogger{})
	return repo, sectors, sync, uc
}

func TestStockUsecase_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, _, _, uc := newTestStockUsecase(ctrl)

	repo.EXPECT().List(gomock.Any(), domain.StockFilter{Query: "BB"}).
		Return([]domain.Stock{{Symbol: "BBCA"}}, 1, nil)

	stocks, total, err := uc.List(context.Background(), domain.StockFilter{Query: "BB"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, stocks, 1)
}

func TestStockUsecase_Create_Duplicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, _, _, uc := newTestStockUsecase(ctrl)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(domain.ErrStockSymbolTaken)

	_, err := uc.Create(context.Background(), domain.StockCreateInput{
		Symbol: "BBCA", Name: "Bank Central Asia",
	})
	assert.ErrorIs(t, err, domain.ErrStockSymbolTaken)
}

func TestStockUsecase_Create_SuccessReloads(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, _, _, uc := newTestStockUsecase(ctrl)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().GetBySymbol(gomock.Any(), "BBCA").Return(domain.Stock{
		Symbol:    "BBCA",
		Name:      "Bank Central Asia",
		IsActive:  true,
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
	}, nil)

	got, err := uc.Create(context.Background(), domain.StockCreateInput{
		Symbol: "BBCA", Name: "Bank Central Asia",
	})
	require.NoError(t, err)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestStockUsecase_Create_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, _, _, uc := newTestStockUsecase(ctrl)

	_, err := uc.Create(context.Background(), domain.StockCreateInput{Symbol: "", Name: ""})
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestStockUsecase_Create_UnknownSector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, sectors, _, uc := newTestStockUsecase(ctrl)
	sector := "Nope"
	sectors.EXPECT().GetByName(gomock.Any(), "Nope").
		Return(domain.Sector{}, domain.ErrSectorNotFound)

	_, err := uc.Create(context.Background(), domain.StockCreateInput{
		Symbol: "BBCA", Name: "Bank Central Asia", Sector: &sector,
	})
	assert.ErrorIs(t, err, domain.ErrSectorNotFound)
}

func TestStockUsecase_Update_ResolvesSector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, sectors, _, uc := newTestStockUsecase(ctrl)

	sector := "Financials"
	sectors.EXPECT().GetByName(gomock.Any(), "Financials").
		Return(domain.Sector{ID: "s1", Name: "Financials"}, nil)
	repo.EXPECT().Update(gomock.Any(), "BBCA", gomock.Any()).Return(nil)

	err := uc.Update(context.Background(), "bbcA", domain.StockUpdateInput{Sector: &sector})
	require.NoError(t, err)
}

func TestStockUsecase_SoftDelete_BlockedByWatchlists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, _, _, uc := newTestStockUsecase(ctrl)

	repo.EXPECT().SoftDelete(gomock.Any(), "BBCA").
		Return(domain.ErrStockHasWatchlists)

	err := uc.SoftDelete(context.Background(), "bbca")
	assert.ErrorIs(t, err, domain.ErrStockHasWatchlists)
}

func TestStockUsecase_SyncAll_MixedResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, _, sync, uc := newTestStockUsecase(ctrl)

	sync.EXPECT().ListSymbols(gomock.Any()).Return([]domain.Stock{
		{Symbol: "A", Name: "A Inc"},
		{Symbol: "B", Name: "B Inc"},
		{Symbol: "C", Name: "C Inc"},
	}, nil)
	gomock.InOrder(
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(true, nil),
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(false, nil),
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(false, errors.New("dup key")),
	)

	res, err := uc.SyncAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, res.Fetched)
	assert.Equal(t, 1, res.Created)
	assert.Equal(t, 1, res.Updated)
	assert.Equal(t, 1, res.Failed)
	assert.Len(t, res.Errors, 1)
}

func TestStockUsecase_SyncAll_UpstreamError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, _, sync, uc := newTestStockUsecase(ctrl)

	sync.EXPECT().ListSymbols(gomock.Any()).Return(nil, errors.New("boom"))

	_, err := uc.SyncAll(context.Background())
	assert.Error(t, err)
}
