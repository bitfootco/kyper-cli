package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type KyperFile struct {
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	Tagline      string            `yaml:"tagline,omitempty"`
	Category     string            `yaml:"category"`
	Docker       DockerConfig      `yaml:"docker"`
	Processes    map[string]string `yaml:"processes"`
	Deps         []DepEntry        `yaml:"deps,omitempty"`
	Pricing      PricingConfig     `yaml:"pricing,omitempty"`
	Resources    ResourceConfig    `yaml:"resources,omitempty"`
	Env          []string          `yaml:"env,omitempty"`
	Integrations []string          `yaml:"integrations,omitempty"`
	Storage      StorageConfig     `yaml:"storage,omitempty"`
	Hooks        HooksConfig       `yaml:"hooks,omitempty"`
	Healthcheck  HealthcheckConfig `yaml:"healthcheck,omitempty"`
	Security     SecurityConfig    `yaml:"security,omitempty"`
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
	MinMemoryMB  int     `yaml:"min_memory_mb,omitempty"`
	MinCPU       float64 `yaml:"min_cpu,omitempty"`
	minMemorySet bool
	minCPUSet    bool
}

func (r *ResourceConfig) UnmarshalYAML(value *yaml.Node) error {
	type rawResourceConfig struct {
		MinMemoryMB int     `yaml:"min_memory_mb,omitempty"`
		MinCPU      float64 `yaml:"min_cpu,omitempty"`
	}
	var raw rawResourceConfig
	if err := value.Decode(&raw); err != nil {
		return err
	}

	r.MinMemoryMB = raw.MinMemoryMB
	r.MinCPU = raw.MinCPU
	for i := 0; i < len(value.Content)-1; i += 2 {
		switch value.Content[i].Value {
		case "min_memory_mb":
			r.minMemorySet = true
		case "min_cpu":
			r.minCPUSet = true
		}
	}
	return nil
}

func (r ResourceConfig) HasMinMemoryMB() bool {
	return r.minMemorySet || r.MinMemoryMB != 0
}

func (r ResourceConfig) HasMinCPU() bool {
	return r.minCPUSet || r.MinCPU != 0
}

type StorageConfig struct {
	Mounts []StorageMount `yaml:"mounts,omitempty"`
}

type StorageMount struct {
	Path      string `yaml:"path"`
	StorageGB *int   `yaml:"storage_gb,omitempty"`
}

type HooksConfig struct {
	// Release runs as a one-shot K8s Job after deps (postgres, redis, etc.)
	// are ready and BEFORE app pods are created. Use this for migrations and
	// any other setup that must complete before the web process starts.
	Release string `yaml:"release,omitempty"`
	// OnDeploy runs INSIDE a healthy web pod after pods are ready, on a fresh
	// deployment. Use for cache warmup, post-deploy notifications, etc. Do
	// NOT use for migrations — see Release.
	OnDeploy string `yaml:"on_deploy,omitempty"`
	// OnUpdate runs INSIDE a healthy web pod after pods are ready, on a
	// version update. Same execution model as OnDeploy.
	OnUpdate string `yaml:"on_update,omitempty"`
	// Unknown captures any hook keys NOT in the known set (release, on_deploy,
	// on_update). The validator surfaces these as errors so a typo like
	// `hooks.releaes` doesn't silently no-op. Populated via yaml inline tag.
	Unknown map[string]interface{} `yaml:",inline"`
}

type HealthcheckConfig struct {
	Path         string `yaml:"path,omitempty"`
	InitialDelay int    `yaml:"initial_delay,omitempty"`
	Interval     int    `yaml:"interval,omitempty"`
	Timeout      int    `yaml:"timeout,omitempty"`
}

type SecurityConfig struct {
	IgnoreCVEs []IgnoreCVE `yaml:"ignore_cves,omitempty"`
}

type IgnoreCVE struct {
	ID     string `yaml:"id"`
	Reason string `yaml:"reason"`
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
