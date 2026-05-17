package strapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

// assertJSONEqual unmarshals both sides into any and compares — key ordering
// in marshaled JSON is not stable, so we compare the decoded shape instead.
func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("unmarshal got: %v\nraw: %s", err, got)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatalf("unmarshal want: %v\nraw: %s", err, want)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("JSON mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestNormalizeV4SingleEntry(t *testing.T) {
	in := []byte(`{
		"data": {
			"id": 1,
			"attributes": {
				"title": "Home",
				"slug":  "home"
			}
		},
		"meta": {}
	}`)
	want := []byte(`{
		"data": {
			"id": 1,
			"title": "Home",
			"slug":  "home"
		},
		"meta": {}
	}`)
	got, err := normalizeV4ToV5(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	assertJSONEqual(t, got, want)
}

func TestNormalizeV4SingleRelationEnvelope(t *testing.T) {
	in := []byte(`{
		"data": {
			"id": 1,
			"attributes": {
				"title": "Home",
				"author": { "data": { "id": 5, "attributes": { "name": "Alice" } } }
			}
		}
	}`)
	want := []byte(`{
		"data": {
			"id": 1,
			"title": "Home",
			"author": { "id": 5, "name": "Alice" }
		}
	}`)
	got, err := normalizeV4ToV5(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	assertJSONEqual(t, got, want)
}

func TestNormalizeV4ArrayRelationEnvelope(t *testing.T) {
	in := []byte(`{
		"data": {
			"id": 1,
			"attributes": {
				"tags": { "data": [
					{ "id": 7, "attributes": { "name": "go" } },
					{ "id": 8, "attributes": { "name": "rest" } }
				] }
			}
		}
	}`)
	want := []byte(`{
		"data": {
			"id": 1,
			"tags": [
				{ "id": 7, "name": "go" },
				{ "id": 8, "name": "rest" }
			]
		}
	}`)
	got, err := normalizeV4ToV5(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	assertJSONEqual(t, got, want)
}

func TestNormalizeV4NullRelationEnvelope(t *testing.T) {
	in := []byte(`{
		"data": {
			"id": 1,
			"attributes": {
				"author": { "data": null }
			}
		}
	}`)
	want := []byte(`{
		"data": {
			"id": 1,
			"author": null
		}
	}`)
	got, err := normalizeV4ToV5(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	assertJSONEqual(t, got, want)
}

func TestNormalizeV4ListResponse(t *testing.T) {
	in := []byte(`{
		"data": [
			{ "id": 1, "attributes": { "title": "A" } },
			{ "id": 2, "attributes": { "title": "B" } }
		],
		"meta": { "pagination": { "page": 1, "pageSize": 25, "pageCount": 1, "total": 2 } }
	}`)
	want := []byte(`{
		"data": [
			{ "id": 1, "title": "A" },
			{ "id": 2, "title": "B" }
		],
		"meta": { "pagination": { "page": 1, "pageSize": 25, "pageCount": 1, "total": 2 } }
	}`)
	got, err := normalizeV4ToV5(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	assertJSONEqual(t, got, want)
}

func TestNormalizeV4DeepNestedPopulate(t *testing.T) {
	in := []byte(`{
		"data": {
			"id": 1,
			"attributes": {
				"title": "Post",
				"author": { "data": {
					"id": 5,
					"attributes": {
						"name": "Alice",
						"avatar": { "data": {
							"id": 9,
							"attributes": { "url": "/uploads/alice.png", "name": "alice.png" }
						} }
					}
				} }
			}
		}
	}`)
	want := []byte(`{
		"data": {
			"id": 1,
			"title": "Post",
			"author": {
				"id": 5,
				"name": "Alice",
				"avatar": { "id": 9, "url": "/uploads/alice.png", "name": "alice.png" }
			}
		}
	}`)
	got, err := normalizeV4ToV5(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	assertJSONEqual(t, got, want)
}
