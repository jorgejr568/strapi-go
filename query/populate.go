package query

import "strconv"

// PopulateAll adds populate=* (shallow — one level).
func PopulateAll() Option {
	return func(q *Query) {
		q.add([]string{"populate"}, "*")
	}
}

// Populate adds populate[0]=name&populate[1]=other for the named relations.
func Populate(fields ...string) Option {
	return func(q *Query) {
		for i, f := range fields {
			q.add([]string{"populate", strconv.Itoa(i)}, f)
		}
	}
}

// With adds one or more deep-populate clauses built via Field.
func With(builders ...*PopulateBuilder) Option {
	return func(q *Query) {
		for _, b := range builders {
			b.encode(q, []string{"populate"})
		}
	}
}

// PopulateBuilder describes a deep-populate clause for a single relation
// field. Construct one with Field, then chain Fields, Sort, Where, Populate
// to refine it.
//
// Example:
//
//	Field("articles").
//	    Sort("publishedAt:desc").
//	    Populate(Field("author").Fields("name"))
type PopulateBuilder struct {
	name   string
	fields []string
	sort   []string
	filter Filter
	nested []*PopulateBuilder
}

// Field constructs a populate builder for the named relation.
func Field(name string) *PopulateBuilder {
	return &PopulateBuilder{name: name}
}

// Fields selects which sub-fields of the populated relation to return.
func (p *PopulateBuilder) Fields(names ...string) *PopulateBuilder {
	p.fields = append(p.fields, names...)
	return p
}

// Sort sets the sort order on the populated relation.
func (p *PopulateBuilder) Sort(specs ...string) *PopulateBuilder {
	p.sort = append(p.sort, specs...)
	return p
}

// Where applies a filter to the populated relation.
func (p *PopulateBuilder) Where(f Filter) *PopulateBuilder {
	p.filter = f
	return p
}

// Populate nests further populate builders inside this one.
func (p *PopulateBuilder) Populate(children ...*PopulateBuilder) *PopulateBuilder {
	p.nested = append(p.nested, children...)
	return p
}

func (p *PopulateBuilder) encode(q *Query, prefix []string) {
	base := append(append([]string{}, prefix...), p.name)
	for i, f := range p.fields {
		q.add(append(append([]string{}, base...), "fields", strconv.Itoa(i)), f)
	}
	for i, s := range p.sort {
		q.add(append(append([]string{}, base...), "sort", strconv.Itoa(i)), s)
	}
	if p.filter != nil {
		p.filter.encode(q, append(append([]string{}, base...), "filters"))
	}
	for _, child := range p.nested {
		child.encode(q, append(append([]string{}, base...), "populate"))
	}
}
