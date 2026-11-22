package problem_test

import (
	"reflect"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

type settingsArgs struct {
	globalTime, globalMemory int
	languageLimits           map[string]problem.LanguageLimit
	supported                map[string]struct{}
	extensions               map[string]string
	tags                     map[string]struct{}
	uploadConcurrency        int
	sample                   int
	zipMB                    int
	testCase                 int
	defaultMB                int
}

func validSettingsArgs(t *testing.T) settingsArgs {
	t.Helper()
	cpp, err := problem.NewLanguageLimit("cpp20", 2000, 512)
	if err != nil {
		t.Fatalf("NewLanguageLimit: %v", err)
	}
	return settingsArgs{
		globalTime:        5000,
		globalMemory:      1024,
		languageLimits:    map[string]problem.LanguageLimit{"cpp20": cpp},
		supported:         map[string]struct{}{"cpp20": {}, "python310": {}},
		extensions:        map[string]string{".cpp": "cpp20", ".py": "python310"},
		tags:              map[string]struct{}{"dp": {}, "graphs": {}},
		uploadConcurrency: 4,
		sample:            3,
		zipMB:             100,
		testCase:          500,
		defaultMB:         10,
	}
}

func (a settingsArgs) build() (problem.PlatformSettings, error) {
	return problem.NewPlatformSettings(
		a.globalTime, a.globalMemory, a.languageLimits, a.supported, a.extensions, a.tags,
		a.uploadConcurrency, a.sample, a.zipMB, a.testCase, a.defaultMB,
	)
}

func TestNewPlatformSettings_Rejections(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(a *settingsArgs)
		wantFields []string
	}{
		{"a global time limit of zero", func(a *settingsArgs) { a.globalTime = 0 }, []string{"globalMaxTimeLimit"}},
		{"a negative global memory limit", func(a *settingsArgs) { a.globalMemory = -1 }, []string{"globalMaxMemoryLimit"}},
		{"an upload concurrency of zero", func(a *settingsArgs) { a.uploadConcurrency = 0 }, []string{"uploadMaxConcurrency"}},
		{"a sample file count of zero", func(a *settingsArgs) { a.sample = 0 }, []string{"maxFileCountSample"}},
		{"a zip size of zero", func(a *settingsArgs) { a.zipMB = 0 }, []string{"maxZipUploadSizeMB"}},
		{"a test case file count of zero", func(a *settingsArgs) { a.testCase = 0 }, []string{"maxFileCountTestCase"}},
		{"a default file size of zero", func(a *settingsArgs) { a.defaultMB = 0 }, []string{"maxFileSizeDefaultMB"}},
		{"no supported languages", func(a *settingsArgs) { a.supported = nil }, []string{"supportedLanguages"}},
		{"a default file size above the zip size", func(a *settingsArgs) { a.defaultMB, a.zipMB = 200, 100 }, []string{"maxFileSizeDefaultMB"}},
		{"a language time limit above the global one", func(a *settingsArgs) { a.globalTime = 1500 }, []string{"languageLimits.cpp20.maxTimeLimit"}},
		{"a language memory limit above the global one", func(a *settingsArgs) { a.globalMemory = 256 }, []string{"languageLimits.cpp20.maxMemoryLimit"}},
		{"several problems are all reported, in order", func(a *settingsArgs) { a.uploadConcurrency, a.sample = 0, 0 }, []string{"uploadMaxConcurrency", "maxFileCountSample"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := validSettingsArgs(t)
			tt.mutate(&args)

			_, err := args.build()

			if got := detailFields(t, err); !reflect.DeepEqual(got, tt.wantFields) {
				t.Errorf("fields = %v, want %v", got, tt.wantFields)
			}
		})
	}
}

func TestNewPlatformSettings_AcceptsTheBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(a *settingsArgs)
	}{
		{"a default file size equal to the zip size", func(a *settingsArgs) { a.defaultMB, a.zipMB = 100, 100 }},
		{"language limits equal to the global ones", func(a *settingsArgs) { a.globalTime, a.globalMemory = 2000, 512 }},
		{"a single supported language", func(a *settingsArgs) { a.supported = map[string]struct{}{"cpp20": {}} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := validSettingsArgs(t)
			tt.mutate(&args)

			if _, err := args.build(); err != nil {
				t.Errorf("NewPlatformSettings: %v", err)
			}
		})
	}
}

func TestPlatformSettings_ExposesEachConfiguredValue(t *testing.T) {
	s, err := validSettingsArgs(t).build()
	if err != nil {
		t.Fatalf("NewPlatformSettings: %v", err)
	}

	maxTime, maxMemory := s.GlobalLimits()
	if maxTime != 5000 || maxMemory != 1024 {
		t.Errorf("GlobalLimits() = %d, %d, want 5000, 1024", maxTime, maxMemory)
	}
	if s.UploadMaxConcurrency() != 4 {
		t.Errorf("UploadMaxConcurrency() = %d, want 4", s.UploadMaxConcurrency())
	}
	if s.MaxFileCountSample() != 3 {
		t.Errorf("MaxFileCountSample() = %d, want 3", s.MaxFileCountSample())
	}
	if s.MaxZipUploadSizeMB() != 100 {
		t.Errorf("MaxZipUploadSizeMB() = %d, want 100", s.MaxZipUploadSizeMB())
	}
	if s.MaxFileCountTestCase() != 500 {
		t.Errorf("MaxFileCountTestCase() = %d, want 500", s.MaxFileCountTestCase())
	}
	if s.MaxFileSizeDefaultMB() != 10 {
		t.Errorf("MaxFileSizeDefaultMB() = %d, want 10", s.MaxFileSizeDefaultMB())
	}
}

func TestPlatformSettings_Lookups(t *testing.T) {
	s, err := validSettingsArgs(t).build()
	if err != nil {
		t.Fatalf("NewPlatformSettings: %v", err)
	}

	for language, want := range map[string]bool{"cpp20": true, "python310": true, "cobol": false} {
		if got := s.IsLanguageSupported(language); got != want {
			t.Errorf("IsLanguageSupported(%q) = %v, want %v", language, got, want)
		}
	}
	for tag, want := range map[string]bool{"dp": true, "graphs": true, "geometry": false} {
		if got := s.IsValidTag(tag); got != want {
			t.Errorf("IsValidTag(%q) = %v, want %v", tag, got, want)
		}
	}
	if lang, ok := s.LanguageByExtension(".cpp"); !ok || lang != "cpp20" {
		t.Errorf("LanguageByExtension(.cpp) = %q, %v, want cpp20, true", lang, ok)
	}
	if lang, ok := s.LanguageByExtension(".rs"); ok || lang != "" {
		t.Errorf("LanguageByExtension(.rs) = %q, %v, want an empty miss", lang, ok)
	}
	limit, ok := s.LanguageLimit("cpp20")
	if !ok || limit.Language() != "cpp20" || limit.MaxTimeLimit() != 2000 || limit.MaxMemoryLimit() != 512 {
		t.Errorf("LanguageLimit(cpp20) = %+v, %v, want cpp20 2000/512", limit, ok)
	}
	if _, ok := s.LanguageLimit("cobol"); ok {
		t.Error("LanguageLimit(cobol) found a limit for a language without one")
	}
}

func TestPlatformSettings_AllowedTags_ReturnsACopy(t *testing.T) {
	s, err := validSettingsArgs(t).build()
	if err != nil {
		t.Fatalf("NewPlatformSettings: %v", err)
	}

	got := s.AllowedTags()
	delete(got, "dp")

	if !s.IsValidTag("dp") {
		t.Error("deleting from the returned map changed the settings")
	}
	if len(s.AllowedTags()) != 2 {
		t.Errorf("AllowedTags() has %d tags, want 2", len(s.AllowedTags()))
	}
}
