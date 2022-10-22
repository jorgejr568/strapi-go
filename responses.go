package strapi_go

type StrapiResponseData[T any, M any] struct {
	ID         uint64 `json:"id"`
	Attributes T      `json:"attributes"`
	Meta       M      `json:"meta"`
}
type strapiResponse struct {
}
