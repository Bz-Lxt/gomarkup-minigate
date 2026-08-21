package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"minigate/internal/model"
)

// Source is the V2-ready config abstraction. MVP implements FileSource only.
type Source interface {
	Name() string
	Load() (*model.GatewayConfig, error)
	Save(*model.GatewayConfig) error
}

type FileSource struct {
	Path string
}

func (f *FileSource) Name() string { return "file" }

func (f *FileSource) Load() (*model.GatewayConfig, error) {
	raw, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", f.Path, err)
	}
	var cfg model.GatewayConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", f.Path, err)
	}
	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (f *FileSource) Save(cfg *model.GatewayConfig) error {
	if err := Validate(cfg); err != nil {
		return err
	}
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, f.Path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func Hash(cfg *model.GatewayConfig) string {
	raw, _ := yaml.Marshal(cfg)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func Clone(cfg *model.GatewayConfig) *model.GatewayConfig {
	raw, _ := yaml.Marshal(cfg)
	var out model.GatewayConfig
	_ = yaml.Unmarshal(raw, &out)
	return &out
}
