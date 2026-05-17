package query

import (
	"fmt"
	"strconv"
)

// Filter is anything that can encode itself into a Query at a given prefix.
// All path components are appended literally — callers must include array
// indices for $and/$or children themselves (Or/And handle this).
type Filter interface {
	encode(q *Query, prefix []string)
}

// Where adds a top-level filter under filters[...].
func Where(f Filter) Option {
	return func(q *Query) {
		f.encode(q, []string{"filters"})
	}
}

// --- comparison operators ------------------------------------------------

type cmp struct {
	path []string
	op   string
	val  any
}

func (c cmp) encode(q *Query, prefix []string) {
	full := append(append([]string{}, prefix...), c.path...)
	full = append(full, c.op)
	q.add(full, fmt.Sprint(c.val))
}

// Eq matches when field equals val.
func Eq(field string, val any) Filter        { return cmp{[]string{field}, "$eq", val} }
func EqI(field string, val any) Filter       { return cmp{[]string{field}, "$eqi", val} }
func Ne(field string, val any) Filter        { return cmp{[]string{field}, "$ne", val} }
func Lt(field string, val any) Filter        { return cmp{[]string{field}, "$lt", val} }
func Lte(field string, val any) Filter       { return cmp{[]string{field}, "$lte", val} }
func Gt(field string, val any) Filter        { return cmp{[]string{field}, "$gt", val} }
func Gte(field string, val any) Filter       { return cmp{[]string{field}, "$gte", val} }
func Contains(field string, val any) Filter  { return cmp{[]string{field}, "$contains", val} }
func ContainsI(field string, val any) Filter { return cmp{[]string{field}, "$containsi", val} }
func NotContains(field string, val any) Filter {
	return cmp{[]string{field}, "$notContains", val}
}
func StartsWith(field string, val any) Filter { return cmp{[]string{field}, "$startsWith", val} }
func EndsWith(field string, val any) Filter   { return cmp{[]string{field}, "$endsWith", val} }

// EqPath is Eq across a dotted/nested field path (filters on a relation).
func EqPath(path []string, val any) Filter {
	return cmp{path, "$eq", val}
}

// --- null / range / set --------------------------------------------------

// Null matches entries where the field is null.
func Null(field string) Filter {
	return cmp{[]string{field}, "$null", true}
}

// NotNull matches entries where the field is not null.
func NotNull(field string) Filter {
	return cmp{[]string{field}, "$notNull", true}
}

// Between matches values in the inclusive range [lo, hi].
func Between(field string, lo, hi any) Filter {
	return arrayFilter{field: field, op: "$between", vals: []any{lo, hi}}
}

// In matches values in the given set.
func In(field string, vals ...any) Filter {
	return arrayFilter{field: field, op: "$in", vals: vals}
}

// NotIn matches values NOT in the given set.
func NotIn(field string, vals ...any) Filter {
	return arrayFilter{field: field, op: "$notIn", vals: vals}
}

type arrayFilter struct {
	field string
	op    string
	vals  []any
}

func (a arrayFilter) encode(q *Query, prefix []string) {
	for i, v := range a.vals {
		full := append(append([]string{}, prefix...), a.field, a.op, strconv.Itoa(i))
		q.add(full, fmt.Sprint(v))
	}
}

// --- logical combinators -------------------------------------------------

type logical struct {
	op       string // "$and", "$or", "$not"
	children []Filter
}

func (l logical) encode(q *Query, prefix []string) {
	if l.op == "$not" {
		full := append(append([]string{}, prefix...), "$not")
		if len(l.children) == 1 {
			l.children[0].encode(q, full)
		}
		return
	}
	for i, ch := range l.children {
		full := append(append([]string{}, prefix...), l.op, strconv.Itoa(i))
		ch.encode(q, full)
	}
}

// And combines filters with logical AND.
func And(filters ...Filter) Filter {
	return logical{op: "$and", children: filters}
}

// Or combines filters with logical OR.
func Or(filters ...Filter) Filter {
	return logical{op: "$or", children: filters}
}

// Not negates a filter.
func Not(f Filter) Filter {
	return logical{op: "$not", children: []Filter{f}}
}
