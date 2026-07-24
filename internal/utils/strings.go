package utils

import "strings"

// JoinWithAnd joins a slice of strings with commas, using " AND " before the last element.
func JoinWithAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " AND " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " AND " + items[len(items)-1]
	}
}
