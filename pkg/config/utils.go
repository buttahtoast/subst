package config

import (
	"regexp"
)

func getValueAfterUnderscore(input string) string {
	re, _ := regexp.Compile("_(.+)")

	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		// No match found or the part after underscore is missing
		return input
	}

	return matches[1]
}
