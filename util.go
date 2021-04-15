package svgg

import (
	"strconv"
	"strings"
)

// unitSuffixes are suffixes sometimes applied to the width and height attributes
// of the svg element.
var unitSuffixes = []string{"cm", "mm", "px", "pt"}

// trimSuffixes removes unitSuffixes from any number that is not just numeric
func trimSuffixes(a string) (b string) {
	if a == "" || (a[len(a)-1] >= '0' && a[len(a)-1] <= '9') {
		return a
	}
	b = a
	for _, v := range unitSuffixes {
		b = strings.TrimSuffix(b, v)
	}
	return
}

// parseFloat is a helper function that strips suffixes before passing to strconv.ParseFloat
func parseFloat(s string, bitSize int) (float64, error) {
	val := trimSuffixes(s)
	return strconv.ParseFloat(val, bitSize)
}
