package handler

import "regexp"

var digitCodeRegex = regexp.MustCompile(`^\d{6}$`)
