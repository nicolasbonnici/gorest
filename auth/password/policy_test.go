package password

import "testing"

func TestPolicyRejectsWhatTheAuditFound(t *testing.T) {
	p := Policy{Blocklist: true}

	// The four the pentest suite registered successfully before this existed.
	for _, pw := range []string{"a", "12345678", "password", ""} {
		if err := p.Validate(pw); err == nil {
			t.Errorf("Validate(%q) = nil, want a rejection", pw)
		}
	}
}

func TestPolicyLength(t *testing.T) {
	p := Policy{MinLength: 12}

	if err := p.Validate("elevenchars"); err == nil {
		t.Error("an 11-character password was accepted against a minimum of 12")
	}
	if err := p.Validate("twelvechars!"); err != nil {
		t.Errorf("a 12-character password was refused: %v", err)
	}

	// Runes, not bytes: a passphrase in a non-Latin script is not shorter for
	// needing fewer characters than ASCII would.
	if err := p.Validate("日本語のパスワードです"); err == nil {
		t.Error("an 11-rune password was accepted against a minimum of 12")
	}
	if err := p.Validate("日本語のパスワードですよ"); err != nil {
		t.Errorf("a 12-rune password was refused: %v", err)
	}

	// bcrypt truncates past 72 bytes, so a longer secret is not the one the
	// user thinks they chose.
	long := make([]byte, MaxLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if err := p.Validate(string(long)); err == nil {
		t.Error("a password longer than bcrypt's input limit was accepted")
	}
}

func TestBlocklistCatchesDressedUpCommonPasswords(t *testing.T) {
	p := Policy{MinLength: 8, Blocklist: true}

	// Each of these is long enough to pass a length check on its own.
	for _, pw := range []string{
		"password1234", "P@ssw0rd2024", "letmein99999", "qwertyuiop12",
		"iloveyou1234", "aaaaaaaaaaaa", "motdepasse12", "adminadmin",
	} {
		if err := p.Validate(pw); err == nil {
			t.Errorf("Validate(%q) = nil, want the blocklist to refuse it", pw)
		}
	}
}

func TestBlocklistAcceptsAPassphrase(t *testing.T) {
	p := Policy{MinLength: 12, Blocklist: true}

	for _, pw := range []string{
		"correct horse battery staple",
		"Str0ng-Pentest-Passw0rd-123",
		"vault-tumbler-orchid-9182",
	} {
		if err := p.Validate(pw); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", pw, err)
		}
	}
}

func TestBlocklistOffLeavesLengthOnly(t *testing.T) {
	p := Policy{MinLength: 8, Blocklist: false}

	if err := p.Validate("password"); err != nil {
		t.Errorf("with the blocklist off, only length applies: %v", err)
	}
	if err := p.Validate("short"); err == nil {
		t.Error("length must still apply with the blocklist off")
	}
}

func TestErrorTextNeverEchoesThePassword(t *testing.T) {
	p := Policy{MinLength: 12, Blocklist: true}
	// Not the word "password": the rejection text legitimately contains it,
	// and asserting on a secret that is a substring of the message tests the
	// wording rather than the leak.
	secret := "qq"

	err := p.Validate(secret)
	if err == nil {
		t.Fatal("expected a rejection")
	}
	// The message is shown to the user and may be logged by a client; the
	// rejected password must not travel with it.
	if contains(err.Error(), secret) {
		t.Errorf("rejection message contains the password: %q", err.Error())
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		})()
}
