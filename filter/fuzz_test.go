package filter

import (
	"net/url"
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/postgres"
)

// Filter, ordering and expand parameters are raw attacker input that ends up
// shaping SQL, so they are the highest-value fuzz target in the codebase. The
// invariants asserted here are the ones that matter for injection: no panic,
// and nothing outside the allowlist ever reaches the generated clause.

var fuzzAllowed = []string{"id", "name", "email", "status", "created_at"}

func dialect() *postgres.PostgresDialect { return &postgres.PostgresDialect{} }

func FuzzFilterParsing(f *testing.F) {
	for _, seed := range []string{
		"filter[name][eq]=alice",
		"filter[id][in]=1,2,3",
		"filter[name][like]=%25a%25",
		"filter[status][eq]=x' OR '1'='1",
		"filter[name][eq]=x; DROP TABLE users--",
		"filter[secret][eq]=1",
		"filter[name]=alice",
		"filter[][]=",
		"filter[name][nosuchop]=1",
		"filter[created_at][gt]=2020-01-01",
		strings.Repeat("filter[name][eq]=a&", 50),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		q, err := url.ParseQuery(raw)
		if err != nil {
			return
		}

		fs := NewFilterSet(fuzzAllowed, dialect())
		if err := fs.ParseFromQuery(q); err != nil {
			return
		}

		clause, args := fs.BuildWhereClause()

		// Every value must be parameterised. A literal quote in the clause
		// means a value was interpolated rather than bound.
		if strings.Contains(clause, "'") {
			t.Fatalf("quote in generated clause %q (args %v) from %q", clause, args, raw)
		}
		for _, banned := range []string{";", "--", "/*"} {
			if strings.Contains(clause, banned) {
				t.Fatalf("%q in generated clause %q from %q", banned, clause, raw)
			}
		}

		// Only allowlisted columns may be named.
		for _, f := range fs.Filters {
			if !isAllowed(f.Field) {
				t.Fatalf("field %q passed the allowlist, from %q", f.Field, raw)
			}
		}
	})
}

func FuzzOrderParsing(f *testing.F) {
	for _, seed := range []string{
		"sort=name",
		"sort=-created_at",
		"sort=name,-id",
		"sort=name;DROP TABLE users",
		"sort=(SELECT 1)",
		"sort=secret",
		"sort=",
		"sort=" + strings.Repeat("name,", 100),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		q, err := url.ParseQuery(raw)
		if err != nil {
			return
		}

		os := NewOrderSet(fuzzAllowed)
		if err := os.ParseFromQuery(q); err != nil {
			return
		}

		clause := os.BuildOrderByClause()
		for _, banned := range []string{";", "--", "/*", "'", "("} {
			if strings.Contains(clause, banned) {
				t.Fatalf("%q in ORDER BY clause %q from %q", banned, clause, raw)
			}
		}
		for _, c := range os.OrderClauses() {
			if !isAllowed(c.Column) {
				t.Fatalf("field %q passed the allowlist, from %q", c.Column, raw)
			}
		}
	})
}

func FuzzExpandParsing(f *testing.F) {
	for _, seed := range []string{"expand=author", "expand=author,comments", "expand=../../etc/passwd", "expand=" + strings.Repeat("a,", 200)} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		q, err := url.ParseQuery(raw)
		if err != nil {
			return
		}
		es := NewExpandSet([]string{"author", "comments"})
		if err := es.ParseFromQuery(q); err != nil {
			return
		}
		for _, rel := range []string{"author", "comments"} {
			_ = es.Has(rel)
		}
	})
}

func isAllowed(field string) bool {
	for _, a := range fuzzAllowed {
		if a == field {
			return true
		}
	}
	return false
}
