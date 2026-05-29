package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chaitin/chaitin-cli/products/safeline/pkg/client"
)

func TestResolveContextRemoteWins(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ServerControlledConfigAPI" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"err":  nil,
			"data": map[string]any{"version": "25.03.007_r7", "operation_mode": []string{"Software Reverse Proxy"}},
		})
	}))
	defer srv.Close()

	ctx, err := ResolveContext(client.New(srv.URL, srv.Client()), Options{ConfigVersion: "23.01.014", ConfigOperationMode: "Software Cluster Reverse Proxy"})
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}
	if ctx.Version != "25.03.007_r7" || ctx.VersionSource != "remote" {
		t.Fatalf("bad version %+v", ctx)
	}
	if ctx.OperationMode != ModeSoftwareReverseProxy || ctx.OperationModeSource != "remote" {
		t.Fatalf("bad mode %+v", ctx)
	}
	if len(ctx.Warnings) == 0 {
		t.Fatalf("expected config mismatch warning")
	}
}

func TestResolveContextOverrideWinsWhenRemoteMissing(t *testing.T) {
	ctx, err := ResolveContext(nil, Options{VersionOverride: "23.01.014", OperationModeOverride: "software-cluster-reverse-proxy"})
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}
	if ctx.Version != "23.01.014" || ctx.VersionSource != "override" {
		t.Fatalf("bad version %+v", ctx)
	}
	if ctx.OperationMode != ModeSoftwareClusterReverseProxy || ctx.OperationModeSource != "override" {
		t.Fatalf("bad mode %+v", ctx)
	}
	if ctx.Endpoint != "/api/SoftwareClusterReverseProxyWebsiteAPI" || !ctx.SiteCreateSupported {
		t.Fatalf("bad endpoint/support %+v", ctx)
	}
}

func TestResolveContextConfigFallback(t *testing.T) {
	ctx, err := ResolveContext(nil, Options{ConfigVersion: "25.03.009", ConfigOperationMode: "Software Reverse Proxy"})
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}
	if ctx.VersionFamily != Family25_03 || ctx.OperationMode != ModeSoftwareReverseProxy {
		t.Fatalf("bad context %+v", ctx)
	}
}
