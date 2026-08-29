package plan

import "encoding/json"

// planWire mirrors the parts of `terraform show -json` output that price-checker
// uses. Fields it does not need are omitted; unknown JSON keys are ignored.
type planWire struct {
	FormatVersion    string               `json:"format_version"`
	TerraformVersion string               `json:"terraform_version"`
	ResourceChanges  []resourceChangeWire `json:"resource_changes"`
	Configuration    *configWire          `json:"configuration"`
}

type resourceChangeWire struct {
	Address       string     `json:"address"`
	ModuleAddress string     `json:"module_address"`
	Mode          string     `json:"mode"`
	Type          string     `json:"type"`
	Name          string     `json:"name"`
	ProviderName  string     `json:"provider_name"`
	Change        changeWire `json:"change"`
}

type changeWire struct {
	Actions      []string        `json:"actions"`
	Before       json.RawMessage `json:"before"`
	After        json.RawMessage `json:"after"`
	AfterUnknown json.RawMessage `json:"after_unknown"`
}

type configWire struct {
	ProviderConfig map[string]providerConfigWire `json:"provider_config"`
	RootModule     configModuleWire              `json:"root_module"`
}

type providerConfigWire struct {
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	Expressions struct {
		Region *struct {
			ConstantValue any `json:"constant_value"`
		} `json:"region"`
	} `json:"expressions"`
}

type configModuleWire struct {
	Resources []struct {
		Address           string `json:"address"`
		ProviderConfigKey string `json:"provider_config_key"`
	} `json:"resources"`
	ModuleCalls map[string]struct {
		Module configModuleWire `json:"module"`
	} `json:"module_calls"`
}

// decodeObject unmarshals a RawMessage that is expected to be a JSON object into
// a map. A JSON null or absent value yields a nil map with no error.
func decodeObject(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}
