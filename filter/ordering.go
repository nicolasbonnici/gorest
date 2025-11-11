package filter

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
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
}

func NewOrderSet(allowedFields []string) *OrderSet {
	allowed := make(map[string]bool)
	for _, field := range allowedFields {
		allowed[field] = true
	}
	return &OrderSet{
		Orders:        []Order{},
		AllowedFields: allowed,
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

		os.Orders = append(os.Orders, Order{
			Field:     field,
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
