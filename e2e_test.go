package coraza

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/caddyserver/caddy/v2/caddytest"
)

func TestLongResponseBody(t *testing.T) {
	const responseBodyLimit = 64

	testCases := []struct {
		testName      string
		contentLength int
	}{
		{testName: "EqualToLimit", contentLength: responseBodyLimit},
		{testName: "OneByteLongerThanLimit", contentLength: responseBodyLimit + 1},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			content := strings.Repeat("a", tc.contentLength)

			originHandler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", strconv.Itoa(len(content)))
				w.Header().Set("Content-Type", "text/plain")
				if n, err := fmt.Fprint(w, content); err != nil {
					t.Logf("failed to write response: %s", err)
				} else if got, want := n, len(content); got != want {
					t.Errorf("written response byte count mismatch, got=%d, want=%d", got, want)
				}
			}
			originServer := httptest.NewServer(http.HandlerFunc(originHandler))
			t.Cleanup(originServer.Close)
			originServerAddr := originServer.Listener.Addr().String()

			caddyPort := findFreePort(t)
			caddyAdminPort := caddytest.Default.AdminPort
			tester := caddytest.NewTester(t)
			config := fmt.Sprintf(`{
				admin localhost:%d
				auto_https off
				order coraza_waf first
			}
			(waf) {
				coraza_waf {
					directives `+"`"+`
						SecRuleEngine On
						SecResponseBodyAccess On
						SecResponseBodyMimeType text/plain
						SecResponseBodyLimit %d
						SecResponseBodyLimitAction ProcessPartial
					`+"`"+`
				}
			}
			:%d {
				import waf
				reverse_proxy %s
			}`, caddyAdminPort, responseBodyLimit, caddyPort, originServerAddr)
			if err := os.WriteFile("/tmp/Caddyfile", []byte(config), 0o644); err != nil {
				t.Fatal(err)
			}
			tester.InitServer(config, "caddyfile")

			tester.AssertGetResponse(fmt.Sprintf("http://127.0.0.1:%d", caddyPort), 200, content)
		})
	}
}

func findFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	return port
}
