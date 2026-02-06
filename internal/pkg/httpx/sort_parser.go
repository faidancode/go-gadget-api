package httpx

import (
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

type SortResult struct {
	SortBy  string
	SortDir string
}

// ParseSort mendukung:
// - legacy: ?sort=createdAt:desc
// - new:    ?sort_col=created_at&sort_dir=desc
func ParseSort(
	c *gin.Context,
	defaultBy string,
	defaultDir string,
) SortResult {
	sortBy := defaultBy
	sortDir := defaultDir

	// legacy support: ?sort=createdAt:desc
	if legacy := c.Query("sort"); legacy != "" {
		parts := strings.Split(legacy, ":")
		if len(parts) == 2 {
			sortBy = toSnakeCase(parts[0])
			sortDir = strings.ToLower(parts[1])
			return SortResult{SortBy: sortBy, SortDir: sortDir}
		}
	}

	// new format
	sortBy = c.DefaultQuery("sort_col", sortBy)
	sortDir = strings.ToLower(c.DefaultQuery("sort_dir", sortDir))

	return SortResult{SortBy: sortBy, SortDir: sortDir}
}

// simple camelCase → snake_case
func toSnakeCase(s string) string {
	var out []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}
