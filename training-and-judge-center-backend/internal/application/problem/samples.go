package problem

import (
	"context"
	"path"
	"sort"
	"strconv"
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

	// Everything under the key, not just <key>/data/sample/: a package zipped inside a root folder
	// ("pkg/data/sample/1.in") was stored with that folder in the path, and looking only at
	// <key>/data/sample/ made its examples vanish from the problem page.
	base := *testCasesKey + "/"
	paths, err := storage.ListFiles(ctx, base)
	if err != nil {
		return nil, err
	}

	inputs := make(map[string]string, len(paths))
	outputs := make(map[string]string, len(paths))
	for _, filePath := range paths {
		if !isSampleFile(strings.TrimPrefix(filePath, base)) {
			continue
		}
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
	sortSampleNames(names)

	samples := make([]SampleTestCase, 0, len(names))
	for _, name := range names {
		samples = append(samples, SampleTestCase{Name: name, Input: inputs[name], Output: outputs[name]})
	}

	return samples, nil
}

// sortSampleNames sorts numerically (so "10" doesn't land before "2") only when every
// name is a plain number, matching the usual ICPC sample naming convention (1.in, 2.in, ...).
// If any name doesn't parse as a number, it falls back to plain alphabetical order for the
// whole set, rather than mixing both criteria.
func sortSampleNames(names []string) {
	values := make(map[string]int, len(names))
	for _, name := range names {
		n, err := strconv.Atoi(name)
		if err != nil {
			sort.Strings(names)
			return
		}
		values[name] = n
	}

	sort.Slice(names, func(i, j int) bool {
		return values[names[i]] < values[names[j]]
	})
}

// isSampleFile reports whether a path (relative to the test cases key) is an .in/.ans of a
// data/sample directory, wherever the package's root folder is.
func isSampleFile(rel string) bool {
	if ext := path.Ext(rel); ext != ".in" && ext != ".ans" {
		return false
	}
	dir := path.Dir(rel)
	return dir == "data/sample" || strings.HasSuffix(dir, "/data/sample")
}
