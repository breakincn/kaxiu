package handlers

import "testing"

func TestTechnicianStatusConflictText(t *testing.T) {
	cases := []struct {
		status string
		want   string
	}{
		{status: "busy", want: "在服务中"},
		{status: "paused", want: "在暂停服务中"},
		{status: "rest", want: "已下班"},
	}

	for _, tc := range cases {
		if got := technicianStatusConflictText(tc.status); got != tc.want {
			t.Fatalf("status=%s want=%s got=%s", tc.status, tc.want, got)
		}
	}
}

func TestShouldBlockCustomerServiceStartByAttendance(t *testing.T) {
	cases := []struct {
		name                   string
		status                 string
		hasOtherServingSession bool
		want                   bool
	}{
		{name: "idle_allowed", status: "idle", want: false},
		{name: "busy_current_pending_allowed", status: "busy", hasOtherServingSession: false, want: false},
		{name: "busy_other_serving_blocked", status: "busy", hasOtherServingSession: true, want: true},
		{name: "paused_blocked", status: "paused", want: true},
		{name: "rest_blocked", status: "rest", want: true},
	}

	for _, tc := range cases {
		if got := shouldBlockCustomerServiceStartByAttendance(tc.status, tc.hasOtherServingSession); got != tc.want {
			t.Fatalf("%s: want=%v got=%v", tc.name, tc.want, got)
		}
	}
}
