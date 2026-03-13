package querybuilder

import (
	"fmt"
	"strings"
)

// InsertBuilder constructs parameterized INSERT statements with automatic $N
// placeholder numbering.
type InsertBuilder struct {
	table   string
	columns []string
	args    []interface{}
	argN    int
}

// Insert creates a new InsertBuilder for the given table.
func Insert(table string) *InsertBuilder {
	return &InsertBuilder{table: table}
}

// Set adds a column and its value to the INSERT.
func (b *InsertBuilder) Set(col string, val interface{}) *InsertBuilder {
	b.columns = append(b.columns, col)
	b.argN++
	b.args = append(b.args, val)
	return b
}

// Build assembles the final INSERT SQL and argument slice.
func (b *InsertBuilder) Build() (string, []interface{}) {
	var sb strings.Builder

	sb.WriteString("INSERT INTO ")
	sb.WriteString(b.table)
	sb.WriteString(" (")
	sb.WriteString(strings.Join(b.columns, ", "))
	sb.WriteString(") VALUES (")

	for i := range b.columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "$%d", i+1)
	}

	sb.WriteString(")")

	return sb.String(), b.args
}
