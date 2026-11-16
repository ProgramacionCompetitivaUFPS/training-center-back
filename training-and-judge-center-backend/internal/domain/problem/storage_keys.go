package problem

type StorageKeys struct {
	TestCasesPrefix string
	Files           []string
}

func (p *Problem) StorageKeys() StorageKeys {
	keys := StorageKeys{}
	if p.testCasesKey != nil {
		keys.TestCasesPrefix = *p.testCasesKey
	}
	for _, solution := range p.solutions {
		keys.Files = append(keys.Files, solution.fileKey)
	}
	keys.Files = appendJudgingFileKeys(keys.Files, p.checker)
	keys.Files = appendJudgingFileKeys(keys.Files, p.validator)
	return keys
}

func appendJudgingFileKeys(files []string, file *JudgingFile) []string {
	if file == nil {
		return files
	}
	files = append(files, file.fileKey)
	if file.compiledKey != nil {
		files = append(files, *file.compiledKey)
	}
	return files
}

func (k StorageKeys) NoLongerIn(current StorageKeys) StorageKeys {
	gone := StorageKeys{}
	if k.TestCasesPrefix != current.TestCasesPrefix {
		gone.TestCasesPrefix = k.TestCasesPrefix
	}
	kept := make(map[string]struct{}, len(current.Files))
	for _, file := range current.Files {
		kept[file] = struct{}{}
	}
	for _, file := range k.Files {
		if _, ok := kept[file]; !ok {
			gone.Files = append(gone.Files, file)
		}
	}
	return gone
}
