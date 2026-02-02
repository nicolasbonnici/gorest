package filter

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/query"
)

type FilterOperator string

const (
	OpEqual              FilterOperator = "eq"
	OpNotEqual           FilterOperator = "ne"
	OpGreaterThan        FilterOperator = "gt"
	OpGreaterThanOrEqual FilterOperator = "gte"
	OpLessThan           FilterOperator = "lt"
	OpLessThanOrEqual    FilterOperator = "lte"
	OpLike               FilterOperator = "like"
	OpILike              FilterOperator = "ilike"
	OpIn                 FilterOperator = "in"
	OpNotIn              FilterOperator = "nin"
)

type Filter struct {
	Field    string
	Operator FilterOperator
	Values   []string
}

type FilterSet struct {
	Filters       []Filter
	AllowedFields map[string]bool
	paramIndex    int
	dialect       database.Dialect
}

func NewFilterSet(allowedFields []string, dialect database.Dialect) *FilterSet {
	allowed := make(map[string]bool)
	for _, field := range allowedFields {
		allowed[field] = true
	}
	return &FilterSet{
		Filters:       []Filter{},
		AllowedFields: allowed,
		paramIndex:    1,
		dialect:       dialect,
	}
}

func (fs *FilterSet) ParseFromQuery(query url.Values) error {
	for key, values := range query {
		if key == "page" || key == "limit" || key == "count" || strings.HasPrefix(key, "order[") {
			continue
		}

		field, operator := fs.parseFieldAndOperator(key)

		if !fs.AllowedFields[field] {
			continue
		}

		if len(values) == 0 {
			continue
		}

		filter := Filter{
			Field:    field,
			Operator: operator,
			Values:   values,
		}

		// Handle optional array both field[]=val1&field[]=val2 AND field=val1&field=val2 syntax
		if operator == OpEqual && len(values) > 1 {
			filter.Operator = OpIn
		}

		if operator == OpIn && len(values) == 1 {
			filter.Operator = OpEqual
		}

		fs.Filters = append(fs.Filters, filter)
	}

	return nil
}

func (fs *FilterSet) parseFieldAndOperator(key string) (string, FilterOperator) {
	if strings.HasSuffix(key, "[]") {
		trimmedKey := strings.TrimSuffix(key, "[]")
		// Check for [nin][] pattern: field[nin][]
		if strings.HasSuffix(trimmedKey, "[nin]") {
			return strings.TrimSuffix(trimmedKey, "[nin]"), OpNotIn
		}
		return trimmedKey, OpIn
	}

	if strings.Contains(key, "[") && strings.Contains(key, "]") {
		start := strings.Index(key, "[")
		end := strings.Index(key, "]")
		field := key[:start]
		op := key[start+1 : end]

		switch op {
		case "gte":
			return field, OpGreaterThanOrEqual
		case "lte":
			return field, OpLessThanOrEqual
		case "gt":
			return field, OpGreaterThan
		case "lt":
			return field, OpLessThan
		case "ne":
			return field, OpNotEqual
		case "like":
			return field, OpLike
		case "ilike":
			return field, OpILike
		case "nin":
			return field, OpNotIn
		}
	}

	return key, OpEqual
}

func (fs *FilterSet) BuildWhereClause() (string, []interface{}) {
	if len(fs.Filters) == 0 {
		return "", nil
	}

	var conditions []string
	var args []interface{}

	for _, filter := range fs.Filters {
		switch filter.Operator {
		case OpEqual:
			conditions = append(conditions, fmt.Sprintf("%s = %s", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
			args = append(args, filter.Values[0])
			fs.paramIndex++

		case OpNotEqual:
			conditions = append(conditions, fmt.Sprintf("%s != %s", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
			args = append(args, filter.Values[0])
			fs.paramIndex++

		case OpGreaterThan:
			conditions = append(conditions, fmt.Sprintf("%s > %s", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
			args = append(args, filter.Values[0])
			fs.paramIndex++

		case OpGreaterThanOrEqual:
			conditions = append(conditions, fmt.Sprintf("%s >= %s", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
			args = append(args, filter.Values[0])
			fs.paramIndex++

		case OpLessThan:
			conditions = append(conditions, fmt.Sprintf("%s < %s", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
			args = append(args, filter.Values[0])
			fs.paramIndex++

		case OpLessThanOrEqual:
			conditions = append(conditions, fmt.Sprintf("%s <= %s", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
			args = append(args, filter.Values[0])
			fs.paramIndex++

		case OpLike:
			conditions = append(conditions, fmt.Sprintf("%s LIKE %s", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
			args = append(args, "%"+filter.Values[0]+"%")
			fs.paramIndex++

		case OpILike:
			method := fs.dialect.CaseInsensitiveLike()
			if method == "ILIKE" {
				conditions = append(conditions, fmt.Sprintf("%s ILIKE %s", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
				args = append(args, "%"+filter.Values[0]+"%")
			} else {
				conditions = append(conditions, fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", filter.Field, fs.dialect.Placeholder(fs.paramIndex)))
				args = append(args, "%"+filter.Values[0]+"%")
			}
			fs.paramIndex++

		case OpIn:
			placeholders := make([]string, len(filter.Values))
			for i, val := range filter.Values {
				placeholders[i] = fs.dialect.Placeholder(fs.paramIndex)
				args = append(args, val)
				fs.paramIndex++
			}
			conditions = append(conditions, fmt.Sprintf("%s IN (%s)", filter.Field, strings.Join(placeholders, ", ")))

		case OpNotIn:
			placeholders := make([]string, len(filter.Values))
			for i, val := range filter.Values {
				placeholders[i] = fs.dialect.Placeholder(fs.paramIndex)
				args = append(args, val)
				fs.paramIndex++
			}
			conditions = append(conditions, fmt.Sprintf("%s NOT IN (%s)", filter.Field, strings.Join(placeholders, ", ")))
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// Conditions returns all filters as query.Condition slice for use with query builder.
func (fs *FilterSet) Conditions() []query.Condition {
	var conditions []query.Condition
	for _, filter := range fs.Filters {
		if cond := fs.filterToCondition(filter); cond != nil {
			conditions = append(conditions, cond)
		}
	}
	return conditions
}

// filterToCondition converts a Filter to a query.Condition.
func (fs *FilterSet) filterToCondition(filter Filter) query.Condition {
	switch filter.Operator {
	case OpEqual:
		return query.Eq(filter.Field, filter.Values[0])
	case OpNotEqual:
		return query.Ne(filter.Field, filter.Values[0])
	case OpGreaterThan:
		return query.Gt(filter.Field, filter.Values[0])
	case OpGreaterThanOrEqual:
		return query.Gte(filter.Field, filter.Values[0])
	case OpLessThan:
		return query.Lt(filter.Field, filter.Values[0])
	case OpLessThanOrEqual:
		return query.Lte(filter.Field, filter.Values[0])
	case OpLike:
		return query.Like(filter.Field, "%"+filter.Values[0]+"%")
	case OpILike:
		return query.ILike(filter.Field, "%"+filter.Values[0]+"%")
	case OpIn:
		vals := make([]any, len(filter.Values))
		for i, v := range filter.Values {
			vals[i] = v
		}
		return query.In(filter.Field, vals...)
	case OpNotIn:
		vals := make([]any, len(filter.Values))
		for i, v := range filter.Values {
			vals[i] = v
		}
		return query.NotIn(filter.Field, vals...)
	}
	return nil
}
