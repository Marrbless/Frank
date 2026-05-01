package config

// Config holds picobot configuration (minimal for v0).
type Config struct {
	Agents        AgentsConfig                  `json:"agents"`
	MCPServers    map[string]MCPServerConfig    `json:"mcpServers"`
	Channels      ChannelsConfig                `json:"channels"`
	Providers     ProvidersConfig               `json:"providers"`
	Models        map[string]ModelProfileConfig `json:"models,omitempty"`
	ModelAliases  map[string]string             `json:"modelAliases,omitempty"`
	ModelRouting  ModelRoutingConfig            `json:"modelRouting,omitempty"`
	LocalRuntimes map[string]LocalRuntimeConfig `json:"localRuntimes,omitempty"`
}

// MCPServerConfig describes a single MCP server connection.
// Use Command+Args for stdio transport, or URL+Headers for HTTP transport.
type MCPServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
}

type AgentDefaults struct {
	Workspace                   string  `json:"workspace"`
	Model                       string  `json:"model"`
	MaxTokens                   int     `json:"maxTokens"`
	Temperature                 float64 `json:"temperature"`
	MaxToolIterations           int     `json:"maxToolIterations"`
	HeartbeatIntervalS          int     `json:"heartbeatIntervalS"`
	RequestTimeoutS             int     `json:"requestTimeoutS"`
	EnableToolActivityIndicator *bool   `json:"enableToolActivityIndicator,omitempty"`
}

type ChannelsConfig struct {
	Telegram TelegramConfig `json:"telegram"`
	Discord  DiscordConfig  `json:"discord"`
	Slack    SlackConfig    `json:"slack"`
	WhatsApp WhatsAppConfig `json:"whatsapp"`
}

type DiscordConfig struct {
	Enabled   bool     `json:"enabled"`
	Token     string   `json:"token"`
	AllowFrom []string `json:"allowFrom"`
	OpenMode  bool     `json:"openMode,omitempty"`
}

type TelegramConfig struct {
	Enabled   bool     `json:"enabled"`
	Token     string   `json:"token"`
	AllowFrom []string `json:"allowFrom"`
	OpenMode  bool     `json:"openMode,omitempty"`
}

type SlackConfig struct {
	Enabled         bool     `json:"enabled"`
	AppToken        string   `json:"appToken"`
	BotToken        string   `json:"botToken"`
	AllowUsers      []string `json:"allowUsers"`
	AllowChannels   []string `json:"allowChannels"`
	OpenUserMode    bool     `json:"openUserMode,omitempty"`
	OpenChannelMode bool     `json:"openChannelMode,omitempty"`
}

type WhatsAppConfig struct {
	Enabled   bool     `json:"enabled"`
	DBPath    string   `json:"dbPath"`
	AllowFrom []string `json:"allowFrom"`
	OpenMode  bool     `json:"openMode,omitempty"`
}

type ProvidersConfig struct {
	OpenAI *ProviderConfig           `json:"openai,omitempty"`
	Named  map[string]ProviderConfig `json:"-"`
}

type ProviderConfig struct {
	Type            string `json:"type,omitempty"`
	APIKey          string `json:"apiKey"`
	APIBase         string `json:"apiBase"`
	UseResponses    bool   `json:"useResponses,omitempty"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
}

type ModelProfileConfig struct {
	Provider      string             `json:"provider"`
	ProviderModel string             `json:"providerModel"`
	DisplayName   string             `json:"displayName,omitempty"`
	Capabilities  ModelCapabilities  `json:"capabilities,omitempty"`
	Request       ModelRequestConfig `json:"request,omitempty"`
}

type ModelCapabilities struct {
	Local                bool               `json:"local"`
	Offline              bool               `json:"offline"`
	SupportsTools        bool               `json:"supportsTools"`
	SupportsStreaming    bool               `json:"supportsStreaming"`
	SupportsResponsesAPI bool               `json:"supportsResponsesAPI"`
	SupportsVision       bool               `json:"supportsVision"`
	SupportsAudio        bool               `json:"supportsAudio"`
	ContextTokens        int                `json:"contextTokens,omitempty"`
	MaxOutputTokens      int                `json:"maxOutputTokens,omitempty"`
	AuthorityTier        ModelAuthorityTier `json:"authorityTier,omitempty"`
	CostTier             ModelCostTier      `json:"costTier,omitempty"`
	LatencyTier          ModelLatencyTier   `json:"latencyTier,omitempty"`
}

type ModelRequestConfig struct {
	MaxTokens       int      `json:"maxTokens,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	TimeoutS        int      `json:"timeoutS,omitempty"`
	UseResponses    *bool    `json:"useResponses,omitempty"`
	ReasoningEffort string   `json:"reasoningEffort,omitempty"`
}

type ModelRoutingConfig struct {
	DefaultModel                string              `json:"defaultModel,omitempty"`
	LocalPreferredModel         string              `json:"localPreferredModel,omitempty"`
	Fallbacks                   map[string][]string `json:"fallbacks,omitempty"`
	AllowCloudFallbackFromLocal bool                `json:"allowCloudFallbackFromLocal,omitempty"`
	AllowLowerAuthorityFallback bool                `json:"allowLowerAuthorityFallback,omitempty"`
}

type LocalRuntimeConfig struct {
	Kind            string `json:"kind,omitempty"`
	Provider        string `json:"provider,omitempty"`
	ExpectedBaseURL string `json:"expectedBaseURL,omitempty"`
	StartCommand    string `json:"startCommand,omitempty"`
	HealthURL       string `json:"healthURL,omitempty"`
	Notes           string `json:"notes,omitempty"`
}
