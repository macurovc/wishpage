package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePort(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "default", value: "", want: defaultPort},
		{name: "configured", value: "8080", want: 8080},
		{name: "not a number", value: "http", wantErr: true},
		{name: "zero", value: "0", wantErr: true},
		{name: "above range", value: "65536", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := parsePort(tt.value)
			if tt.wantErr {
				require.ErrorIs(t, err, errInvalidPort)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, port)
		})
	}
}
