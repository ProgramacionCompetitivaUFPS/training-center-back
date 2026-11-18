package judge

import (
	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
)

func NewSourceCodeDownloaderLocal(dir string) *SourceCodeDownloader {
	return &SourceCodeDownloader{reader: newLocalReader(dir)}
}

func NewTestCaseProviderLocal(dir string, db infraPostgres.Querier, cfg TestDataConfig) (*TestCaseProvider, error) {
	cache, err := newTestDataCache(cfg, newSizedLocalReader(dir))
	if err != nil {
		return nil, err
	}
	return &TestCaseProvider{cache: cache, db: db}, nil
}

func NewOutputCheckerLocal(p *pool.Pool, docker dockerExecClient, cfg ArtifactConfig, dir string, judgingRoot string) *OutputChecker {
	return &OutputChecker{pool: p, docker: docker, reader: newLocalReader(dir), cfg: cfg, judgingRoot: judgingRoot}
}

func NewArtifactUploaderLocal(dir string) *ArtifactUploader {
	return &ArtifactUploader{writer: newLocalWriter(dir)}
}

func NewValidatorRunnerLocal(p *pool.Pool, docker dockerExecClient, cfg ArtifactConfig, dir string) *ValidatorRunner {
	return &ValidatorRunner{pool: p, docker: docker, reader: newLocalReader(dir), cfg: cfg}
}
