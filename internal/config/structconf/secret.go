package structconf

import (
	"reflect"
	"slices"
	"strings"
)

// secretTag marks a string setting whose value must never be shown with `secret:"true"`.
const secretTag = "secret"

// redactedValue replaces a set secret in Redacted.
const redactedValue = "[redacted]"

// secretField is a string field tagged as secret, addressed by the dotted path of its JSON names,
// since a configuration leaves the process as JSON.
type secretField struct {
	path  string
	value reflect.Value
}

// Redacted returns a copy of cfg in which every set secret is replaced by a placeholder,
// so the configuration can be logged or printed. Empty secrets stay empty,
// so the copy still shows which secrets are missing.
func Redacted[T any](cfg T) T {
	copied := cfg
	eachSecret(reflect.ValueOf(&copied).Elem(), "", func(f secretField) {
		if f.value.String() != "" {
			f.value.SetString(redactedValue)
		}
	})
	return copied
}

// HideSecrets clears every secret of cfg and reports by JSON path whether it was set,
// so a receiver learns which secrets exist without seeing them.
func HideSecrets(cfg any) map[string]bool {
	set := make(map[string]bool)
	eachSecret(reflect.ValueOf(cfg).Elem(), "", func(f secretField) {
		set[f.path] = f.value.String() != ""
		f.value.SetString("")
	})
	return set
}

// KeepSecrets fills the empty secrets of dst from current except those listed in clear by JSON path,
// since a form that never received the secrets sends them empty.
func KeepSecrets(dst, current any, clear []string) {
	currentValues := make(map[string]string)
	eachSecret(reflect.ValueOf(current).Elem(), "", func(f secretField) {
		currentValues[f.path] = f.value.String()
	})

	eachSecret(reflect.ValueOf(dst).Elem(), "", func(f secretField) {
		if f.value.String() != "" || slices.Contains(clear, f.path) {
			return
		}
		f.value.SetString(currentValues[f.path])
	})
}

// eachSecret calls fn for every secret string field of the struct v points to.
func eachSecret(v reflect.Value, prefix string, fn func(secretField)) {
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		fieldValue := v.Field(i)
		if fieldValue.Kind() == reflect.Struct {
			eachSecret(fieldValue, path, fn)
			continue
		}

		if field.Tag.Get(secretTag) == "true" && fieldValue.Kind() == reflect.String {
			fn(secretField{
				path:  path,
				value: fieldValue,
			})
		}
	}
}
