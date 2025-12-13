package filter

import (
	"net/url"
	"strings"
)

type ExpandSet struct {
	Relations     []string
	AllowedFields map[string]bool
}

func NewExpandSet(allowedRelations []string) *ExpandSet {
	allowed := make(map[string]bool)
	for _, rel := range allowedRelations {
		allowed[rel] = true
	}
	return &ExpandSet{
		Relations:     []string{},
		AllowedFields: allowed,
	}
}

func (es *ExpandSet) ParseFromQuery(query url.Values) error {
	seen := make(map[string]bool)

	for key, values := range query {
		if !strings.HasPrefix(key, "expand") {
			continue
		}

		if strings.HasSuffix(key, "[]") {
			for _, value := range values {
				if es.AllowedFields[value] && !seen[value] {
					es.Relations = append(es.Relations, value)
					seen[value] = true
				}
			}
		}
	}

	return nil
}

func (es *ExpandSet) Has(relation string) bool {
	for _, r := range es.Relations {
		if r == relation {
			return true
		}
	}
	return false
}
