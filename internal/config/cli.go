package config

// CLIFlags holds the raw command-line flag values before environment-variable resolution.
type CLIFlags struct {
	DocID           string
	CredentialsPath string
	DryRun          *bool
	ChunkSize       int
	PageRefresh     *bool
	OutputDir       string
	Model           string
	SummaryModel    string
	TargetRepo      string
	ArtifactsDir    string
	BranchPrefix    string
	OpenPR          *bool
	OpenIssue       *bool
}
