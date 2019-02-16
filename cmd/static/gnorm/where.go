package gnorm

import "fmt"

// IsInArray Creates a where clause for when provided value is in field array
type IsInArray struct {
	Field string
	Value string
}

// NewIsInArray Returns a new IsInArray where clause
func NewIsInArray(field string, value string) IsInArray {
	return IsInArray{Field: field, Value: value}
}

func (i IsInArray) String(idx *int) string {
	if idx != nil {
		str := fmt.Sprintf("$%d = ANY(%s)", *idx, i.Field)
		(*idx)++
		return str
	}
	return ""
}

func (i IsInArray) Values() []interface{} {
	return []interface{}{i.Value}
}
