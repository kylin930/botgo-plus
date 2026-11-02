package search

import (
	"github.com/kylin930/botgo-plus/interaction/signature"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSimulateSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		pass, err := signature.Verify("local-test-secret", r.Header, body)
		if r.Method != "POST" || err != nil || !pass {
			t.Error("unsigned local search callback")
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{}`)
	}))
	defer server.Close()
	if _, err := SimulateSearch(&Config{AppID: "test", EndPoint: server.URL, Secret: "local-test-secret"}, "hello"); err != nil {
		t.Fatal(err)
	}
}
