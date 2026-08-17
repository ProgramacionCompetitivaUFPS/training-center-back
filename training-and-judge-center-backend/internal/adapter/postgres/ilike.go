package postgres

import "strings"

// EscapeILIKE escapes ILIKE wildcard characters in a user-supplied search
// term so it can be safely embedded in a "%...%" pattern.
func EscapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
