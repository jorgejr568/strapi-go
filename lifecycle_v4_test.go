package strapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jorgejr568/strapi-go/media"
)

// authorAttrs is the content-type used by TestV4DecodesRelationsAndMedia.
type authorAttrs struct {
	Name   string      `json:"name"`
	Avatar *media.File `json:"avatar,omitempty"`
}

// tagAttrs is the content-type used by TestV4DecodesRelationsAndMedia.
type tagAttrs struct {
	Name string `json:"name"`
}

// postWithRelations is the content-type used by TestV4DecodesRelationsAndMedia.
type postWithRelations struct {
	Title  string                 `json:"title"`
	Slug   string                 `json:"slug"`
	Author *Document[authorAttrs] `json:"author,omitempty"`
	Tags   []Document[tagAttrs]   `json:"tags,omitempty"`
}

func serveV4Fixture(t *testing.T, name string) *Client {
	t.Helper()
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		_, _ = w.Write(b)
	})
	return New(WithBaseURL(srv.URL), WithAPIVersion(APIVersionV4))
}

func TestV4FindDecodesSingleEntry(t *testing.T) {
	c := serveV4Fixture(t, "page_single_v4.json")
	pages := NewCollection[pageAttrs](c, "pages")

	doc, err := pages.Find(context.Background(), "1")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if doc.ID != 1 {
		t.Errorf("ID = %d want 1", doc.ID)
	}
	if doc.DocumentID != "" {
		t.Errorf("DocumentID should be empty in v4, got %q", doc.DocumentID)
	}
	if doc.Attributes.Title != "Home" || doc.Attributes.Slug != "home" {
		t.Errorf("attrs = %+v", doc.Attributes)
	}
	if doc.Locale != "en" {
		t.Errorf("Locale = %q", doc.Locale)
	}
}

func TestV4ListDecodesEntries(t *testing.T) {
	c := serveV4Fixture(t, "page_list_v4.json")
	pages := NewCollection[pageAttrs](c, "pages")

	list, err := pages.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Data) != 2 {
		t.Fatalf("Data len = %d want 2", len(list.Data))
	}
	if list.Data[0].Attributes.Title != "Home" {
		t.Errorf("Data[0].Title = %q", list.Data[0].Attributes.Title)
	}
	if list.Data[1].PublishedAt != nil {
		t.Errorf("Data[1].PublishedAt should be nil (draft)")
	}
	if list.Meta.Pagination.Total != 2 {
		t.Errorf("Total = %d want 2", list.Meta.Pagination.Total)
	}
}

func TestV4DecodesRelationsAndMedia(t *testing.T) {
	c := serveV4Fixture(t, "page_relations_v4.json")
	pages := NewCollection[postWithRelations](c, "pages")

	doc, err := pages.Find(context.Background(), "1")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	// Single relation envelope unwrapped → Author populated.
	if doc.Attributes.Author == nil {
		t.Fatal("Author is nil — single-relation envelope was not unwrapped")
	}
	if doc.Attributes.Author.ID != 5 {
		t.Errorf("Author.ID = %d want 5", doc.Attributes.Author.ID)
	}
	if doc.Attributes.Author.Attributes.Name != "Alice" {
		t.Errorf("Author.Name = %q want Alice", doc.Attributes.Author.Attributes.Name)
	}
	// Media nested inside relation → still resolves.
	avatar := doc.Attributes.Author.Attributes.Avatar
	if avatar == nil {
		t.Fatal("Avatar is nil — nested relation/media envelope not unwrapped")
	}
	if avatar.URL != "/uploads/alice.png" || avatar.MIME != "image/png" {
		t.Errorf("Avatar = %+v", avatar)
	}
	// Array relation envelope unwrapped → Tags has both items.
	if len(doc.Attributes.Tags) != 2 {
		t.Fatalf("Tags len = %d want 2", len(doc.Attributes.Tags))
	}
	if doc.Attributes.Tags[0].Attributes.Name != "go" || doc.Attributes.Tags[1].Attributes.Name != "rest" {
		t.Errorf("Tags = %+v", doc.Attributes.Tags)
	}
}

// memoryStoreV4 is a v4-shaped sibling of memoryStore. It emits v4 entry
// envelopes ({id, attributes: {...}}) so the normalizer is exercised
// end-to-end through every CRUD operation.
type memoryStoreV4 struct {
	mu     sync.Mutex
	pages  map[int]map[string]any // numeric id → flat attribute map
	nextID int
}

func newMemoryStoreV4() *memoryStoreV4 {
	return &memoryStoreV4{pages: map[int]map[string]any{}}
}

func (m *memoryStoreV4) wrapEntry(id int, attrs map[string]any) map[string]any {
	return map[string]any{"id": id, "attributes": attrs}
}

func (m *memoryStoreV4) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/pages" && r.Method == http.MethodGet:
			m.list(w)
		case r.URL.Path == "/api/pages" && r.Method == http.MethodPost:
			m.create(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/pages/") && r.Method == http.MethodGet:
			m.find(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/pages/") && r.Method == http.MethodPut:
			m.update(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/pages/") && r.Method == http.MethodDelete:
			m.del(w, r)
		default:
			http.Error(w, "not found", 404)
		}
	})
}

func (m *memoryStoreV4) list(w http.ResponseWriter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data := make([]map[string]any, 0, len(m.pages))
	for id, attrs := range m.pages {
		data = append(data, m.wrapEntry(id, attrs))
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": data,
		"meta": map[string]any{"pagination": map[string]any{
			"page": 1, "pageSize": 25, "pageCount": 1, "total": len(data),
		}},
	})
}

func (m *memoryStoreV4) create(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(body, &env)
	m.nextID++
	m.pages[m.nextID] = env.Data
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": m.wrapEntry(m.nextID, env.Data),
		"meta": map[string]any{},
	})
}

func (m *memoryStoreV4) parseID(path string) (int, bool) {
	idStr := strings.TrimPrefix(path, "/api/pages/")
	for _, c := range idStr {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n := 0
	for _, c := range idStr {
		n = n*10 + int(c-'0')
	}
	return n, n != 0
}

func (m *memoryStoreV4) find(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.parseID(r.URL.Path)
	if !ok {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"data":null,"error":{"status":404,"name":"NotFoundError","message":"Not Found"}}`))
		return
	}
	attrs, ok := m.pages[id]
	if !ok {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"data":null,"error":{"status":404,"name":"NotFoundError","message":"Not Found"}}`))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": m.wrapEntry(id, attrs),
		"meta": map[string]any{},
	})
}

func (m *memoryStoreV4) update(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.parseID(r.URL.Path)
	if !ok {
		w.WriteHeader(404)
		return
	}
	attrs, ok := m.pages[id]
	if !ok {
		w.WriteHeader(404)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(body, &env)
	for k, v := range env.Data {
		attrs[k] = v
	}
	m.pages[id] = attrs
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": m.wrapEntry(id, attrs),
		"meta": map[string]any{},
	})
}

func (m *memoryStoreV4) del(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, _ := m.parseID(r.URL.Path)
	delete(m.pages, id)
	w.WriteHeader(204)
}

// TestV4CollectionLifecycle drives Create → Find → Update → List → Delete →
// Find(404) end-to-end through a v4-shaped in-memory store. It mirrors
// TestCollectionLifecycle (v5) so a side-by-side comparison shows the
// public API is identical regardless of API version — the only difference
// is the use of doc.PathID() to chain operations (since v4 has no
// DocumentID).
func TestV4CollectionLifecycle(t *testing.T) {
	store := newMemoryStoreV4()
	srv := newTestServer(t, store.handler().ServeHTTP)

	c := New(WithBaseURL(srv.URL), WithAPIVersion(APIVersionV4))
	pages := NewCollection[pageAttrs](c, "pages")
	ctx := context.Background()

	// 1. Create
	created, err := pages.Create(ctx, pageAttrs{Title: "Hello", Slug: "hello"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("Create: numeric ID not set")
	}
	if created.DocumentID != "" {
		t.Errorf("Create: DocumentID should be empty in v4, got %q", created.DocumentID)
	}
	if created.Attributes.Title != "Hello" {
		t.Errorf("Create: Title = %q", created.Attributes.Title)
	}

	// 2. Find via PathID — version-agnostic chaining.
	found, err := pages.Find(ctx, created.PathID())
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found.Attributes.Title != "Hello" || found.Attributes.Slug != "hello" {
		t.Errorf("Find: got %+v", found.Attributes)
	}

	// 3. Update
	updated, err := pages.Update(ctx, created.PathID(), map[string]any{"title": "Updated"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Attributes.Title != "Updated" {
		t.Errorf("Update: Title = %q", updated.Attributes.Title)
	}
	if updated.Attributes.Slug != "hello" {
		t.Errorf("Update: Slug should be preserved, got %q", updated.Attributes.Slug)
	}

	// 4. List
	list, err := pages.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Data) != 1 {
		t.Fatalf("List Data len = %d want 1", len(list.Data))
	}
	if list.Meta.Pagination.Total != 1 {
		t.Errorf("List Total = %d", list.Meta.Pagination.Total)
	}

	// 5. Delete
	if err := pages.Delete(ctx, created.PathID()); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// 6. Find again — 404
	_, err = pages.Find(ctx, created.PathID())
	if err == nil {
		t.Fatal("Find after Delete: expected ErrNotFound")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Find after Delete: err = %v, want errors.Is(ErrNotFound)", err)
	}
}
