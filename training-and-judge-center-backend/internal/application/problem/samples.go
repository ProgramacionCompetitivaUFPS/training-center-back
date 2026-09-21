package problem

import (
	"context"
	"path"
	"sort"
	"strings"
)

type SampleTestCase struct {
	Name   string
	Input  string
	Output string
}

// loadSamples pairs the .in/.ans files under {testCasesKey}/data/sample/ by basename,
// discarding any file whose counterpart is missing. Returns an empty slice (not nil)
// when no test cases have been uploaded yet.
func loadSamples(ctx context.Context, storage ProblemFileRepository, testCasesKey *string) ([]SampleTestCase, error) {
	if testCasesKey == nil {
		return []SampleTestCase{}, nil
	}

	prefix := *testCasesKey + "/data/sample/"
	paths, err := storage.ListFiles(ctx, prefix)
	if err != nil {
		return nil, err
	}

	inputs := make(map[string]string, len(paths))
	outputs := make(map[string]string, len(paths))
	for _, filePath := range paths {
		ext := path.Ext(filePath)
		name := strings.TrimSuffix(path.Base(filePath), ext)

		content, err := storage.DownloadFile(ctx, filePath)
		if err != nil {
			return nil, err
		}

		switch ext {
		case ".in":
			inputs[name] = string(content)
		case ".ans":
			outputs[name] = string(content)
		}
	}

	names := make([]string, 0, len(inputs))
	for name := range inputs {
		if _, ok := outputs[name]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	samples := make([]SampleTestCase, 0, len(names))
	for _, name := range names {
		samples = append(samples, SampleTestCase{Name: name, Input: inputs[name], Output: outputs[name]})
	}

	return samples, nil
}
