package models

import "testing"

func TestIsModeConsistentWithStatus(t *testing.T) {
	cases := []struct {
		name string
		s    *ServiceSession
		want bool
	}{
		{
			name: "nil_session",
			s:    nil,
			want: true,
		},
		{
			name: "empty_mode_always_true",
			s:    &ServiceSession{SessionMode: "", Status: "cs_start_pending"},
			want: true,
		},
		{
			name: "no_prefix_status_true",
			s:    &ServiceSession{SessionMode: SessionModeCustomerService, Status: "start_pending"},
			want: true,
		},
		{
			name: "cs_prefix_match",
			s:    &ServiceSession{SessionMode: SessionModeCustomerService, Status: "cs_start_pending"},
			want: true,
		},
		{
			name: "cs_prefix_mismatch",
			s:    &ServiceSession{SessionMode: SessionModeCustomerService, Status: "qs_start_pending"},
			want: false,
		},
		{
			name: "qs_prefix_match",
			s:    &ServiceSession{SessionMode: SessionModeQueueAutoSingle, Status: "qs_staff_selecting"},
			want: true,
		},
		{
			name: "qm_prefix_match",
			s:    &ServiceSession{SessionMode: SessionModeQueueAutoMulti, Status: "qm_start_pending"},
			want: true,
		},
		{
			name: "qms_prefix_match",
			s:    &ServiceSession{SessionMode: SessionModeQueueManualSingle, Status: "qms_delay_pending"},
			want: true,
		},
		{
			name: "qmm_prefix_match",
			s:    &ServiceSession{SessionMode: SessionModeQueueManualMulti, Status: "qmm_start_pending"},
			want: true,
		},
		{
			name: "qmm_prefix_mismatch",
			s:    &ServiceSession{SessionMode: SessionModeQueueManualMulti, Status: "qms_start_pending"},
			want: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := IsModeConsistentWithStatus(tc.s); got != tc.want {
				t.Fatalf("want=%v got=%v session=%+v", tc.want, got, tc.s)
			}
		})
	}
}
