package log

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cmdpkg "github.com/chaitin/chaitin-cli/products/safeline/cmd"
	"github.com/chaitin/chaitin-cli/products/safeline/pkg/client"
)

func TestDetectGetDoesNotRequireTimestamp(t *testing.T) {
	testDetectGetOmitsEmptyTimestamp(t, []string{"--event-id", "event-1"})
}

func TestDetectGetOmitsExplicitEmptyTimestamp(t *testing.T) {
	testDetectGetOmitsEmptyTimestamp(t, []string{"--event-id", "event-1", "--timestamp", ""})
}

func testDetectGetOmitsEmptyTimestamp(t *testing.T, args []string) {
	t.Helper()

	var sawTimestamp bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/FilterV2API" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if got := q.Get("scope"); got != "log:detect_log:detail" {
			t.Fatalf("scope = %q, want log:detect_log:detail", got)
		}
		if got := q.Get("event_id__exact"); got != "event-1" {
			t.Fatalf("event_id__exact = %q, want event-1", got)
		}
		_, sawTimestamp = q["timestamp__exact"]

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.Envelope{
			Data: json.RawMessage(`[{"event_id":"event-1","timestamp":"123","src_ip":"1.1.1.1"}]`),
		})
	}))
	defer srv.Close()

	cmdpkg.SetFlags(srv.URL, "token", "json", false, false)

	c := newDetectGetCmd()
	out := &strings.Builder{}
	c.SetOut(out)
	c.SetErr(&strings.Builder{})
	c.SetArgs(args)

	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if sawTimestamp {
		t.Fatal("timestamp__exact was sent without --timestamp")
	}
}
