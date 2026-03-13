package querybuilder

import (
	"fmt"
	"strings"
)

// Builder constructs parameterized SELECT queries with automatic $N placeholder
// numbering. Methods are fluent — each returns *Builder for chaining.
type Builder struct {
	columns []string
	table   string
	joins   []string
	where   []string
	args    []interface{}
	argN    int
	orderBy string
	limit   int
	offset  int
}

// Select creates a new Builder with the given column list.
func Select(cols ...string) *Builder {
	return &Builder{columns: cols}
}

// From sets the FROM clause (e.g. "products p").
func (b *Builder) From(table string) *Builder {
	b.table = table
	return b
}

// LeftJoin appends a LEFT JOIN clause.
func (b *Builder) LeftJoin(table, on string) *Builder {
	b.joins = append(b.joins, fmt.Sprintf("LEFT JOIN %s ON %s", table, on))
	return b
}

// WithFilter adds "col = $N" to WHERE if cond is true.
func (b *Builder) WithFilter(cond bool, col string, val interface{}) *Builder {
	if !cond {
		return b
	}
	b.where = append(b.where, fmt.Sprintf("%s = %s", col, b.nextArg(val)))
	return b
}

// WithLike adds "col ILIKE $N" to WHERE if cond is true.
func (b *Builder) WithLike(cond bool, col string, val string) *Builder {
	if !cond {
		return b
	}
	b.where = append(b.where, fmt.Sprintf("%s ILIKE %s", col, b.nextArg(val)))
	return b
}

// WithSearch adds a full-text search condition combined with an ILIKE fallback.
// When cond is true it emits:
//
//	(vectorCol @@ websearch_to_tsquery('english', $N) OR likeCol ILIKE $N+1)
//
// The ILIKE uses a wrapping wildcard (%term%) so partial matches are still found.
func (b *Builder) WithSearch(cond bool, vectorCol, likeCol, term string) *Builder {
	if !cond {
		return b
	}
	tsArg := b.nextArg(term)
	likeArg := b.nextArg("%" + term + "%")
	b.where = append(b.where, fmt.Sprintf(
		"(%s @@ websearch_to_tsquery('english', %s) OR %s ILIKE %s)",
		vectorCol, tsArg, likeCol, likeArg,
	))
	return b
}

// WithRange adds ">= $N" and/or "<= $N" conditions for non-nil bounds.
func (b *Builder) WithRange(col string, lower, upper *int64) *Builder {
	if lower != nil {
		b.where = append(b.where, fmt.Sprintf("%s >= %s", col, b.nextArg(*lower)))
	}
	if upper != nil {
		b.where = append(b.where, fmt.Sprintf("%s <= %s", col, b.nextArg(*upper)))
	}
	return b
}

// WithRaw adds a raw WHERE fragment if cond is true. Each %s in the clause is
// replaced with the next $N placeholder. The number of %s markers must match
// the number of args.
func (b *Builder) WithRaw(cond bool, clause string, args ...interface{}) *Builder {
	if !cond {
		return b
	}
	placeholders := make([]interface{}, len(args))
	for i, arg := range args {
		placeholders[i] = b.nextArg(arg)
	}
	b.where = append(b.where, fmt.Sprintf(clause, placeholders...))
	return b
}

// OrderBy sets the ORDER BY clause (e.g. "p.sort_order, p.id").
func (b *Builder) OrderBy(clause string) *Builder {
	b.orderBy = clause
	return b
}

// OrderByRaw sets the ORDER BY clause with parameterized arguments.
// Each %s in the clause is replaced with the next $N placeholder.
func (b *Builder) OrderByRaw(clause string, args ...interface{}) *Builder {
	placeholders := make([]interface{}, len(args))
	for i, arg := range args {
		placeholders[i] = b.nextArg(arg)
	}
	b.orderBy = fmt.Sprintf(clause, placeholders...)
	return b
}

// Limit sets the LIMIT value.
func (b *Builder) Limit(n int) *Builder {
	b.limit = n
	return b
}

// Offset sets the OFFSET value.
func (b *Builder) Offset(n int) *Builder {
	b.offset = n
	return b
}

// Where adds an unconditional "col = $N" to WHERE.
func (b *Builder) Where(col string, val interface{}) *Builder {
	b.where = append(b.where, fmt.Sprintf("%s = %s", col, b.nextArg(val)))
	return b
}

// WhereIn adds "col = ANY($N)" to WHERE.
func (b *Builder) WhereIn(col string, vals interface{}) *Builder {
	b.where = append(b.where, fmt.Sprintf("%s = ANY(%s)", col, b.nextArg(vals)))
	return b
}

// Build assembles the final SQL string and argument slice.
func (b *Builder) Build() (string, []interface{}) {
	var sb strings.Builder

	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(b.columns, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(b.table)

	for _, j := range b.joins {
		sb.WriteString(" ")
		sb.WriteString(j)
	}

	if len(b.where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(b.where, " AND "))
	}

	if b.orderBy != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(b.orderBy)
	}

	if b.limit > 0 {
		fmt.Fprintf(&sb, " LIMIT %s", b.nextArg(b.limit))
	}

	if b.offset > 0 {
		fmt.Fprintf(&sb, " OFFSET %s", b.nextArg(b.offset))
	}

	return sb.String(), b.args
}

// nextArg appends val to the args slice and returns the next "$N" placeholder.
func (b *Builder) nextArg(val interface{}) string {
	b.argN++
	b.args = append(b.args, val)
	return fmt.Sprintf("$%d", b.argN)
}
