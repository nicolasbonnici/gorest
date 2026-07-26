package filter

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/nicolasbonnici/gorest/query"
)

type OrderDirection string

const (
	OrderAsc  OrderDirection = "asc"
	OrderDesc OrderDirection = "desc"
)

type Order struct {
	Field     string
	Direction OrderDirection
}

type OrderSet struct {
	Orders        []Order
	AllowedFields map[string]bool
	FieldMap      map[string]string // Maps JSON field names to DB column names
}

func NewOrderSet(allowedFields []string) *OrderSet {
	return NewOrderSetWithAllowedSet(AllowedSet(allowedFields))
}

// NewOrderSetWithAllowedSet lets a caller with a fixed field list build the set
// once (see AllowedSet) instead of paying for a map per request.
func NewOrderSetWithAllowedSet(allowed map[string]bool) *OrderSet {
	return &OrderSet{
		Orders:        []Order{},
		AllowedFields: allowed,
		FieldMap:      nil,
	}
}

// NewOrderSetWithMapping creates an OrderSet with field name mapping from JSON to DB columns.
func NewOrderSetWithMapping(fieldMap map[string]string) *OrderSet {
	allowed := make(map[string]bool)
	for jsonName := range fieldMap {
		allowed[jsonName] = true
	}
	return &OrderSet{
		Orders:        []Order{},
		AllowedFields: allowed,
		FieldMap:      fieldMap,
	}
}

func (os *OrderSet) ParseFromQuery(query url.Values) error {
	keys := make([]string, 0, len(query))
	for key := range query {
		if strings.HasPrefix(key, "order[") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	for _, key := range keys {
		if !strings.HasPrefix(key, "order[") || !strings.HasSuffix(key, "]") {
			continue
		}

		field := key[6 : len(key)-1]

		if !os.AllowedFields[field] {
			continue
		}

		values := query[key]
		if len(values) == 0 {
			continue
		}

		direction := OrderAsc
		dirStr := strings.ToLower(values[0])
		if dirStr == "desc" {
			direction = OrderDesc
		}

		// Map JSON field name to DB column name if mapping exists
		dbField := field
		if os.FieldMap != nil {
			if mapped, ok := os.FieldMap[field]; ok {
				dbField = mapped
			}
		}

		os.Orders = append(os.Orders, Order{
			Field:     dbField,
			Direction: direction,
		})
	}

	return nil
}

func (os *OrderSet) BuildOrderByClause() string {
	if len(os.Orders) == 0 {
		return ""
	}

	var parts []string
	for _, order := range os.Orders {
		direction := "ASC"
		if order.Direction == OrderDesc {
			direction = "DESC"
		}
		parts = append(parts, fmt.Sprintf("%s %s", order.Field, direction))
	}

	return "ORDER BY " + strings.Join(parts, ", ")
}

// OrderClause represents a column ordering for use with query builder.
type OrderClause struct {
	Column    string
	Direction query.Order
}

// OrderClauses returns all orders as OrderClause slice for use with query builder.
func (os *OrderSet) OrderClauses() []OrderClause {
	clauses := make([]OrderClause, 0, len(os.Orders))
	for _, order := range os.Orders {
		direction := query.ASC
		if order.Direction == OrderDesc {
			direction = query.DESC
		}
		clauses = append(clauses, OrderClause{
			Column:    order.Field,
			Direction: direction,
		})
	}
	return clauses
}
