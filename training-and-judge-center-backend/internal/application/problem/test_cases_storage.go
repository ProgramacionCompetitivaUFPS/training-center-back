package problem

// A problem stores the upload prefix, not the ZIP: sample files live there too.
const testCasesZipName = "testcases.zip"

// TestCasesZipKey resolves the uploaded ZIP inside the prefix a problem stores.
func TestCasesZipKey(prefix string) string {
	return prefix + "/" + testCasesZipName
}
