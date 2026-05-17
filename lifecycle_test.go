package strapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/jorgejr568/strapi-go/blocks"
	"github.com/jorgejr568/strapi-go/query"
)

// richPage is a content-type struct used by TestBlocksRoundtrip to exercise
// the blocks subpackage end-to-end.
type richPage struct {
	Title string        `json:"title"`
	Slug  string        `json:"slug"`
	Body  blocks.Blocks `json:"body"`
}

// memoryStore is a minimal in-memory Strapi simulator used by lifecycle
// scenarios. It supports a single collection of pages addressed by
// documentId and a single homepage record (for SingleType scenarios).
type memoryStore struct {
	mu       sync.Mutex
	pages    map[string]map[string]any
	homepage map[string]any
	nextID   int
	nextDoc  int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		pages:    map[string]map[string]any{},
		homepage: map[string]any{"headline": "default", "subline": ""},
	}
}

// handler routes Strapi-shaped requests against the in-memory store.
func (m *memoryStore) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/pages" && r.Method == http.MethodGet:
			m.listPages(w, r)
		case r.URL.Path == "/api/pages" && r.Method == http.MethodPost:
			m.createPage(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/pages/") && r.Method == http.MethodGet:
			m.findPage(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/pages/") && r.Method == http.MethodPut:
			m.updatePage(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/pages/") && r.Method == http.MethodDelete:
			m.deletePage(w, r)
		case r.URL.Path == "/api/homepage" && r.Method == http.MethodGet:
			m.getHomepage(w, r)
		case r.URL.Path == "/api/homepage" && r.Method == http.MethodPut:
			m.updateHomepage(w, r)
		default:
			http.Error(w, "not found", 404)
		}
	})
}

func (m *memoryStore) listPages(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data := make([]map[string]any, 0, len(m.pages))
	for _, p := range m.pages {
		data = append(data, p)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": data,
		"meta": map[string]any{
			"pagination": map[string]any{
				"page": 1, "pageSize": 25, "pageCount": 1, "total": len(data),
			},
		},
	})
}

func (m *memoryStore) createPage(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(body, &env)
	m.nextID++
	m.nextDoc++
	docID := "doc" + strings.Repeat("0", 21-len("doc")) + intToStr(m.nextDoc)
	page := map[string]any{
		"id": m.nextID, "documentId": docID,
		"createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z",
	}
	for k, v := range env.Data {
		page[k] = v
	}
	m.pages[docID] = page
	_ = json.NewEncoder(w).Encode(map[string]any{"data": page})
}

func (m *memoryStore) findPage(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	docID := strings.TrimPrefix(r.URL.Path, "/api/pages/")
	p, ok := m.pages[docID]
	if !ok {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"data":null,"error":{"status":404,"name":"NotFoundError","message":"Not Found","details":{}}}`))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": p})
}

func (m *memoryStore) updatePage(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	docID := strings.TrimPrefix(r.URL.Path, "/api/pages/")
	p, ok := m.pages[docID]
	if !ok {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"data":null,"error":{"status":404,"name":"NotFoundError","message":"Not Found","details":{}}}`))
		return
	}
	body, _ := io.ReadAll(r.Body)
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(body, &env)
	for k, v := range env.Data {
		p[k] = v
	}
	p["updatedAt"] = "2026-01-02T00:00:00Z"
	m.pages[docID] = p
	_ = json.NewEncoder(w).Encode(map[string]any{"data": p})
}

func (m *memoryStore) deletePage(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	docID := strings.TrimPrefix(r.URL.Path, "/api/pages/")
	delete(m.pages, docID)
	w.WriteHeader(204)
}

func (m *memoryStore) getHomepage(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]any{
		"id": 1, "documentId": "homepage-doc-id-x000000001",
		"createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z",
	}
	for k, v := range m.homepage {
		out[k] = v
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": out})
}

func (m *memoryStore) updateHomepage(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(body, &env)
	for k, v := range env.Data {
		m.homepage[k] = v
	}
	out := map[string]any{
		"id": 1, "documentId": "homepage-doc-id-x000000001",
		"createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z",
	}
	for k, v := range m.homepage {
		out[k] = v
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": out})
}

func intToStr(n int) string {
	// helper to avoid importing strconv just for the harness
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// TestCollectionLifecycle exercises the full CRUD flow against an in-memory
// Strapi simulator: Create → Find → partial Update → List → Delete →
// Find (404). Each step's response is generated from real state, so this
// covers the full read+write codepath plus error-envelope decoding.
func TestCollectionLifecycle(t *testing.T) {
	store := newMemoryStore()
	srv := newTestServer(t, store.handler().ServeHTTP)

	c := New(WithBaseURL(srv.URL), WithToken("t0k"))
	pages := NewCollection[pageAttrs](c, "pages")
	ctx := context.Background()

	// 1. Create
	created, err := pages.Create(ctx, pageAttrs{Title: "Hello", Slug: "hello"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.DocumentID == "" {
		t.Fatal("Create: DocumentID empty")
	}
	if created.Attributes.Title != "Hello" {
		t.Errorf("Create: Title = %q", created.Attributes.Title)
	}

	// 2. Find — pulls the page we just created
	found, err := pages.Find(ctx, created.DocumentID)
	if err != nil {
		t.Fatalf("Find after Create: %v", err)
	}
	if found.Attributes.Title != "Hello" || found.Attributes.Slug != "hello" {
		t.Errorf("Find: got %+v", found.Attributes)
	}

	// 3. Update — partial map; server merges (does not overwrite slug)
	updated, err := pages.Update(ctx, created.DocumentID, map[string]any{"title": "Updated"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Attributes.Title != "Updated" {
		t.Errorf("Update: Title = %q", updated.Attributes.Title)
	}
	if updated.Attributes.Slug != "hello" {
		t.Errorf("Update: Slug should be unchanged, got %q", updated.Attributes.Slug)
	}

	// 4. List — returns the single page with pagination meta
	list, err := pages.List(ctx, query.Paginate(1, 25))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Data) != 1 {
		t.Fatalf("List Data len = %d want 1", len(list.Data))
	}
	if list.Data[0].Attributes.Title != "Updated" {
		t.Errorf("List: title = %q", list.Data[0].Attributes.Title)
	}
	if list.Meta.Pagination.Total != 1 {
		t.Errorf("List: Total = %d", list.Meta.Pagination.Total)
	}

	// 5. Delete
	if err := pages.Delete(ctx, created.DocumentID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// 6. Find again — should be 404 / ErrNotFound
	_, err = pages.Find(ctx, created.DocumentID)
	if err == nil {
		t.Fatal("Find after Delete: expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Find after Delete: err = %v, want errors.Is(ErrNotFound)", err)
	}
}

// TestSingleTypeLifecycle drives a SingleType through Get → Update → Get
// against the same in-memory simulator. Verifies the update is persisted
// and the second Get sees the new state.
func TestSingleTypeLifecycle(t *testing.T) {
	store := newMemoryStore()
	srv := newTestServer(t, store.handler().ServeHTTP)

	c := New(WithBaseURL(srv.URL))
	hp := NewSingleType[homepageAttrs](c, "homepage")
	ctx := context.Background()

	// 1. Get initial state
	initial, err := hp.Get(ctx)
	if err != nil {
		t.Fatalf("Get initial: %v", err)
	}
	if initial.Attributes.Headline != "default" {
		t.Errorf("initial.Headline = %q", initial.Attributes.Headline)
	}

	// 2. Update with a partial map
	updated, err := hp.Update(ctx, map[string]any{"headline": "Welcome back"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Attributes.Headline != "Welcome back" {
		t.Errorf("updated.Headline = %q", updated.Attributes.Headline)
	}

	// 3. Get again — the update should be persisted server-side
	second, err := hp.Get(ctx)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if second.Attributes.Headline != "Welcome back" {
		t.Errorf("second.Headline = %q want 'Welcome back'", second.Attributes.Headline)
	}
}

// TestComplexQueryRoundtrip builds a maximally-rich query and verifies
// every query parameter makes it onto the wire and the response decodes.
func TestComplexQueryRoundtrip(t *testing.T) {
	var gotQuery string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{
            "data": [
                {"id": 1, "documentId": "d1", "title": "A", "slug": "a",
                 "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}
            ],
            "meta": {"pagination": {"page": 2, "pageSize": 5, "pageCount": 3, "total": 12}}
        }`))
	})

	c := New(WithBaseURL(srv.URL))
	pages := NewCollection[pageAttrs](c, "pages")

	list, err := pages.List(context.Background(),
		query.Paginate(2, 5),
		query.Sort("title:asc", "createdAt:desc"),
		query.Locale("en"),
		query.Status(query.StatusPublished),
		query.Where(query.And(
			query.Eq("status", "active"),
			query.Or(
				query.Eq("featured", true),
				query.Gt("views", 1000),
			),
		)),
		query.With(
			query.Field("author").Fields("name", "email"),
			query.Field("categories").
				Sort("name:asc").
				Populate(query.Field("parent").Fields("name")),
		),
	)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Verify response decoded.
	if len(list.Data) != 1 || list.Meta.Pagination.Page != 2 || list.Meta.Pagination.Total != 12 {
		t.Fatalf("response decode unexpected: %+v", list)
	}

	// Verify every query layer made it onto the wire.
	// net/url.ParseQuery decodes percent-encoded bracket keys back to their
	// literal Go form, so we can assert on "populate[0]" etc. directly.
	parsed, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query: %v\nraw: %s", err, gotQuery)
	}
	mustHave := []struct {
		key   string
		value string
	}{
		{"pagination[page]", "2"},
		{"pagination[pageSize]", "5"},
		{"sort[0]", "title:asc"},
		{"sort[1]", "createdAt:desc"},
		{"locale", "en"},
		{"status", "published"},
		{"filters[$and][0][status][$eq]", "active"},
		{"filters[$and][1][$or][0][featured][$eq]", "true"},
		{"filters[$and][1][$or][1][views][$gt]", "1000"},
		{"populate[author][fields][0]", "name"},
		{"populate[author][fields][1]", "email"},
		{"populate[categories][sort][0]", "name:asc"},
		{"populate[categories][populate][parent][fields][0]", "name"},
	}
	for _, mh := range mustHave {
		if got := parsed.Get(mh.key); got != mh.value {
			t.Errorf("missing query param %q=%q (got %q)", mh.key, mh.value, got)
		}
	}
}

// TestBlocksRoundtrip exercises the entire blocks subpackage: decode a
// document containing every node type and every text modifier, then render
// it via RenderHTML and assert the output contains every expected element.
// This is the end-to-end test for content rendering.
func TestBlocksRoundtrip(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
            "data": {
                "id": 1, "documentId": "doc1",
                "title": "Rich Page",
                "slug": "rich",
                "body": [
                    {"type":"heading","level":1,"children":[{"type":"text","text":"Title"}]},
                    {"type":"paragraph","children":[
                        {"type":"text","text":"plain "},
                        {"type":"text","text":"bold","bold":true},
                        {"type":"text","text":" italic","italic":true},
                        {"type":"text","text":" both","bold":true,"italic":true}
                    ]},
                    {"type":"list","format":"unordered","children":[
                        {"type":"list-item","children":[{"type":"text","text":"first"}]},
                        {"type":"list-item","children":[{"type":"text","text":"second"}]}
                    ]},
                    {"type":"list","format":"ordered","children":[
                        {"type":"list-item","children":[{"type":"text","text":"one"}]}
                    ]},
                    {"type":"paragraph","children":[
                        {"type":"text","text":"with "},
                        {"type":"link","url":"https://example.com/docs","children":[{"type":"text","text":"inline link"}]},
                        {"type":"text","text":" here"}
                    ]},
                    {"type":"list","format":"unordered","children":[
                        {"type":"list-item","children":[
                            {"type":"text","text":"outer "},
                            {"type":"list","format":"unordered","children":[
                                {"type":"list-item","children":[{"type":"text","text":"nested"}]}
                            ]}
                        ]}
                    ]},
                    {"type":"quote","children":[{"type":"text","text":"quoted"}]},
                    {"type":"code","children":[{"type":"text","text":"go run ."}]},
                    {"type":"link","url":"https://example.com","children":[{"type":"text","text":"site"}]},
                    {"type":"image","image":{"url":"/uploads/x.jpg","alternativeText":"alt","width":800,"height":600},"children":[{"type":"text","text":""}]},
                    {"type":"unknown-future","payload":{"x":1}}
                ],
                "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"
            },
            "meta": {}
        }`))
	})

	pages := NewCollection[richPage](New(WithBaseURL(srv.URL)), "pages")
	page, err := pages.Find(context.Background(), "doc1")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(page.Attributes.Body) != 11 {
		t.Fatalf("body len = %d want 11", len(page.Attributes.Body))
	}

	html := blocks.RenderHTML(page.Attributes.Body)

	mustContain := []string{
		"<h1>Title</h1>",
		"<p>plain ",
		"<strong>bold</strong>",
		"<em> italic</em>",
		"<strong><em> both</em></strong>",
		"<ul><li>first</li><li>second</li></ul>",
		"<ol><li>one</li></ol>",
		// New: inline link inside paragraph
		`<a href="https://example.com/docs">inline link</a>`,
		// New: nested list inside list-item
		`<ul><li>outer <ul><li>nested</li></ul></li></ul>`,
		"<blockquote>quoted</blockquote>",
		"<pre><code>go run .</code></pre>",
		`<a href="https://example.com">site</a>`,
		`<img src="/uploads/x.jpg"`,
		`alt="alt"`,
		`width="800"`,
		`height="600"`,
	}
	for _, s := range mustContain {
		if !strings.Contains(html, s) {
			t.Errorf("html missing %q\nhtml=%s", s, html)
		}
	}
	// The unknown-future block must NOT crash the renderer and must not
	// appear in output (Unknown is silently skipped per the design).
	// Negative assertions: the unknown-future block must NOT appear in output.
	for _, forbidden := range []string{"unknown-future", `"x":1`} {
		if strings.Contains(html, forbidden) {
			t.Errorf("html should not contain %q\nhtml=%s", forbidden, html)
		}
	}
}
