package stockapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleProfilePayload = `{
  "success": true,
  "data": {
    "background": "PT Bank Central Asia Tbk.",
    "history": {
      "amount": "927 B",
      "board": "Papan Utama",
      "date": "31 May 2000",
      "price": "1,400",
      "registrar": "BAE",
      "shares": "662,400,000",
      "free_float": "42.46%",
      "underwriters": ["PT. Danareksa Sekuritas"]
    },
    "key_executive": {
      "commissioner": [{"id": "21223145", "key": "Commissioner", "value": "TONNY KUSNADI"}],
      "director": [{"id": "21223159", "key": "Director", "value": "DAVID FORMULA"}],
      "independent_commissioner": [{"id": "21223147", "key": "Commissioner (Independent)", "value": "SUMANTRI SLAMET"}]
    },
    "address": [{
      "office": "Menara BCA, Grand Indonesia",
      "phone": "021-23588000",
      "fax": "021-23588300",
      "email": ["investor_relations@bca.co.id"],
      "website": "www.bca.co.id",
      "npwp": "01.000.000"
    }],
    "beneficiary": [{"name": "ROBERT BUDI HARTONO"}],
    "shareholder": [
      {"percentage": "54.942%", "name": "PT DWIMURIA INVESTAMA ANDALAN", "value": "67.73 B", "badges": ["pengendali"]}
    ],
    "shareholder_one_percent": [
      {"id": "8911", "percentage": "54.94%", "name": "DWIMURIA INVESTAMA ANDALAN", "value": "67,729,950,000", "badges": []}
    ],
    "shareholder_director_commissioner": [
      {"percentage": "0.003%", "name": "TONNY KUSNADI", "value": "1.00 B", "badges": []}
    ],
    "shareholder_numbers": [
      {"shareholder_date": "31 Jul 2026", "total_share": "782,697", "change": -14418, "change_formatted": "(-14,418)"}
    ]
  }
}`

const sampleSubsidiariesPayload = `{
  "success": true,
  "data": {
    "currency": "CURRENCY_IDR",
    "last_updated_period": "Q2 2026",
    "unit": "UNIT_MILLION",
    "subsidiaries": [
      {
        "company_name": "PT Bank Digital BCA",
        "business_type": "Perbankan",
        "location": "Jakarta",
        "commercial_year": "1965",
        "total_assets": "20,818,005",
        "percentage": "100.00",
        "operational_status": "",
        "period": ""
      }
    ]
  }
}`

func TestClient_FetchCompanyProfile_OK(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/api/v1/company/BBCA/profile":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sampleProfilePayload))
		case "/api/v1/company/BBCA/subsidiaries":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sampleSubsidiariesPayload))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	p, err := c.FetchCompanyProfile(context.Background(), "BBCA")
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())

	// Header scalar mapping.
	assert.Equal(t, "BBCA", p.Symbol)
	require.NotNil(t, p.Background)
	assert.Equal(t, "PT Bank Central Asia Tbk.", *p.Background)
	require.NotNil(t, p.Board)
	assert.Equal(t, "Papan Utama", *p.Board)
	require.NotNil(t, p.ListingDate)
	assert.Equal(t, "31 May 2000", *p.ListingDate)
	require.NotNil(t, p.IpoAmount)
	assert.Equal(t, "927 B", *p.IpoAmount)
	require.NotNil(t, p.FreeFloat)
	assert.Equal(t, "42.46%", *p.FreeFloat)

	// Executives: three kinds, ordered commissioner → director → independent.
	require.Len(t, p.Executives, 3)
	assert.Equal(t, "commissioner", p.Executives[0].Kind)
	assert.Equal(t, "TONNY KUSNADI", p.Executives[0].Name)
	require.NotNil(t, p.Executives[0].ExternalID)
	assert.Equal(t, "21223145", *p.Executives[0].ExternalID)
	assert.Equal(t, "director", p.Executives[1].Kind)
	assert.Equal(t, "independent_commissioner", p.Executives[2].Kind)
	require.NotNil(t, p.Executives[2].Role)
	assert.Equal(t, "Commissioner (Independent)", *p.Executives[2].Role)

	// Holdings: four groups, percentage parsed + raw kept.
	require.Len(t, p.Holdings, 4)
	assert.Equal(t, "shareholder", p.Holdings[0].HolderGroup)
	require.NotNil(t, p.Holdings[0].Percentage)
	assert.InDelta(t, 54.942, *p.Holdings[0].Percentage, 0.0001)
	require.NotNil(t, p.Holdings[0].PercentageRaw)
	assert.Equal(t, "54.942%", *p.Holdings[0].PercentageRaw)
	assert.Equal(t, "one_percent", p.Holdings[1].HolderGroup)
	require.NotNil(t, p.Holdings[1].AmountRaw)
	assert.Equal(t, "67,729,950,000", *p.Holdings[1].AmountRaw)
	assert.Equal(t, "director_commissioner", p.Holdings[2].HolderGroup)
	assert.Equal(t, "beneficiary", p.Holdings[3].HolderGroup)
	assert.Equal(t, "ROBERT BUDI HARTONO", p.Holdings[3].Name)

	// Shareholder numbers.
	require.Len(t, p.ShareholderNumbers, 1)
	assert.Equal(t, "31 Jul 2026", p.ShareholderNumbers[0].ShareholderDate)
	require.NotNil(t, p.ShareholderNumbers[0].Change)
	assert.Equal(t, int64(-14418), *p.ShareholderNumbers[0].Change)

	// Address.
	require.Len(t, p.Addresses, 1)
	require.NotNil(t, p.Addresses[0].Emails)
	assert.Equal(t, []string{"investor_relations@bca.co.id"}, p.Addresses[0].Emails)

	// Subsidiaries from the dedicated endpoint.
	require.Len(t, p.Subsidiaries, 1)
	assert.Equal(t, "PT Bank Digital BCA", p.Subsidiaries[0].Name)
	require.NotNil(t, p.Subsidiaries[0].Percentage)
	assert.InDelta(t, 100.0, *p.Subsidiaries[0].Percentage, 0.001)
	require.NotNil(t, p.Subsidiaries[0].TotalAssets)
	assert.InDelta(t, 20818005.0, *p.Subsidiaries[0].TotalAssets, 0.001)
}

func TestClient_FetchCompanyProfile_ProfileUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.FetchCompanyProfile(context.Background(), "BBCA")
	assert.Error(t, err)
}

func TestClient_FetchCompanyProfile_SubsidiariesUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/company/BBCA/profile" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sampleProfilePayload))
			return
		}
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.FetchCompanyProfile(context.Background(), "BBCA")
	assert.Error(t, err)
}

func TestClient_FetchCompanyProfile_EmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.FetchCompanyProfile(context.Background(), "BBCA")
	assert.Error(t, err)
}

func TestParseDecimal(t *testing.T) {
	cases := []struct {
		in   string
		want *float64
	}{
		{"54.942%", f(54.942)},
		{"100.00", f(100)},
		{"20,818,005", f(20818005)},
		{"", nil},
		{"  ", nil},
		{"-", nil},
	}
	for _, c := range cases {
		got := parseDecimal(c.in)
		if c.want == nil {
			assert.Nil(t, got, c.in)
			continue
		}
		require.NotNil(t, got, c.in)
		assert.InDelta(t, *c.want, *got, 0.001, c.in)
	}
}

func f(v float64) *float64 { return &v }
