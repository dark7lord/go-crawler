package app

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveDelayFlags(t *testing.T) {
	tests := []struct {
		name    string
		delay   time.Duration
		rps     uint
		want    time.Duration
		wantErr string
	}{
		{
			name: "no flags falls back to default in fetcher",
			want: 0,
		},
		{
			name:  "only delay flag",
			delay: 200 * time.Millisecond,
			want:  200 * time.Millisecond,
		},
		{
			name: "only rps flag",
			rps:  5,
			want: 200 * time.Millisecond,
		},
		{
			name: "rps above default rate",
			rps:  20,
			want: 50 * time.Millisecond,
		},
		{
			name:  "delay and rps",
			delay: 10 * time.Millisecond,
			rps:   4,
			want:  250 * time.Millisecond,
		},
		{
			name: "1 second",
			rps:  1,
			want: time.Second,
		},
		{
			name:    "negative delay",
			delay:   -time.Millisecond,
			wantErr: "invalid delay",
		},
		{
			name:    "negative delay is rejected even when rps wins",
			delay:   -time.Millisecond,
			rps:     5,
			wantErr: "invalid delay",
		},
		{
			name:    "rps above 1 billion",
			rps:     2_000_000_000,
			wantErr: "invalid rps",
		},
		{
			name:    "rps overflowing duration",
			rps:     math.MaxUint,
			wantErr: "invalid rps",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDelay(tt.rps, tt.delay)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
