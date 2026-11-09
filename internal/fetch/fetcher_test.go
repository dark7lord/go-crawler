package fetch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewRequestInterval(t *testing.T) {
	tests := []struct {
		name         string
		delay        time.Duration
		wantInterval time.Duration
	}{
		{
			name:         "explicit delay is used as-is",
			delay:        50 * time.Millisecond,
			wantInterval: 50 * time.Millisecond,
		},
		{
			name:         "explicit delay below default is honored",
			delay:        20 * time.Millisecond,
			wantInterval: 20 * time.Millisecond,
		},
		{
			name:         "zero delay falls back to default",
			wantInterval: DefaultDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ftr, _ := newTestFetcher(t, tt.delay)

			require.Equal(t, tt.wantInterval, ftr.requestInterval)
		})
	}
}

func TestNewRetryInterval(t *testing.T) {
	tests := []struct {
		name         string
		delay        time.Duration
		wantInterval time.Duration
	}{
		{
			name:         "short explicit delay keeps minimum retry interval",
			delay:        50 * time.Millisecond,
			wantInterval: minRetryInterval,
		},
		{
			name:         "zero delay uses minimum retry interval",
			wantInterval: minRetryInterval,
		},
		{
			name:         "long explicit delay raises retry interval",
			delay:        500 * time.Millisecond,
			wantInterval: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ftr, _ := newTestFetcher(t, tt.delay)

			require.Equal(t, tt.wantInterval, ftr.retryInterval)
		})
	}
}
