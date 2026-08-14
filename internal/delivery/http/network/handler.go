package network

import (
	"net/http"
	"strconv"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

type ShareholdingNetworkHandler struct {
	uc usecase.ShareholdingNetworkUsecase
	v  validator.Validator
}

func NewShareholdingNetworkHandler(uc usecase.ShareholdingNetworkUsecase, v validator.Validator) *ShareholdingNetworkHandler {
	return &ShareholdingNetworkHandler{uc: uc, v: v}
}

type networkRequest struct {
	RootID         string `json:"root_id" validate:"required"`
	RootType       string `json:"root_type" validate:"required"`
	MaxDepth       int    `json:"max_depth" validate:"omitempty,min=1"`
	MaxEdgePerNode int    `json:"max_edge_per_node" validate:"omitempty,min=1"`
}

type shareholdingNetworkResponse struct {
	RootID     string         `json:"root_id"`
	RootType   string         `json:"root_type"`
	ReportDate string         `json:"report_date"`
	Nodes      []nodeResponse `json:"nodes"`
	Edges      []edgeResponse `json:"edges"`
}

type nodeResponse struct {
	ID         string           `json:"id"`
	NodeType   string           `json:"node_type"`
	Metadata   metadataResponse `json:"metadata"`
	MinDepth   int              `json:"min_depth"`
	IsRendered bool             `json:"is_rendered"`
}

type metadataResponse struct {
	Company  *companyResponse  `json:"company"`
	Investor *investorResponse `json:"investor"`
}

type companyResponse struct {
	ID      int64  `json:"id"`
	Symbol  string `json:"symbol"`
	Name    string `json:"name"`
	IconURL string `json:"icon_url"`
}

type investorResponse struct {
	ID                     int64                `json:"id"`
	Name                   string               `json:"name"`
	InvestorType           rawFormattedResponse `json:"investor_type"`
	Location               rawFormattedResponse `json:"location"`
	Nationality            rawFormattedResponse `json:"nationality"`
	Domicile               rawFormattedResponse `json:"domicile"`
	InvestorClassification rawFormattedResponse `json:"investor_classification"`
}

type rawFormattedResponse struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

type edgeResponse struct {
	FromID       string          `json:"from_id"`
	ToID         string          `json:"to_id"`
	Shareholding holdingResponse `json:"shareholding"`
	IsRendered   bool            `json:"is_rendered"`
}

type holdingResponse struct {
	Scripless   rawFormattedResponse `json:"scripless"`
	Scrip       rawFormattedResponse `json:"scrip"`
	TotalShares rawFormattedResponse `json:"total_shares"`
	Percentage  percentResponse      `json:"percentage"`
}

type percentResponse struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

func parseNetworkIntParam(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	if v := r.URL.Query().Get(name); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			response.ValidationError(w, "validation failed", map[string]string{name: "must be a valid integer"})
			return 0, false
		}
		return n, true
	}
	return 0, true
}

func (h *ShareholdingNetworkHandler) ShareholdingNetwork(w http.ResponseWriter, r *http.Request) {
	req := networkRequest{
		RootID:   r.URL.Query().Get("root_id"),
		RootType: r.URL.Query().Get("root_type"),
	}
	var ok bool
	if req.MaxDepth, ok = parseNetworkIntParam(w, r, "max_depth"); !ok {
		return
	}
	if req.MaxEdgePerNode, ok = parseNetworkIntParam(w, r, "max_edge_per_node"); !ok {
		return
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate shareholding network params")
		return
	}

	network, err := h.uc.GetShareholdingNetwork(r.Context(), req.RootID, req.RootType, req.MaxDepth, req.MaxEdgePerNode)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get shareholding network")
		return
	}
	response.OK(w, toResponse(network))
}

func toResponse(n *domain.ShareholdingNetwork) shareholdingNetworkResponse {
	out := shareholdingNetworkResponse{
		RootID:     n.RootID,
		RootType:   n.RootType,
		ReportDate: n.ReportDate,
		Nodes:      make([]nodeResponse, 0, len(n.Nodes)),
		Edges:      make([]edgeResponse, 0, len(n.Edges)),
	}
	for _, nd := range n.Nodes {
		nr := nodeResponse{ID: nd.ID, NodeType: nd.NodeType, MinDepth: nd.MinDepth, IsRendered: nd.IsRendered}
		if nd.Metadata.Company != nil {
			nr.Metadata.Company = &companyResponse{
				ID:      nd.Metadata.Company.ID,
				Symbol:  nd.Metadata.Company.Symbol,
				Name:    nd.Metadata.Company.Name,
				IconURL: nd.Metadata.Company.IconURL,
			}
		}
		if nd.Metadata.Investor != nil {
			nr.Metadata.Investor = &investorResponse{
				ID:                     nd.Metadata.Investor.ID,
				Name:                   nd.Metadata.Investor.Name,
				InvestorType:           toRawFormattedResponse(nd.Metadata.Investor.InvestorType),
				Location:               toRawFormattedResponse(nd.Metadata.Investor.Location),
				Nationality:            toRawFormattedResponse(nd.Metadata.Investor.Nationality),
				Domicile:               toRawFormattedResponse(nd.Metadata.Investor.Domicile),
				InvestorClassification: toRawFormattedResponse(nd.Metadata.Investor.InvestorClassification),
			}
		}
		out.Nodes = append(out.Nodes, nr)
	}
	for _, e := range n.Edges {
		out.Edges = append(out.Edges, edgeResponse{
			FromID: e.FromID,
			ToID:   e.ToID,
			Shareholding: holdingResponse{
				Scripless:   toRawFormattedResponse(e.Shareholding.Scripless),
				Scrip:       toRawFormattedResponse(e.Shareholding.Scrip),
				TotalShares: toRawFormattedResponse(e.Shareholding.TotalShares),
				Percentage:  percentResponse{Raw: e.Shareholding.Percentage.Raw, Formatted: e.Shareholding.Percentage.Formatted},
			},
			IsRendered: e.IsRendered,
		})
	}
	return out
}

func toRawFormattedResponse(in domain.ShareholdingRawFormatted) rawFormattedResponse {
	return rawFormattedResponse{Raw: in.Raw, Formatted: in.Formatted}
}
