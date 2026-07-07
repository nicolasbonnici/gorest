package serializer

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The fast streaming path must produce JSON that decodes identically to the
// map-based path for every shape it claims to handle. This is the guardrail
// that lets fastld.go stay in the hot path.

type equivNested struct {
	City string `json:"city"`
	Zip  string `json:"zip"`
}

type equivAll struct {
	ID        uuid.UUID    `json:"id"`
	AuthorID  uuid.UUID    `json:"author_id"` // uuid FK: NOT rewritten
	OwnerID   string       `json:"owner_id"`  // string FK: rewritten to IRI
	TeamId    string       `json:"teamId"`    // camel string FK: rewritten
	Name      string       `json:"name"`
	Note      string       `json:"note,omitempty"`
	Weird     string       `json:"weird"` // needs escaping
	Views     int          `json:"views"`
	Ratio     float64      `json:"ratio"`
	Active    bool         `json:"active"`
	Address   equivNested  `json:"address"`
	Optional  *equivNested `json:"optional"`
	CreatedAt time.Time    `json:"created_at"`
}

func decodeJSON(t *testing.T, b []byte) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, b)
	}
	return v
}

func assertFastMatchesSlow(t *testing.T, data interface{}, path string) {
	t.Helper()
	s := &JSONLDSerializer{}

	fast, ok := fastSerializeLD(data, path)
	if !ok {
		t.Fatalf("fast path unexpectedly declined for %T", data)
	}
	slow, err := json.Marshal(s.wrapWithContextExpand(data, path, nil))
	if err != nil {
		t.Fatalf("slow marshal: %v", err)
	}

	if !reflect.DeepEqual(decodeJSON(t, fast), decodeJSON(t, slow)) {
		t.Fatalf("fast != slow\n fast: %s\n slow: %s", fast, slow)
	}
}

func TestFastLD_EquivalentToMapPath(t *testing.T) {
	owner := uuid.New().String()
	team := uuid.New().String()
	item := equivAll{
		ID:        uuid.New(),
		AuthorID:  uuid.New(),
		OwnerID:   owner,
		TeamId:    team,
		Name:      "A normal name",
		Note:      "", // omitempty → dropped
		Weird:     `he said "hi" <b>&</b>` + "\n\ttab line",
		Views:     42,
		Ratio:     3.14,
		Active:    true,
		Address:   equivNested{City: "Paris", Zip: "75001"},
		Optional:  &equivNested{City: "Lyon", Zip: "69001"},
		CreatedAt: time.Date(2026, 7, 7, 22, 31, 52, 0, time.UTC),
	}

	// Slice (the @graph path)
	assertFastMatchesSlow(t, []equivAll{item, item}, "/things")
	// Single struct (inline path)
	assertFastMatchesSlow(t, item, "/things")
	// Non-omitempty populated note
	item.Note = "present"
	assertFastMatchesSlow(t, []equivAll{item}, "/things")
	// Nil pointer field
	item.Optional = nil
	assertFastMatchesSlow(t, []equivAll{item}, "/things")
	// Path already ending with the id (no double-append in @id)
	assertFastMatchesSlow(t, item, "/things/"+item.ID.String())
}

// An empty-string FK must fall back to the map path (which keeps the original
// key) rather than emit a bogus IRI.
func TestFastLD_EmptyStringFKFallsBack(t *testing.T) {
	item := equivAll{ID: uuid.New(), OwnerID: "", TeamId: uuid.New().String()}
	if _, ok := fastSerializeLD([]equivAll{item}, "/things"); ok {
		t.Fatal("expected fast path to decline on empty string FK")
	}
}
