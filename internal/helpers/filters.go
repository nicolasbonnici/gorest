package helpers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nicolasbonnici/gorest/pkg/database"
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
)

type Filter struct {
	Field    string
	Operator FilterOperator
	Values   []string
}

type FilterSet struct {
	Filters        []Filter
	AllowedFields  map[string]bool
	paramIndex     int
	dialect        database.Dialect
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

		if operator == OpIn && len(values) == 1 {
			filter.Operator = OpEqual
		}

		fs.Filters = append(fs.Filters, filter)
	}

	return nil
}

func (fs *FilterSet) parseFieldAndOperator(key string) (string, FilterOperator) {
	if strings.HasSuffix(key, "[]") {
		return strings.TrimSuffix(key, "[]"), OpIn
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
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}
