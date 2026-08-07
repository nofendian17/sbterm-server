package response

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponses(t *testing.T) {
	tests := []struct {
		name        string
		write       func(w http.ResponseWriter)
		wantStatus  int
		wantSuccess bool
		wantMessage string
		wantErrCode string
		wantDetails map[string]string
		wantData    bool
		wantMeta    *MetaBody
		wantNoBody  bool
	}{
		{
			name: "ok writes 200 with data",
			write: func(w http.ResponseWriter) {
				OK(w, map[string]string{"id": "1"})
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
			wantData:    true,
		},
		{
			name: "created writes 201",
			write: func(w http.ResponseWriter) {
				Created(w, "resource")
			},
			wantStatus:  http.StatusCreated,
			wantSuccess: true,
			wantData:    true,
		},
		{
			name: "success writes custom status",
			write: func(w http.ResponseWriter) {
				Success(w, http.StatusAccepted, "accepted")
			},
			wantStatus:  http.StatusAccepted,
			wantSuccess: true,
			wantData:    true,
		},
		{
			name: "message writes without data",
			write: func(w http.ResponseWriter) {
				Message(w, http.StatusOK, "all good")
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
			wantMessage: "all good",
			wantData:    false,
		},
		{
			name: "no content writes empty body",
			write: func(w http.ResponseWriter) {
				NoContent(w)
			},
			wantStatus: http.StatusNoContent,
			wantNoBody: true,
		},
		{
			name: "error writes error envelope",
			write: func(w http.ResponseWriter) {
				Error(w, http.StatusNotFound, CodeNotFound, "user not found")
			},
			wantStatus:  http.StatusNotFound,
			wantSuccess: false,
			wantMessage: "user not found",
			wantErrCode: CodeNotFound,
		},
		{
			name: "validation error writes details",
			write: func(w http.ResponseWriter) {
				ValidationError(w, "invalid input", map[string]string{"email": "must be a valid email address"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantSuccess: false,
			wantMessage: "invalid input",
			wantErrCode: CodeValidation,
			wantDetails: map[string]string{"email": "must be a valid email address"},
		},
		{
			name: "paginated writes data and meta",
			write: func(w http.ResponseWriter) {
				Paginated(w, []string{"item1", "item2"}, &MetaBody{
					Page:       1,
					Limit:      10,
					TotalItems: 100,
					TotalPages: 10,
				})
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
			wantData:    true,
			wantMeta: &MetaBody{
				Page:       1,
				Limit:      10,
				TotalItems: 100,
				TotalPages: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.write(rec)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantNoBody {
				assert.Empty(t, rec.Body.String())
				return
			}

			var env struct {
				Success bool            `json:"success"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
				Meta    *MetaBody       `json:"meta"`
				Error   *struct {
					Code    string            `json:"code"`
					Message string            `json:"message"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			assert.Equal(t, tt.wantSuccess, env.Success)
			if tt.wantMessage != "" {
				assert.Equal(t, tt.wantMessage, env.Message)
			}
			if tt.wantData {
				assert.NotEmpty(t, env.Data)
			} else {
				assert.Empty(t, env.Data)
			}
			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
			} else {
				assert.Nil(t, env.Error)
			}
			if tt.wantDetails != nil {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantDetails, env.Error.Details)
			}
			if tt.wantMeta != nil {
				require.NotNil(t, env.Meta)
				assert.Equal(t, tt.wantMeta, env.Meta)
			}
		})
	}
}

func TestWriteJSONContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, map[string]string{"a": "b"})

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Contains(t, bytes.NewBuffer(rec.Body.Bytes()).String(), `"a":"b"`)
}

func TestWriteJSONEncodeError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, map[string]any{"bad": make(chan struct{})})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), http.StatusText(http.StatusInternalServerError))
}
