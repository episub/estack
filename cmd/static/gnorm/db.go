package gnorm

// Note that this file is *NOT* generated. :)

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx"
)

var safeField = regexp.MustCompile(`^[a-zA-Z_0-9]+\z`)

// DB is the common interface for database operations.
// This should work with database/sql.DB and database/sql.Tx.
type DB interface {
	Exec(string, ...interface{}) (pgx.CommandTag, error)
	Query(string, ...interface{}) (*pgx.Rows, error)
	QueryRow(string, ...interface{}) *pgx.Row
}

// WhereClause has a String function should return a properly formatted where
// clause (not including the WHERE) for positional arguments starting at idx.
type WhereClause interface {
	String(idx *int) string
	Values() []interface{}
}

// Order Specifies an order for field
type Order struct {
	Fields     []string
	Descending bool
}

// NewOrder Convenience function to return new order
func NewOrder(descending bool) Order {
	return Order{Descending: descending}
}

// AddField Add a field to sort by
func (o *Order) AddField(field string) error {
	// Extra layer to help prevent SQL injection attack
	if !safeField.MatchString(field) {
		return fmt.Errorf("Invalid field for sorting")
	}

	o.Fields = append(o.Fields, field)

	return nil
}

// Length Returns how many fields are being sorted by
func (o *Order) Length() int {
	return len(o.Fields)
}

// String Returns an order string
func (o *Order) String() string {
	ord := "ASC"

	if o.Descending {
		ord = "DESC"
	}

	str := strings.Join(o.Fields, fmt.Sprintf(" %s, ", ord))
	if len(o.Fields) == 0 {
		str = "true"
	}
	return str + " " + ord
}

type comparison string

const (
	compEqual   comparison = " = "
	compGreater comparison = " > "
	compLess    comparison = " < "
	compGTE     comparison = " >= "
	compLTE     comparison = " <= "
	compNE      comparison = " <> "
)

type inClause struct {
	field  string
	values []interface{}
}

func (in inClause) String(idx *int) string {
	if len(in.values) == 0 {
		return "false"
	}
	ret := in.field + " in ( VALUES "

	// We may need to cast the type -- e.g., string to uuid
	var cast string
	switch reflect.TypeOf(in.values[0]).String() {
	case "int":
		cast = "::int"
	case "string":
		cast = "::uuid"
	}

	for x := range in.values {
		if x != 0 {
			ret += ", "
		}
		// TODO: Don't hard code 'uuid', but instead set it somewhere else, such as inClause, specifying the correct field type?
		ret += "($" + strconv.Itoa(*idx) + cast + ")"
		(*idx)++
	}
	ret += ")"
	return ret
}

func (in inClause) Values() []interface{} {
	return in.values
}

type whereClause struct {
	field string
	comp  comparison
	value interface{}
}

func (w whereClause) String(idx *int) string {
	ret := w.field + string(w.comp) + "$" + strconv.Itoa(*idx)
	(*idx)++
	return ret
}

func (w whereClause) Values() []interface{} {
	return []interface{}{w.value}
}

type nullClause struct {
	field  string
	isNull bool
}

func (n nullClause) String(idx *int) string {
	var comp string
	if n.isNull {
		comp = " IS NULL "
	} else {
		comp = " IS NOT NULL "
	}
	return n.field + comp
}

func (n nullClause) Values() []interface{} {
	return []interface{}{}
}

// CustomClause Allows us to specify a custom clause for the field when none of the existing suit
type CustomClause struct {
	strFunc func(*int) string
	value   interface{}
}

// NewCustomClause Returns a new Custom Clause
func NewCustomClause(strf func(*int) string, val interface{}) CustomClause {
	return CustomClause{
		strFunc: strf,
		value:   val,
	}
}

func (c CustomClause) String(idx *int) string {
	ret := c.strFunc(idx)
	(*idx)++
	return ret
}

func (c CustomClause) Values() []interface{} {
	return []interface{}{c.value}
}

// AndClause returns a WhereClause that serializes to the AND
// of all the given where clauses.
func AndClause(wheres ...WhereClause) WhereClause {
	return andClause(wheres)
}

type andClause []WhereClause

func (a andClause) String(idx *int) string {
	wheres := make([]string, len(a))
	for x := 0; x < len(a); x++ {
		wheres[x] = a[x].String(idx)
	}
	return strings.Join(wheres, " AND ")
}

func (a andClause) Values() []interface{} {
	vals := make([]interface{}, 0, len(a))
	for x := 0; x < len(a); x++ {
		vals = append(vals, a[x].Values()...)
	}
	return vals
}

// OrClause returns a WhereClause that serializes to the OR
// of all the given where clauses.
func OrClause(wheres ...WhereClause) WhereClause {
	return orClause(wheres)
}

type orClause []WhereClause

func (o orClause) String(idx *int) string {
	wheres := make([]string, len(o))
	for x := 0; x < len(wheres); x++ {
		wheres[x] = o[x].String(idx)
	}
	return strings.Join(wheres, " OR ")
}

func (o orClause) Values() []interface{} {
	vals := make([]interface{}, len(o))
	for x := 0; x < len(o); x++ {
		vals = append(vals, o[x].Values()...)
	}
	return vals
}
