package config

import "testing"

func TestTrustedProxiesFromEnv(t *testing.T) {
	t.Setenv("KABAO_TRUSTED_PROXIES", "127.0.0.1, 10.0.0.0/8, ,::1")

	got := trustedProxiesFromEnv()
	if len(got) != 3 {
		t.Fatalf("want 3 proxies, got %v", got)
	}
	if got[0] != "127.0.0.1" || got[1] != "10.0.0.0/8" || got[2] != "::1" {
		t.Fatalf("unexpected proxies: %v", got)
	}
}

func TestTLSHandshakeLogFilterDropsUnknownCertificateNoise(t *testing.T) {
	var buf stubBuffer
	w := tlsHandshakeLogFilter{next: &buf}

	n, err := w.Write([]byte("http: TLS handshake error from 10.0.0.20: remote error: tls: unknown certificate\n"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n == 0 {
		t.Fatalf("want reported bytes written")
	}
	if buf.s != "" {
		t.Fatalf("want filtered line to be dropped, got %q", buf.s)
	}

	_, err = w.Write([]byte("http: TLS handshake error from 10.0.0.20: EOF\n"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if buf.s == "" {
		t.Fatalf("want non-matching TLS error to be preserved")
	}
}

type stubBuffer struct {
	s string
}

func (b *stubBuffer) Write(p []byte) (int, error) {
	b.s += string(p)
	return len(p), nil
}
