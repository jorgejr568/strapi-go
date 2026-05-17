package strapi

import "encoding/json"

// normalizeV4ToV5 rewrites a Strapi v4 REST response into the v5 flat-entry
// shape the SDK's typed envelopes (Document[T], ListResponse[T], single[T])
// expect. It is a pure transformation — no I/O, no client state.
//
// Rules applied recursively to the decoded JSON tree:
//
//   - Entry envelope: an object with {id, attributes: <object>, ...} is
//     replaced by a flat object that lifts every key from attributes up to
//     the parent level (the "id" and any sibling keys are preserved).
//   - Relation envelope: an object whose only key is "data" is replaced by
//     the value of "data" (which may itself be an entry, an array of
//     entries, or null — recursion handles each).
//
// The top-level {data, meta} response envelope is preserved: only the
// inner "data" value is walked. This is what keeps the existing
// Document[T] decode path working unchanged.
func normalizeV4ToV5(in []byte) ([]byte, error) {
	var root any
	if err := json.Unmarshal(in, &root); err != nil {
		return nil, err
	}
	if obj, ok := root.(map[string]any); ok {
		if data, hasData := obj["data"]; hasData {
			obj["data"] = walkV4(data)
			return json.Marshal(obj)
		}
	}
	return json.Marshal(walkV4(root))
}

// walkV4 recursively normalizes a single decoded JSON value. See
// normalizeV4ToV5 for the rule set.
func walkV4(v any) any {
	switch x := v.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(x))
		for k, val := range x {
			normalized[k] = walkV4(val)
		}
		if attrs, ok := normalized["attributes"].(map[string]any); ok {
			if _, hasID := normalized["id"]; hasID {
				merged := make(map[string]any, len(normalized)+len(attrs)-1)
				for k, v := range attrs {
					merged[k] = v
				}
				for k, v := range normalized {
					if k == "attributes" {
						continue
					}
					merged[k] = v
				}
				return merged
			}
		}
		return normalized
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = walkV4(item)
		}
		return out
	default:
		return v
	}
}
