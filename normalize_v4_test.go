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
