package problem

import "fmt"

// A judging update keeps everything it stages for the checker under one directory, apart from the live checker's files.
func stagedCheckerPrefix(slug, validationID string) string {
	return fmt.Sprintf("problems/%s/checker/%s/", slug, validationID)
}

func stagedCheckerCompiledKey(slug, validationID string) string {
	return stagedCheckerPrefix(slug, validationID) + "compiled"
}
