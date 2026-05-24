package judge

type LanguageExecConfig struct {
	CompileCmd string // space-separated args; empty = interpreted language, skip compile
	RunCmd     string // space-separated run command
	Extension  string // without leading dot: "cpp", "java", "py"
}

type ExecutorConfig struct {
	Languages map[string]LanguageExecConfig
}
