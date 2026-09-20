package crud

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

type uuidKeyed struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

func (uuidKeyed) TableName() string { return "uuid_keyed" }

type intKeyed struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

func (intKeyed) TableName() string { return "int_keyed" }

type stringKeyed struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

func (stringKeyed) TableName() string { return "string_keyed" }

func TestCheckIDRejectsWhatTheDriverWouldReject(t *testing.T) {
	meta := getFieldMeta(reflect.TypeOf(uuidKeyed{}))

	// Every one of these reached PostgreSQL before this guard existed and came
	// back as `invalid input syntax for type uuid: "…" (SQLSTATE 22P02)`.
	for _, id := range []string{
		"0", "-1", "null", "%00", "1e309", "%ff%fe",
		"'%20OR%20'1'='1", "..%2f..%2fetc%2fpasswd", "",
	} {
		if err := checkID(meta, id); err == nil {
			t.Errorf("checkID(uuid key, %q) = nil, want ErrInvalidID", id)
		}
	}

	if err := checkID(meta, uuid.New().String()); err != nil {
		t.Errorf("checkID rejected a well-formed UUID: %v", err)
	}
}

func TestErrInvalidIDReadsAsNotFound(t *testing.T) {
	meta := getFieldMeta(reflect.TypeOf(uuidKeyed{}))
	err := checkID(meta, "0")

	// The whole point of wrapping: a handler written against IsNotFoundError
	// answers 404 with no change, and the key's type stays undisclosed.
	if !errors.Is(err, sql.ErrNoRows) {
		t.Error("ErrInvalidID must wrap sql.ErrNoRows so existing handlers answer 404")
	}
	if !IsNotFoundError(err) {
		t.Error("IsNotFoundError must report an invalid id as not found")
	}
	if !IsInvalidIDError(err) {
		t.Error("IsInvalidIDError must still distinguish it for callers that care")
	}
}

func TestCheckIDByKeyType(t *testing.T) {
	intMeta := getFieldMeta(reflect.TypeOf(intKeyed{}))
	if err := checkID(intMeta, "12"); err != nil {
		t.Errorf("checkID(int key, \"12\") = %v, want nil", err)
	}
	if err := checkID(intMeta, "not-a-number"); err == nil {
		t.Error("checkID(int key, \"not-a-number\") = nil, want ErrInvalidID")
	}
	// A UUID is not an integer, so it cannot address a row in this table.
	if err := checkID(intMeta, uuid.New().String()); err == nil {
		t.Error("checkID(int key, uuid) = nil, want ErrInvalidID")
	}

	// A string key can hold anything, so nothing is rejected locally.
	strMeta := getFieldMeta(reflect.TypeOf(stringKeyed{}))
	for _, id := range []string{"0", "anything", "../../etc/passwd", ""} {
		if err := checkID(strMeta, id); err != nil {
			t.Errorf("checkID(string key, %q) = %v, want nil", id, err)
		}
	}
}

func TestCheckIDLeavesParsedValuesAlone(t *testing.T) {
	meta := getFieldMeta(reflect.TypeOf(uuidKeyed{}))

	// A caller holding a real uuid.UUID has already parsed it; re-checking a
	// non-string would only be a chance to reject something valid.
	if err := checkID(meta, uuid.New()); err != nil {
		t.Errorf("checkID rejected a uuid.UUID value: %v", err)
	}
	if err := checkID(nil, "anything"); err != nil {
		t.Errorf("checkID with no metadata must not reject: %v", err)
	}
}

func TestIsDuplicateErrorAcrossEngines(t *testing.T) {
	for _, tc := range []struct {
		engine string
		err    error
	}{
		{"postgres", errors.New(`ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)`)},
		{"mysql", errors.New("Error 1062 (23000): Duplicate entry 'a@b.test' for key 'users.email'")},
		{"sqlite", errors.New("UNIQUE constraint failed: users.email")},
	} {
		if !IsDuplicateError(tc.err) {
			t.Errorf("IsDuplicateError did not recognise the %s unique violation", tc.engine)
		}
	}

	if IsDuplicateError(nil) {
		t.Error("IsDuplicateError(nil) = true")
	}
	if IsDuplicateError(errors.New("connection refused")) {
		t.Error("IsDuplicateError matched an unrelated error")
	}
}
