package middleware

import "testing"

func TestBuildAllowOriginFunc(t *testing.T) {
	allow := buildAllowOriginFunc(nil)

	cases := []struct {
		origin string
		want   bool
	}{
		{origin: "https://172.20.10.5:3000", want: true},
		{origin: "https://172.20.10.5:3001", want: true},
		{origin: "http://192.168.1.23:5173", want: true},
		{origin: "https://10.0.0.20:3002", want: true},
		{origin: "https://kabao.shop", want: true},
		{origin: "https://8.8.8.8:3000", want: false},
		{origin: "https://172.20.10.5:8080", want: false},
		{origin: "chrome-extension://abc", want: false},
	}

	for _, tc := range cases {
		got := allow(tc.origin)
		if got != tc.want {
			t.Fatalf("origin %q: want %v, got %v", tc.origin, tc.want, got)
		}
	}
}
