package querybuilder

import (
	"fmt"
	"strings"
)

// UpdateBuilder constructs parameterized UPDATE statements with automatic $N
// placeholder numbering.
type UpdateBuilder struct {
	table string
	sets  []string
	where []string
	args  []interface{}
	argN  int
}

// Update creates a new UpdateBuilder for the given table.
func Update(table string) *UpdateBuilder {
	return &UpdateBuilder{table: table}
}

// Set adds "col = $N" to the SET clause.
func (b *UpdateBuilder) Set(col string, val interface{}) *UpdateBuilder {
	b.sets = append(b.sets, fmt.Sprintf("%s = %s", col, b.nextArg(val)))
	return b
}

// SetRaw adds a raw SET expression. Each %s in the expr is replaced with $N.
func (b *UpdateBuilder) SetRaw(col string, expr string, args ...interface{}) *UpdateBuilder {
	placeholders := make([]interface{}, len(args))
	for i, arg := range args {
		placeholders[i] = b.nextArg(arg)
	}
	if len(args) > 0 {
		b.sets = append(b.sets, fmt.Sprintf("%s = %s", col, fmt.Sprintf(expr, placeholders...)))
	} else {
		b.sets = append(b.sets, fmt.Sprintf("%s = %s", col, expr))
	}
	return b
}

// Where adds "col = $N" to the WHERE clause.
func (b *UpdateBuilder) Where(col string, val interface{}) *UpdateBuilder {
	b.where = append(b.where, fmt.Sprintf("%s = %s", col, b.nextArg(val)))
	return b
}

// Build assembles the final UPDATE SQL and argument slice.
func (b *UpdateBuilder) Build() (string, []interface{}) {
	var sb strings.Builder

	sb.WriteString("UPDATE ")
	sb.WriteString(b.table)
	sb.WriteString(" SET ")
	sb.WriteString(strings.Join(b.sets, ", "))

	if len(b.where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(b.where, " AND "))
	}

	return sb.String(), b.args
}

// nextArg appends val to the args slice and returns the next "$N" placeholder.
func (b *UpdateBuilder) nextArg(val interface{}) string {
	b.argN++
	b.args = append(b.args, val)
	return fmt.Sprintf("$%d", b.argN)
}
