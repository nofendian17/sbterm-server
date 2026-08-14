package marketdetector

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

// defaultFilter values applied when the corresponding query param is omitted.
const (
	defaultTransactionType = "TRANSACTION_TYPE_NET"
	defaultMarketBoard     = "MARKET_BOARD_REGULER"
	defaultInvestorType    = "INVESTOR_TYPE_ALL"
)

type MarketDetectorHandler struct {
	uc usecase.MarketDetectorUsecase
	v  validator.Validator
}

func NewMarketDetectorHandler(uc usecase.MarketDetectorUsecase, v validator.Validator) *MarketDetectorHandler {
	return &MarketDetectorHandler{uc: uc, v: v}
}

type marketDetectorRequest struct {
	Symbol          string `json:"symbol" validate:"required"`
	From            string `json:"from" validate:"required,datetime=2006-01-02"`
	To              string `json:"to" validate:"required,datetime=2006-01-02"`
	TransactionType string `json:"transaction_type" validate:"omitempty,oneof=TRANSACTION_TYPE_GROSS TRANSACTION_TYPE_NET"`
	MarketBoard     string `json:"market_board" validate:"omitempty,oneof=MARKET_BOARD_ALL MARKET_BOARD_REGULER MARKET_BOARD_TUNAI MARKET_BOARD_NEGO"`
	InvestorType    string `json:"investor_type" validate:"omitempty,oneof=INVESTOR_TYPE_ALL INVESTOR_TYPE_DOMESTIC INVESTOR_TYPE_FOREIGN"`
	Limit           int    `json:"limit" validate:"omitempty,min=1"`
}

type marketDetectorResponse struct {
	Symbol         string             `json:"symbol"`
	From           string             `json:"from"`
	To             string             `json:"to"`
	BandarDetector bandarDetectorResp `json:"bandar_detector"`
	BrokerSummary  brokerSummaryResp  `json:"broker_summary"`
}

type bandarDetectorResp struct {
	Average             float64           `json:"average"`
	Avg                 bandarAccdistResp `json:"avg"`
	Avg5                bandarAccdistResp `json:"avg5"`
	BrokerAccdist       string            `json:"broker_accdist"`
	NumberBrokerBuysell int               `json:"number_broker_buysell"`
	Top1                bandarAccdistResp `json:"top1"`
	Top3                bandarAccdistResp `json:"top3"`
	Top5                bandarAccdistResp `json:"top5"`
	Top10               bandarAccdistResp `json:"top10"`
	TotalBuyer          int               `json:"total_buyer"`
	TotalSeller         int               `json:"total_seller"`
	Value               int64             `json:"value"`
	Volume              int64             `json:"volume"`
}

type bandarAccdistResp struct {
	Accdist string  `json:"accdist"`
	Amount  int64   `json:"amount"`
	Percent float64 `json:"percent"`
	Vol     float64 `json:"vol"`
}

type brokerSummaryResp struct {
	BrokersBuy  []brokerBuyResp  `json:"brokers_buy"`
	BrokersSell []brokerSellResp `json:"brokers_sell"`
}

type brokerBuyResp struct {
	Blot             string `json:"blot"`
	Blotv            string `json:"blotv"`
	Bval             string `json:"bval"`
	Bvalv            string `json:"bvalv"`
	NetbsBrokerCode  string `json:"netbs_broker_code"`
	NetbsBuyAvgPrice string `json:"netbs_buy_avg_price"`
	NetbsDate        string `json:"netbs_date"`
	NetbsStockCode   string `json:"netbs_stock_code"`
	Type             string `json:"type"`
	Freq             string `json:"freq"`
}

type brokerSellResp struct {
	NetbsBrokerCode   string `json:"netbs_broker_code"`
	NetbsDate         string `json:"netbs_date"`
	NetbsSellAvgPrice string `json:"netbs_sell_avg_price"`
	NetbsStockCode    string `json:"netbs_stock_code"`
	Slot              string `json:"slot"`
	Slotv             string `json:"slotv"`
	Sval              string `json:"sval"`
	Svalv             string `json:"svalv"`
	Type              string `json:"type"`
	Freq              string `json:"freq"`
}

func (h *MarketDetectorHandler) MarketDetector(w http.ResponseWriter, r *http.Request) {
	req := marketDetectorRequest{
		Symbol:          chi.URLParam(r, "symbol"),
		From:            r.URL.Query().Get("from"),
		To:              r.URL.Query().Get("to"),
		TransactionType: r.URL.Query().Get("transaction_type"),
		MarketBoard:     r.URL.Query().Get("market_board"),
		InvestorType:    r.URL.Query().Get("investor_type"),
	}
	if req.TransactionType == "" {
		req.TransactionType = defaultTransactionType
	}
	if req.MarketBoard == "" {
		req.MarketBoard = defaultMarketBoard
	}
	if req.InvestorType == "" {
		req.InvestorType = defaultInvestorType
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			response.ValidationError(w, "validation failed", map[string]string{"limit": "must be a valid integer"})
			return
		}
		req.Limit = n
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate market detector params")
		return
	}

	data, err := h.uc.GetMarketDetector(r.Context(), req.Symbol, req.From, req.To, req.TransactionType, req.MarketBoard, req.InvestorType, req.Limit)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no market detector data for the requested date range")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get market detector data")
		return
	}
	response.OK(w, toResponse(req.Symbol, data))
}

func toResponse(symbol string, d *domain.MarketDetectorData) marketDetectorResponse {
	bd := d.BandarDetector
	return marketDetectorResponse{
		Symbol: symbol,
		From:   d.From,
		To:     d.To,
		BandarDetector: bandarDetectorResp{
			Average:             bd.Average,
			Avg:                 toBandarAccdistResp(bd.Avg),
			Avg5:                toBandarAccdistResp(bd.Avg5),
			BrokerAccdist:       bd.BrokerAccdist,
			NumberBrokerBuysell: bd.NumberBrokerBuysell,
			Top1:                toBandarAccdistResp(bd.Top1),
			Top3:                toBandarAccdistResp(bd.Top3),
			Top5:                toBandarAccdistResp(bd.Top5),
			Top10:               toBandarAccdistResp(bd.Top10),
			TotalBuyer:          bd.TotalBuyer,
			TotalSeller:         bd.TotalSeller,
			Value:               bd.Value,
			Volume:              bd.Volume,
		},
		BrokerSummary: brokerSummaryResp{
			BrokersBuy:  toBrokerBuyResps(d.BrokerSummary.BrokersBuy),
			BrokersSell: toBrokerSellResps(d.BrokerSummary.BrokersSell),
		},
	}
}

func toBandarAccdistResp(a domain.BandarAccdist) bandarAccdistResp {
	return bandarAccdistResp{Accdist: a.Accdist, Amount: a.Amount, Percent: a.Percent, Vol: a.Vol}
}

func toBrokerBuyResps(items []domain.BrokerBuy) []brokerBuyResp {
	out := make([]brokerBuyResp, 0, len(items))
	for _, b := range items {
		out = append(out, brokerBuyResp{
			Blot:             b.Blot,
			Blotv:            b.Blotv,
			Bval:             b.Bval,
			Bvalv:            b.Bvalv,
			NetbsBrokerCode:  b.NetbsBrokerCode,
			NetbsBuyAvgPrice: b.NetbsBuyAvgPrice,
			NetbsDate:        b.NetbsDate,
			NetbsStockCode:   b.NetbsStockCode,
			Type:             b.Type,
			Freq:             b.Freq,
		})
	}
	return out
}

func toBrokerSellResps(items []domain.BrokerSell) []brokerSellResp {
	out := make([]brokerSellResp, 0, len(items))
	for _, b := range items {
		out = append(out, brokerSellResp{
			NetbsBrokerCode:   b.NetbsBrokerCode,
			NetbsDate:         b.NetbsDate,
			NetbsSellAvgPrice: b.NetbsSellAvgPrice,
			NetbsStockCode:    b.NetbsStockCode,
			Slot:              b.Slot,
			Slotv:             b.Slotv,
			Sval:              b.Sval,
			Svalv:             b.Svalv,
			Type:              b.Type,
			Freq:              b.Freq,
		})
	}
	return out
}
