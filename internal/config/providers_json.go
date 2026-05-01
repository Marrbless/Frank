package config

import (
	"bytes"
	"encoding/json"
	"sort"
)

// UnmarshalJSON preserves the legacy providers.openai field while accepting
// additional named V5 provider entries under the same providers object.
func (p *ProvidersConfig) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*p = ProvidersConfig{}
	if len(raw) == 0 {
		return nil
	}

	p.Named = make(map[string]ProviderConfig)
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		var provider ProviderConfig
		decoder := json.NewDecoder(bytes.NewReader(raw[key]))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&provider); err != nil {
			return err
		}
		if key == LegacyProviderRef {
			provider.defaultType()
			p.OpenAI = &provider
			continue
		}
		provider.defaultType()
		p.Named[key] = provider
	}
	if len(p.Named) == 0 {
		p.Named = nil
	}
	return nil
}

func (p ProvidersConfig) MarshalJSON() ([]byte, error) {
	raw := make(map[string]ProviderConfig)
	if p.OpenAI != nil {
		provider := *p.OpenAI
		provider.defaultType()
		raw[LegacyProviderRef] = provider
	}
	for key, provider := range p.Named {
		provider.defaultType()
		raw[key] = provider
	}
	return json.Marshal(raw)
}

func (p *ProviderConfig) defaultType() {
	if p.Type == "" {
		p.Type = ProviderTypeOpenAICompatible
	}
}
