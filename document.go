package strapi

import (
	"encoding/json"
	"time"
)

// Document is the generic envelope for a single Strapi v5 entry. System
// fields (ID, DocumentID, timestamps, locale) are split out from user-defined
// content fields, which live in Attributes. T should be a struct whose json
// tags match the content type's field names.
type Document[T any] struct {
	ID          int        `json:"-"`
	DocumentID  string     `json:"-"`
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
	PublishedAt *time.Time `json:"-"`
	Locale      string     `json:"-"`
	Attributes  T          `json:"-"`
}

// UnmarshalJSON splits Strapi v5's flat entry shape into system fields and
// user-defined attributes by running two decodes over the same payload. The
// system-fields decode is keyed by Strapi's well-known names; the attribute
// decode targets T directly, which sees only its own tagged fields.
func (d *Document[T]) UnmarshalJSON(data []byte) error {
	var sys struct {
		ID          int        `json:"id"`
		DocumentID  string     `json:"documentId"`
		CreatedAt   time.Time  `json:"createdAt"`
		UpdatedAt   time.Time  `json:"updatedAt"`
		PublishedAt *time.Time `json:"publishedAt"`
		Locale      string     `json:"locale"`
	}
	if err := json.Unmarshal(data, &sys); err != nil {
		return err
	}
	d.ID = sys.ID
	d.DocumentID = sys.DocumentID
	d.CreatedAt = sys.CreatedAt
	d.UpdatedAt = sys.UpdatedAt
	d.PublishedAt = sys.PublishedAt
	d.Locale = sys.Locale
	return json.Unmarshal(data, &d.Attributes)
}

// ListResponse is the envelope for collection-list responses. It is the
// return type of Collection[T].List and the top-level List[T] helper.
//
// (Named ListResponse rather than List because Go places types and
// functions in the same package-scope namespace and the public API also
// exposes a generic helper named List.)
type ListResponse[T any] struct {
	Data []Document[T] `json:"data"`
	Meta Meta          `json:"meta"`
}

// single is the unexported envelope for find-one / mutation responses.
// Consumers always get back *Document[T] from the public API.
type single[T any] struct {
	Data Document[T] `json:"data"`
	Meta Meta        `json:"meta"`
}

// Meta carries pagination and other response metadata.
type Meta struct {
	Pagination Pagination `json:"pagination"`
}

// Pagination is the page-based pagination metadata. Strapi also supports
// start/limit (offset) pagination; in that mode PageCount is zero and Total
// is still populated.
type Pagination struct {
	Page      int `json:"page"`
	PageSize  int `json:"pageSize"`
	PageCount int `json:"pageCount"`
	Total     int `json:"total"`
}
