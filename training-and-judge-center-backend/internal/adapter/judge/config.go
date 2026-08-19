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

type ArtifactLanguageConfig struct {
	SourcePath   string // where the uploaded source is written inside the sandbox
	CompileCmd   string // run through sh -c, so it may chain commands with &&
	ArtifactPath string // what gets extracted once the command succeeds
}

type ArtifactCompilerConfig struct {
	Languages map[string]ArtifactLanguageConfig
}
