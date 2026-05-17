package strapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/jorgejr568/strapi-go/query"
)

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
