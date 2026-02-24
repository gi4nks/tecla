package gitinfo

import (
	"testing"
)

func TestCalculateHealthScore(t *testing.T) {
	tests := []struct {
		name          string
		info          RepoInfo
		expectedScore int
	}{
		{
			name: "Perfect health",
			info: RepoInfo{
				Status: StatusInfo{Clean: true},
			},
			expectedScore: 100,
		},
		{
			name: "Dirty status",
			info: RepoInfo{
				Status: StatusInfo{Clean: false},
			},
			expectedScore: 90,
		},
		{
			name: "Ahead and Behind",
			info: RepoInfo{
				Status: StatusInfo{Clean: true},
				Ahead:  2,
				Behind: 1,
			},
			expectedScore: 85, // 100 - 10 - 5
		},
		{
			name: "CI Failure",
			info: RepoInfo{
				Status:       StatusInfo{Clean: true},
				RemoteStatus: RemoteStatus{CIStatus: "failure"},
			},
			expectedScore: 70, // 100 - 30
		},
		{
			name: "Error",
			info: RepoInfo{
				Status: StatusInfo{Clean: true},
				Error:  "fatal error",
			},
			expectedScore: 50, // 100 - 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.info.CalculateHealthScore()
			if tt.info.HealthScore != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, tt.info.HealthScore)
			}
		})
	}
}
