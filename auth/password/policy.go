// Package password enforces what a new credential must look like before it is
// hashed and stored.
//
// The policy follows NIST SP 800-63B: length is the control that matters, and
// screening against known-common passwords catches what length alone lets
// through. Composition rules (one upper, one digit, one symbol) are
// deliberately absent; they push people towards predictable shapes like
// Password1! without adding real entropy, and 800-63B recommends against them.
package password

import (
	"fmt"
	"strings"
	"unicode"
)

// DefaultMinLength is the floor when the configuration does not set one.
const DefaultMinLength = 12

// MaxLength bounds what is accepted. bcrypt silently truncates beyond 72 bytes,
// so a longer secret is not the secret the user believes it is, and accepting
// megabytes of passphrase is a cheap way to make the hash function expensive.
const MaxLength = 72

// Policy is the set of rules applied to a new password.
type Policy struct {
	// MinLength is the minimum number of characters. Zero means DefaultMinLength.
	MinLength int
	// Blocklist screens against the common-password list when true.
	Blocklist bool
}

// Error reports a password the policy refuses, with a reason meant for the
// person choosing it. The text names what to fix and never echoes the password.
type Error struct {
	Reason string
}

func (e *Error) Error() string { return e.Reason }

// Validate reports whether the password may be used.
func (p Policy) Validate(pw string) error {
	minLen := p.MinLength
	if minLen <= 0 {
		minLen = DefaultMinLength
	}

	// Counted in runes: a passphrase in a non-Latin script is not shorter for
	// being written in fewer bytes than ASCII would need.
	if n := len([]rune(pw)); n < minLen {
		return &Error{Reason: fmt.Sprintf("password must be at least %d characters", minLen)}
	}

	// Measured in bytes, because 72 is bcrypt's byte limit, not a rune limit.
	if len(pw) > MaxLength {
		return &Error{Reason: fmt.Sprintf("password must be at most %d bytes", MaxLength)}
	}

	if strings.TrimSpace(pw) == "" {
		return &Error{Reason: "password must not be only whitespace"}
	}

	if p.Blocklist && IsCommon(pw) {
		return &Error{Reason: "password is too common; choose a less predictable one"}
	}

	return nil
}

// normalize folds the variations people use to dress up a common password so
// the blocklist does not have to carry every one of them. "P@ssw0rd!" and
// "password" collapse to the same entry.
func normalize(pw string) string {
	var b strings.Builder
	b.Grow(len(pw))
	for _, r := range strings.ToLower(pw) {
		switch r {
		case '@':
			b.WriteRune('a')
		case '0':
			b.WriteRune('o')
		case '1', '!', '|':
			b.WriteRune('i')
		case '3':
			b.WriteRune('e')
		case '4':
			b.WriteRune('a')
		case '5', '$':
			b.WriteRune('s')
		case '7':
			b.WriteRune('t')
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// IsCommon reports whether the password is on, or trivially derived from, the
// common-password list.
//
// Rather than carry every dressed-up spelling in the list, the password is
// reduced to a small set of candidate base forms and each is looked up. The
// order matters: trailing decoration is stripped before leetspeak is folded,
// because folding turns digits into letters and would otherwise hide the
// trailing digits from the strip.
func IsCommon(pw string) bool {
	lower := strings.ToLower(pw)

	// A single repeated character reads as long while carrying almost no
	// entropy, and no wordlist could hold every length of it.
	if runes := []rune(lower); len(runes) > 0 {
		same := true
		for _, r := range runes {
			if r != runes[0] {
				same = false
				break
			}
		}
		if same {
			return true
		}
	}

	seen := make(map[string]struct{}, 8)
	var candidates []string
	add := func(c string) {
		if c == "" {
			return
		}
		if _, dup := seen[c]; dup {
			return
		}
		seen[c] = struct{}{}
		candidates = append(candidates, c)
	}

	for _, base := range []string{lower, stripDecoration(lower)} {
		add(base)
		add(normalize(base))
		// letmeinletmein and adminadmin are one word typed twice, which every
		// length rule waves through and no list enumerates.
		add(halfIfDoubled(base))
		add(halfIfDoubled(normalize(base)))
	}

	for _, c := range candidates {
		if len(c) < 3 {
			continue
		}
		if _, hit := commonPasswords[c]; hit {
			return true
		}
	}

	return false
}

// stripDecoration removes the trailing digits and punctuation used to get a
// common word past a blocklist: password2024, letmein!!, qwerty_1.
func stripDecoration(s string) string {
	trimmed := strings.TrimRightFunc(s, func(r rune) bool {
		return unicode.IsDigit(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})
	// Refuse to strip so much that anything would match: "1234" must not
	// become the empty string and then be compared against nothing.
	if len([]rune(trimmed)) < 3 || len(s)-len(trimmed) > 6 {
		return ""
	}
	return trimmed
}

// halfIfDoubled returns the first half of a string that is one value repeated
// exactly twice, and the empty string otherwise.
func halfIfDoubled(s string) string {
	if len(s) < 6 || len(s)%2 != 0 {
		return ""
	}
	half := len(s) / 2
	if s[:half] != s[half:] {
		return ""
	}
	return s[:half]
}
