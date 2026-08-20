package judge

type LanguageExecConfig struct {
	CompileCmd string // space-separated args; empty = interpreted language, skip compile
	RunCmd     string // space-separated run command
	Extension  string // without leading dot: "cpp", "java", "py"
}

type ExecutorConfig struct {
	Languages map[string]LanguageExecConfig
}

// ArtifactNamePlaceholder is what the artifact* commands carry in place of the
// artifact's role name.
const ArtifactNamePlaceholder = "{name}"

// ArtifactLanguageConfig is how one language's checker or validator is built
// and run. The two halves belong to different pools: building happens in the
// heavy one, running in the light one.
type ArtifactLanguageConfig struct {
	SourcePath   string // where the uploaded source is written inside the sandbox
	CompileCmd   string // run through sh -c, so it may chain commands with &&
	ArtifactPath string // extracted once built, and injected again before running
	RunCmd       string // run through sh -c
}

type ArtifactConfig struct {
	Languages map[string]ArtifactLanguageConfig
}
