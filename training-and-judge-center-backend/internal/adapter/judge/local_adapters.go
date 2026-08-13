package judge

import infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"

func NewSourceCodeDownloaderLocal(dir string) *SourceCodeDownloader {
	return &SourceCodeDownloader{reader: newLocalReader(dir)}
}

func NewTestCaseProviderLocal(dir string, db infraPostgres.Querier) *TestCaseProvider {
	return &TestCaseProvider{reader: newLocalReader(dir), db: db}
}

func NewOutputComparatorLocal(dir string) *OutputComparator {
	return &OutputComparator{reader: newLocalReader(dir)}
}

func NewArtifactUploaderLocal(dir string) *ArtifactUploader {
	return &ArtifactUploader{writer: newLocalWriter(dir)}
}
