package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpstream(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		retryAfter     string
		alsoNoData     []int
		wantStatus     int
		wantCode       string
		wantRetryAfter string
	}{
		{name: "400 becomes 422 validation", status: 400, wantStatus: 422, wantCode: CodeValidation},
		{name: "404 in alsoNoData becomes 422", status: 404, alsoNoData: []int{404}, wantStatus: 422, wantCode: CodeValidation},
		{name: "404 alone becomes 500", status: 404, wantStatus: 500, wantCode: CodeInternalError},
		{name: "429 becomes 429 with retry-after", status: 429, retryAfter: "30", wantStatus: 429, wantCode: CodeTooManyRequests, wantRetryAfter: "30"},
		{name: "429 without hint has no header", status: 429, wantStatus: 429, wantCode: CodeTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			wrote := Upstream(rec, tt.status, tt.retryAfter, "no data", "fallback", tt.alsoNoData...)

			require.True(t, wrote)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			assert.Equal(t, tt.wantCode, env.Error.Code)
			assert.Equal(t, tt.wantRetryAfter, rec.Header().Get("Retry-After"))
		})
	}
}
