package orderbook

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type OrderBookHandler struct {
	uc usecase.OrderBookUsecase
	v  validator.Validator
}

func NewOrderBookHandler(uc usecase.OrderBookUsecase, v validator.Validator) *OrderBookHandler {
	return &OrderBookHandler{uc: uc, v: v}
}

type orderBookRequest struct {
	Symbol string `json:"symbol" validate:"required"`
}

type orderBookResponse struct {
	Average                      int                     `json:"average"`
	Bid                          []orderBookLevelResp    `json:"bid"`
	Change                       int                     `json:"change"`
	Close                        int                     `json:"close"`
	Country                      string                  `json:"country"`
	Domestic                     string                  `json:"domestic"`
	Down                         string                  `json:"down"`
	Exchange                     string                  `json:"exchange"`
	FBuy                         int64                   `json:"fbuy"`
	FNet                         int64                   `json:"fnet"`
	Foreign                      string                  `json:"foreign"`
	Frequency                    int64                   `json:"frequency"`
	FSell                        int64                   `json:"fsell"`
	High                         int                     `json:"high"`
	ID                           string                  `json:"id"`
	LastPrice                    int                     `json:"lastprice"`
	Low                          int                     `json:"low"`
	Offer                        []orderBookLevelResp    `json:"offer"`
	Open                         int                     `json:"open"`
	PercentageChange             float64                 `json:"percentage_change"`
	Previous                     int                     `json:"previous"`
	Status                       string                  `json:"status"`
	Symbol                       string                  `json:"symbol"`
	Symbol2                      string                  `json:"symbol_2"`
	Symbol3                      string                  `json:"symbol_3"`
	Tradable                     bool                    `json:"tradable"`
	Unchanged                    string                  `json:"unchanged"`
	Up                           string                  `json:"up"`
	Value                        int64                   `json:"value"`
	Volume                       int64                   `json:"volume"`
	CorpAction                   orderBookCorpActionResp `json:"corp_action"`
	Notation                     []json.RawMessage       `json:"notation"`
	UMA                          bool                    `json:"uma"`
	HasForeignBS                 bool                    `json:"has_foreign_bs"`
	IEPIEV                       json.RawMessage         `json:"iepiev"`
	MarketData                   []orderBookMarketResp   `json:"market_data"`
	Name                         string                  `json:"name"`
	IconURL                      string                  `json:"icon_url"`
	ARA                          orderBookLimitResp      `json:"ara"`
	ARB                          orderBookLimitResp      `json:"arb"`
	CompanyType                  string                  `json:"company_type"`
	TotalBidOffer                orderBookTotalResp      `json:"total_bid_offer"`
	NextARA                      orderBookLimitResp      `json:"next_ara"`
	NextARB                      orderBookLimitResp      `json:"next_arb"`
	AutoRejectTimeLeftInSec      int                     `json:"autoreject_time_left_in_sec"`
	AutoRejectEstimation         []json.RawMessage       `json:"auto_reject_estimation"`
	OrderbookActiveFeatureMobile string                  `json:"orderbook_active_feature_mobile"`
}

type orderBookLevelResp struct {
	Price            string `json:"price"`
	QueNum           string `json:"que_num"`
	Volume           string `json:"volume"`
	ChangePercentage string `json:"change_percentage"`
}

type orderBookCorpActionResp struct {
	Active bool   `json:"active"`
	Icon   string `json:"icon"`
	Text   string `json:"text"`
}

type orderBookLimitResp struct {
	Value   string `json:"value"`
	Visible bool   `json:"visible"`
}

type orderBookMarketResp struct {
	Label     string           `json:"label"`
	Frequency rawFormattedResp `json:"frequency"`
	Volume    rawFormattedResp `json:"volume"`
	Value     rawFormattedResp `json:"value"`
}

type rawFormattedResp struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

type orderBookTotalResp struct {
	Bid        orderBookSideResp `json:"bid"`
	Offer      orderBookSideResp `json:"offer"`
	BidPercent float64           `json:"bid_percent"`
}

type orderBookSideResp struct {
	Freq    string `json:"freq"`
	Lot     string `json:"lot"`
	RawLot  string `json:"raw_lot"`
	RawFreq string `json:"raw_freq"`
}

func (h *OrderBookHandler) OrderBook(w http.ResponseWriter, r *http.Request) {
	req := orderBookRequest{Symbol: chi.URLParam(r, "symbol")}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate order book params")
		return
	}

	data, err := h.uc.GetOrderBook(r.Context(), req.Symbol)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no order book data for the requested symbol")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get order book")
		return
	}
	response.OK(w, toOrderBookResponse(data))
}

func toOrderBookResponse(d *domain.OrderBookData) orderBookResponse {
	out := orderBookResponse{
		Average:                      d.Average,
		Change:                       d.Change,
		Close:                        d.Close,
		Country:                      d.Country,
		Domestic:                     d.Domestic,
		Down:                         d.Down,
		Exchange:                     d.Exchange,
		FBuy:                         d.FBuy,
		FNet:                         d.FNet,
		Foreign:                      d.Foreign,
		Frequency:                    d.Frequency,
		FSell:                        d.FSell,
		High:                         d.High,
		ID:                           d.ID,
		LastPrice:                    d.LastPrice,
		Low:                          d.Low,
		Open:                         d.Open,
		PercentageChange:             d.PercentageChange,
		Previous:                     d.Previous,
		Status:                       d.Status,
		Symbol:                       d.Symbol,
		Symbol2:                      d.Symbol2,
		Symbol3:                      d.Symbol3,
		Tradable:                     d.Tradable,
		Unchanged:                    d.Unchanged,
		Up:                           d.Up,
		Value:                        d.Value,
		Volume:                       d.Volume,
		CorpAction:                   orderBookCorpActionResp{Active: d.CorpAction.Active, Icon: d.CorpAction.Icon, Text: d.CorpAction.Text},
		Notation:                     d.Notation,
		UMA:                          d.UMA,
		HasForeignBS:                 d.HasForeignBS,
		IEPIEV:                       d.IEPIEV,
		MarketData:                   toOrderBookMarkets(d.MarketData),
		Name:                         d.Name,
		IconURL:                      d.IconURL,
		ARA:                          orderBookLimitResp{Value: d.ARA.Value, Visible: d.ARA.Visible},
		ARB:                          orderBookLimitResp{Value: d.ARB.Value, Visible: d.ARB.Visible},
		CompanyType:                  d.CompanyType,
		TotalBidOffer:                orderBookTotalResp{Bid: toOrderBookSide(d.TotalBidOffer.Bid), Offer: toOrderBookSide(d.TotalBidOffer.Offer), BidPercent: d.TotalBidOffer.BidPercent},
		NextARA:                      orderBookLimitResp{Value: d.NextARA.Value, Visible: d.NextARA.Visible},
		NextARB:                      orderBookLimitResp{Value: d.NextARB.Value, Visible: d.NextARB.Visible},
		AutoRejectTimeLeftInSec:      d.AutoRejectTimeLeftInSec,
		AutoRejectEstimation:         d.AutoRejectEstimation,
		OrderbookActiveFeatureMobile: d.OrderbookActiveFeatureMobile,
	}
	out.Bid = make([]orderBookLevelResp, 0, len(d.Bid))
	for _, l := range d.Bid {
		out.Bid = append(out.Bid, toOrderBookLevel(l))
	}
	out.Offer = make([]orderBookLevelResp, 0, len(d.Offer))
	for _, l := range d.Offer {
		out.Offer = append(out.Offer, toOrderBookLevel(l))
	}
	return out
}

func toOrderBookLevel(l domain.OrderBookLevel) orderBookLevelResp {
	return orderBookLevelResp{Price: l.Price, QueNum: l.QueNum, Volume: l.Volume, ChangePercentage: l.ChangePercentage}
}

func toOrderBookMarkets(in []domain.OrderBookMarket) []orderBookMarketResp {
	out := make([]orderBookMarketResp, 0, len(in))
	for _, m := range in {
		out = append(out, orderBookMarketResp{
			Label:     m.Label,
			Frequency: rawFormattedResp{Raw: m.Frequency.Raw, Formatted: m.Frequency.Formatted},
			Volume:    rawFormattedResp{Raw: m.Volume.Raw, Formatted: m.Volume.Formatted},
			Value:     rawFormattedResp{Raw: m.Value.Raw, Formatted: m.Value.Formatted},
		})
	}
	return out
}

func toOrderBookSide(s domain.OrderBookSide) orderBookSideResp {
	return orderBookSideResp{Freq: s.Freq, Lot: s.Lot, RawLot: s.RawLot, RawFreq: s.RawFreq}
}
