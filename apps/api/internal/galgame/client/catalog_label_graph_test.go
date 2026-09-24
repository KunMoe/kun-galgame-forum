package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

func graphCatalog(t *testing.T) (*GalgameClient, func() (string, string)) {
	t.Helper()
	var (
		mu    sync.Mutex
		path  string
		query string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		path, query = r.URL.Path, r.URL.RawQuery
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/catalog/companies/24/graph":
			_, _ = w.Write([]byte(`{"nodes":[` +
				`{"id":"24","display_name":"ねこねこソフト","localized":{"zh-Hans":{"value":"猫猫社","kind":"translation"}},"logo":{"url":"https://cdn.example/aabbccdd.webp","hash":"aabbccdd","width":200,"height":60},"work_count":33},` +
				`{"id":"993","display_name":"VisualArt's","localized":{},"logo":null,"work_count":120},` +
				`{"id":"994","name":"Na-Ga","logo":{"url":"https://cdn.example/11223344.webp","hash":"11223344","width":200,"height":60},"work_count":0}],` +
				`"edges":[{"from":24,"to":993,"relation":"parent"},` +
				`{"from":994,"to":993,"relation":"parent"}]}`))
		case "/v2/catalog/companies/309/graph":
			_, _ = w.Write([]byte(`{"nodes":[` +
				`{"id":"309","display_name":"无关系社","logo":null,"work_count":2}],"edges":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":4,"message":"资源不存在"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "test-key", "https://cdn.test/image"), func() (string, string) {
		mu.Lock()
		defer mu.Unlock()
		return path, query
	}
}

func TestCatalogLabelRelationGraphReadsTheContractShape(t *testing.T) {
	c, asked := graphCatalog(t)

	graph, found, appErr := c.CatalogLabelRelationGraph(context.Background(), "24")
	if appErr != nil {
		t.Fatalf("unexpected error: %v", appErr)
	}
	if !found {
		t.Fatal("found = false for a live label")
	}
	if len(graph.Nodes) != 3 || len(graph.Edges) != 2 {
		t.Fatalf("graph = %d nodes / %d edges, want 3/2", len(graph.Nodes), len(graph.Edges))
	}
	if graph.Nodes[0].Localized["zh-Hans"].Value != "猫猫社" || graph.Nodes[0].WorkCount != 33 {
		t.Errorf("seed node = %+v, want 猫猫社/33", graph.Nodes[0])
	}
	if graph.Nodes[1].DisplayName != "VisualArt's" {
		t.Errorf("untranslated node = %+v, want its display_name", graph.Nodes[1])
	}
	if graph.Nodes[2].Name != "Na-Ga" {
		t.Errorf("bare-name node = %+v, want Na-Ga", graph.Nodes[2])
	}
	if graph.Nodes[1].LogoHash != "" {
		t.Errorf("logoless node kept %q", graph.Nodes[1].LogoHash)
	}
	if e := graph.Edges[0]; e.From != 24 || e.To != 993 || e.Relation != "parent" {
		t.Errorf("edge[0] = %+v, want 24→993 parent", e)
	}

	path, query := asked()
	if path != "/v2/catalog/companies/24/graph" {
		t.Errorf("path = %q", path)
	}
	// The graph face takes the company include vocabulary: without it the nodes
	// come back as id+name and every logo on the relation map disappears.
	if want := "include=" + url.QueryEscape(v2CatalogDetailInclude["companies"]) + "&nsfw=true"; query != want {
		t.Errorf("query = %q, want %q", query, want)
	}
}

func TestCatalogLabelRelationGraphKeepsTheLoneMaker(t *testing.T) {
	c, _ := graphCatalog(t)

	graph, found, appErr := c.CatalogLabelRelationGraph(context.Background(), "309")
	if appErr != nil || !found {
		t.Fatalf("found=%v err=%v, want a one-node graph", found, appErr)
	}
	if len(graph.Nodes) != 1 || len(graph.Edges) != 0 {
		t.Fatalf("graph = %d nodes / %d edges, want 1/0", len(graph.Nodes), len(graph.Edges))
	}
}

func TestCatalogLabelRelationGraphMissIsNotAnError(t *testing.T) {
	c, _ := graphCatalog(t)

	graph, found, appErr := c.CatalogLabelRelationGraph(context.Background(), "999999")
	if appErr != nil {
		t.Fatalf("a 404 became an error: %v", appErr)
	}
	if found || graph != nil {
		t.Fatalf("found=%v graph=%v, want the miss reported as found=false", found, graph)
	}
}
