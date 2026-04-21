package group

import "strconv"

func parseIntParam(raw string, defaultVal int) (int, error) {
	if raw == "" {
		return defaultVal, nil
	}
	return strconv.Atoi(raw)
}

func queryStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
