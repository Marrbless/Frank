package modelsetup

import "github.com/local/picobot/internal/config"

type State string

const (
	StatePresent     State = "present"
	StateMissing     State = "missing"
	StateUnknown     State = "unknown"
	StateUnsupported State = "unsupported"
	StateAmbiguous   State = "ambiguous"
)

type RuntimeKind string

const (
	RuntimeKindOllama   RuntimeKind = "ollama"
	RuntimeKindLlamaCPP RuntimeKind = "llamacpp"
	RuntimeKindCloud    RuntimeKind = "cloud"
)

type PlanStatus string

const (
	PlanStatusPlanned        PlanStatus = "planned"
	PlanStatusSkipped        PlanStatus = "skipped"
	PlanStatusAlreadyPresent PlanStatus = "already_present"
	PlanStatusManualRequired PlanStatus = "manual_required"
	PlanStatusChanged        PlanStatus = "changed"
	PlanStatusFailed         PlanStatus = "failed"
	PlanStatusRolledBack     PlanStatus = "rolled_back"
	PlanStatusBlocked        PlanStatus = "blocked"
)

type SideEffectKind string

const (
	SideEffectNone            SideEffectKind = "none"
	SideEffectReadFile        SideEffectKind = "read_file"
	SideEffectWriteConfig     SideEffectKind = "write_config"
	SideEffectWriteBootScript SideEffectKind = "write_boot_script"
	SideEffectRunCommand      SideEffectKind = "run_command"
	SideEffectDownload        SideEffectKind = "download"
	SideEffectInstallRuntime  SideEffectKind = "install_runtime"
	SideEffectPullModel       SideEffectKind = "pull_model"
	SideEffectStartRuntime    SideEffectKind = "start_runtime"
	SideEffectHealthCheck     SideEffectKind = "health_check"
	SideEffectRouteCheck      SideEffectKind = "route_check"
)

type Preset struct {
	Name                  string
	DisplayName           string
	DefaultSafe           bool
	ExplicitlyGated       bool
	RuntimeKind           RuntimeKind
	ProviderRef           string
	ModelRef              string
	ProviderModel         string
	ModelSource           string
	ManifestID            string
	ExpectedDiskImpact    string
	ExpectedRAMRange      string
	Capabilities          config.ModelCapabilities
	Request               config.ModelRequestConfig
	BaseURL               string
	HealthURL             string
	BindAddress           string
	Port                  int
	BootSupported         bool
	DownloadsRequired     bool
	CloudKeysRequired     bool
	CloudFallbackDefault  bool
	SupportsRegisterLocal bool
	SafetyNotes           []string
}

type EnvSnapshot struct {
	Platform              string
	OS                    string
	Arch                  string
	ConfigPath            string
	Termux                State
	TermuxBoot            State
	Tmux                  State
	Ollama                State
	LlamaCPP              State
	ExistingProviders     []string
	ExistingModels        []string
	ExistingAliases       []string
	ExistingLocalRuntimes []string
	ExistingBootScripts   []string
	UnsafeStates          []string
}

type OperatorChoices struct {
	PresetName               string
	ConfigPath               string
	RuntimeKind              RuntimeKind
	ModelRef                 string
	ProviderRef              string
	InstallBehavior          string
	DownloadBehavior         string
	RegisterExistingBehavior string
	BindAddress              string
	Port                     int
	BootScripts              bool
	ApproveLANBind           bool
	AllowCloudFallback       bool
	Force                    bool
	NonInteractive           bool
	Approve                  bool
	DryRun                   bool
}

type Plan struct {
	PresetName          string
	Status              PlanStatus
	Environment         EnvSnapshot
	Assumptions         []string
	Warnings            []string
	BlockedReasons      []string
	ManualInstructions  []string
	ProviderRef         string
	ModelRef            string
	ProviderModel       string
	RuntimeKind         RuntimeKind
	BindAddress         string
	Port                int
	CloudFallback       bool
	ToolSupport         bool
	AuthorityTier       config.ModelAuthorityTier
	ConfigPatch         *ConfigPatch
	Steps               []PlanStep
	RedactionPolicy     string
	TruncationPolicy    string
	GeneratedReportHint string
}

type ConfigPatch struct {
	ProviderRef     string
	ModelRef        string
	AliasRefs       map[string]string
	RuntimeRef      string
	DefaultModelRef string
	ModelConfig     config.ModelProfileConfig
	ProviderConfig  config.ProviderConfig
	RoutingConfig   config.ModelRoutingConfig
	RuntimeConfig   config.LocalRuntimeConfig
}

type PlanStep struct {
	ID                         string
	Summary                    string
	SideEffect                 SideEffectKind
	Command                    []string
	FilesToRead                []string
	FilesToWrite               []string
	NetworkURL                 string
	ExpectedDownloadSize       string
	ExpectedDiskImpact         string
	RuntimePort                int
	RuntimeBindAddress         string
	ApprovalRequired           bool
	ApprovalReason             string
	IdempotencyKey             string
	AlreadyPresentRule         string
	RollbackCleanup            string
	RedactionPolicy            string
	Dependencies               []string
	Status                     PlanStatus
	ManualInstructions         []string
	DiagnosticsPriority        int
	SafeToOmitWhenTruncating   bool
	PreserveWhenTruncating     bool
	RequiresManifest           bool
	RequiresExplicitLANApprove bool
}
