package response

import "regexp"

// internalDetail matches the shapes an error text takes when it came from the
// database driver, the runtime or the filesystem rather than from a validator.
//
// A driver message names the table, the column, its type and the SQLSTATE, all
// of which are free schema reconnaissance for whoever sent the bad input. A
// runtime message names source paths and line numbers.
var internalDetail = regexp.MustCompile(`(?i)SQLSTATE|invalid input syntax|pq: |pgx|syntax error at or near|relation "|column "|constraint "|goroutine \d|panic: |\.go:\d+|/home/|/usr/|no such file|connection refused|dial tcp`)

// ContainsInternalDetail reports whether a message looks like it escaped from
// a layer below the API.
func ContainsInternalDetail(msg string) bool {
	return internalDetail.MatchString(msg)
}

// SafeMessage returns an error's text when it was written for the caller, and
// the fallback when it was written for an operator.
//
// The rule it enforces is that a message reaches the client only when
// something meant it to. A validator writes for the caller; a driver writes
// for whoever is reading the logs, and forwarding its text is how `invalid
// input syntax for type uuid: "0" (SQLSTATE 22P02)` ended up in API responses
// across six plugins at once.
//
// Handlers should prefer a fixed string where one fits. This exists for the
// places that genuinely want to pass a validation message through and must not
// pass anything else.
func SafeMessage(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	msg := err.Error()
	if msg == "" || internalDetail.MatchString(msg) {
		return fallback
	}
	return msg
}
