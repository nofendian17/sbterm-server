package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Level
		wantErr bool
	}{
		{name: "debug lowercase", in: "debug", want: LevelDebug},
		{name: "info uppercase", in: "INFO", want: LevelInfo},
		{name: "warn mixed case", in: "Warn", want: LevelWarn},
		{name: "error lowercase", in: "error", want: LevelError},
		{name: "invalid level", in: "verbose", wantErr: true},
		{name: "empty string", in: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Format
		wantErr bool
	}{
		{name: "text", in: "text", want: FormatText},
		{name: "json", in: "JSON", want: FormatJSON},
		{name: "invalid", in: "yaml", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
