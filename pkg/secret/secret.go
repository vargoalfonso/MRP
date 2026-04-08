// Package secret provides helpers for masking sensitive values in logs and fmt output.
//
// Use Mask() in log statements to prevent passwords, tokens, and other secrets
// from leaking into log output:
//
//	slog.String("password", secret.Mask(req.Password))
//	slog.String("token",    secret.MaskPartial(token))
//	slog.String("email",    secret.MaskEmail(email))
package secret

import (
	"log/slog"
	"strings"
)

// Mask replaces the entire value with "***".
// Use this for passwords, JWT secrets, API keys.
func Mask(_ string) string { return "***" }

// MaskPartial shows the first 3 and last 3 characters, masking the middle.
// For short strings (≤6 chars) it returns "***".
// Use this for tokens or IDs where partial visibility helps with debugging.
//
//	"eyJhbGciOi...xyz" → "eyJ***xyz"
func MaskPartial(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "***" + s[len(s)-3:]
}

// MaskEmail masks the local part of an email, keeping the domain visible.
//
//	"john.doe@example.com" → "jo***@example.com"
func MaskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 2 {
		return "***" + domain
	}
	return local[:2] + "***" + domain
}

// MaskStruct returns a slog.Value with masked fields.
// Pass key-value pairs where odd indices are field names (string)
// and even indices are already-masked values.
//
//	secret.MaskFields("password", req.Password, "token", token)
func MaskFields(keysAndValues ...string) slog.Value {
	if len(keysAndValues)%2 != 0 {
		return slog.StringValue("invalid mask fields args")
	}
	attrs := make([]slog.Attr, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		attrs = append(attrs, slog.String(keysAndValues[i], keysAndValues[i+1]))
	}
	return slog.GroupValue(attrs...)
}

// Secret is a string type that always masks itself in fmt and slog output.
// Assign secrets to this type so they cannot leak through accidental logging.
//
//	type MyConfig struct {
//	    DBPassword secret.Secret
//	}
//	cfg.DBPassword = secret.Secret(os.Getenv("DB_PASSWORD"))
//	// use the real value:
//	dsn := "password=" + cfg.DBPassword.Value()
type Secret string

// Value returns the underlying plaintext string.
func (s Secret) Value() string { return string(s) }

// String implements fmt.Stringer — returns "***".
func (s Secret) String() string { return "***" }

// GoString implements fmt.GoStringer — returns masked representation.
func (s Secret) GoString() string { return `secret.Secret("***")` }

// MarshalText implements encoding.TextMarshaler — masks in JSON/YAML output.
func (s Secret) MarshalText() ([]byte, error) { return []byte("***"), nil }

// LogValue implements slog.LogValuer — masks in structured log output.
func (s Secret) LogValue() slog.Value { return slog.StringValue("***") }
