package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type KyperFile struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Tagline     string            `yaml:"tagline,omitempty"`
	Category    string            `yaml:"category"`
	Docker      DockerConfig      `yaml:"docker"`
	Processes   map[string]string `yaml:"processes"`
	Deps        []DepEntry        `yaml:"deps,omitempty"`
	Pricing     PricingConfig     `yaml:"pricing,omitempty"`
	Resources   ResourceConfig    `yaml:"resources,omitempty"`
	Env         []string          `yaml:"env,omitempty"`
	Hooks       HooksConfig       `yaml:"hooks,omitempty"`
	Healthcheck HealthcheckConfig `yaml:"healthcheck,omitempty"`
}

type DockerConfig struct {
	Dockerfile string `yaml:"dockerfile"`
	Image      string `yaml:"image,omitempty"`
}

type PricingConfig struct {
	OneTime      *float64 `yaml:"one_time,omitempty"`
	Subscription *float64 `yaml:"subscription,omitempty"`
}

type ResourceConfig struct {
	MinMemoryMB int `yaml:"min_memory_mb,omitempty"`
	MinCPU      int `yaml:"min_cpu,omitempty"`
}

type HooksConfig struct {
	OnDeploy string `yaml:"on_deploy,omitempty"`
	OnUpdate string `yaml:"on_update,omitempty"`
}

type HealthcheckConfig struct {
	Path     string `yaml:"path,omitempty"`
	Interval int    `yaml:"interval,omitempty"`
	Timeout  int    `yaml:"timeout,omitempty"`
}

// DepEntry represents a dependency with optional version and storage config.
// Supports three YAML formats:
//   - string: "postgres"
//   - colon-pinned: "redis:7"
//   - hash: {postgres: "16", storage_gb: 50}
type DepEntry struct {
	Name      string
	Version   string
	StorageGB int
}

func (d *DepEntry) UnmarshalYAML(value *yaml.Node) error {
	// Format 1 & 2: plain string like "postgres" or "redis:7"
	if value.Kind == yaml.ScalarNode {
		s := value.Value
		if parts := strings.SplitN(s, ":", 2); len(parts) == 2 {
			d.Name = parts[0]
			d.Version = parts[1]
		} else {
			d.Name = s
		}
		return nil
	}

	// Format 3: mapping like {postgres: "16", storage_gb: 50}
	if value.Kind == yaml.MappingNode {
		for i := 0; i < len(value.Content)-1; i += 2 {
			key := value.Content[i].Value
			val := value.Content[i+1]
			switch key {
			case "storage_gb":
				var gb int
				if err := val.Decode(&gb); err != nil {
					return fmt.Errorf("invalid storage_gb: %w", err)
				}
				d.StorageGB = gb
			default:
				// The dep name is the key, version is the value.
				// A null node (bare key with no value) means no version pinned.
				d.Name = key
				if val.Tag != "!!null" {
					d.Version = val.Value
				}
			}
		}
		return nil
	}

	return fmt.Errorf("invalid dep entry format")
}

func (d DepEntry) MarshalYAML() (interface{}, error) {
	if d.StorageGB > 0 {
		// Use empty string (not nil) when no version is set so the round-trip
		// is stable: nil marshals as "null" which UnmarshalYAML cannot reliably
		// distinguish from the string "null".
		m := map[string]interface{}{
			d.Name:       d.Version,
			"storage_gb": d.StorageGB,
		}
		return m, nil
	}
	if d.Version != "" {
		return fmt.Sprintf("%s:%s", d.Name, d.Version), nil
	}
	return d.Name, nil
}

// LoadKyperFile reads and parses a kyper.yml file.
// Returns the parsed struct and the raw bytes.
func LoadKyperFile(path string) (*KyperFile, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var kf KyperFile
	if err := yaml.Unmarshal(data, &kf); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &kf, data, nil
}
