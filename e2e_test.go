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
		testName           string
		action             string
		contentLength      int
		expectedStatusCode int
	}{
		{
			testName:           "EqualToLimit_ProcessPartial",
			action:             "ProcessPartial",
			contentLength:      responseBodyLimit,
			expectedStatusCode: http.StatusOK,
		},
		{
			testName:           "OneByteLongerThanLimit_ProcessPartial",
			action:             "ProcessPartial",
			contentLength:      responseBodyLimit + 1,
			expectedStatusCode: http.StatusOK,
		},
		{
			testName:      "EqualToLimit_Reject",
			action:        "Reject",
			contentLength: responseBodyLimit,
			// Is 413 appropriate when the response body is too long?
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			testName:      "OneByteLongerThanLimit_Reject",
			action:        "Reject",
			contentLength: responseBodyLimit + 1,
			// Is 413 appropriate when the response body is too long?
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			testName:           "OneByteShorterThanLimit_Reject",
			action:             "Reject",
			contentLength:      responseBodyLimit - 1,
			expectedStatusCode: http.StatusOK,
		},
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
				log {
					level WARN
				}
			}
			(waf) {
				coraza_waf {
					directives `+"`"+`
						SecRuleEngine On
						SecResponseBodyAccess On
						SecResponseBodyMimeType text/plain
						SecResponseBodyLimit %d
						SecResponseBodyLimitAction %s
					`+"`"+`
				}
			}
			:%d {
				import waf
				reverse_proxy %s
			}`, caddyAdminPort, responseBodyLimit, tc.action, caddyPort, originServerAddr)
			if err := os.WriteFile("/tmp/Caddyfile", []byte(config), 0o644); err != nil {
				t.Fatal(err)
			}
			tester.InitServer(config, "caddyfile")

			if tc.expectedStatusCode == http.StatusOK {
				tester.AssertGetResponse(fmt.Sprintf("http://127.0.0.1:%d", caddyPort), http.StatusOK, content)
			} else {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d", caddyPort), nil)
				if err != nil {
					t.Fatal(err)
				}
				tester.AssertResponseCode(req, tc.expectedStatusCode)
			}
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
