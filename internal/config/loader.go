package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader reads config values from one source (typically a config file).
type Loader interface {
	Load(path string) (LoadedConfig, error)
}

// Validator validates a resolved configuration.
type Validator interface {
	Validate(cfg ResolvedConfig) error
}

// FileLoader implements Loader with TOML config files.
type FileLoader struct{}

// Load satisfies Loader.
func (FileLoader) Load(path string) (LoadedConfig, error) {
	return LoadFile(path)
}

// ConfigValidator implements Validator.
type ConfigValidator struct{}

// Validate satisfies Validator.
func (ConfigValidator) Validate(cfg ResolvedConfig) error {
	return Validate(cfg)
}

// Resolve loads file/env/CLI layers and validates the final result.
func Resolve(configPath string, cli Overlay) (Config, error) {
	return resolve(configPath, cli, os.LookupEnv, os.UserHomeDir, os.Getwd)
}

func resolve(
	configPath string,
	cli Overlay,
	lookupEnv lookupEnvFn,
	homeDirFn func() (string, error),
	getwdFn func() (string, error),
) (Config, error) {
	files, err := loadResolvedConfigFiles(configPath, homeDirFn, getwdFn)
	if err != nil {
		return Config{}, err
	}
	envCfg, err := loadEnv(lookupEnv)
	if err != nil {
		return Config{}, err
	}
	return ResolveWithFileOverlays(Default(), files, envCfg, cli)
}

// ResolveWithOverlays merges layers with fixed precedence and validates:
// CLI > ENV > file > defaults.
func ResolveWithOverlays(defaults Config, file LoadedConfig, env Overlay, cli Overlay) (Config, error) {
	return ResolveWithFileOverlays(defaults, []LoadedConfig{file}, env, cli)
}

// ResolveWithFileOverlays merges multiple file overlays with fixed precedence:
// CLI > ENV > project file > global file > defaults.
func ResolveWithFileOverlays(defaults Config, files []LoadedConfig, env Overlay, cli Overlay) (Config, error) {
	overlays := make([]Overlay, 0, len(files)+2)
	for _, file := range files {
		overlays = append(overlays, file.Overlay)
	}
	overlays = append(overlays, env, cli)
	cfg := Merge(defaults, overlays...)
	if err := Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func loadResolvedConfigFiles(
	configPath string,
	homeDirFn func() (string, error),
	getwdFn func() (string, error),
) ([]LoadedConfig, error) {
	trimmed := strings.TrimSpace(configPath)
	if trimmed != "" {
		loaded, err := LoadFile(trimmed)
		if err != nil {
			return nil, err
		}
		return []LoadedConfig{loaded}, nil
	}

	homeDir, err := homeDirFn()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	cwd, err := getwdFn()
	if err != nil {
		return nil, fmt.Errorf("resolve current directory: %w", err)
	}
	paths := defaultConfigPaths(homeDir, cwd)

	out := make([]LoadedConfig, 0, len(paths))
	for _, p := range paths {
		loaded, ok, err := loadFileIfExists(p)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, loaded)
		}
	}
	return out, nil
}

func defaultConfigPaths(homeDir string, cwd string) []string {
	return []string{
		filepath.Join(homeDir, ".config", "tgup", "config.toml"),
		filepath.Join(cwd, "tgup.toml"),
	}
}

// ResolveTelegramOnly loads config layers but only validates Telegram fields.
// Used by commands (like login) that don't need scan/upload/etc config.
func ResolveTelegramOnly(configPath string, cli Overlay) (Config, error) {
	return resolveTelegramOnly(configPath, cli, os.LookupEnv, os.UserHomeDir, os.Getwd)
}

func resolveTelegramOnly(
	configPath string,
	cli Overlay,
	lookupEnv lookupEnvFn,
	homeDirFn func() (string, error),
	getwdFn func() (string, error),
) (Config, error) {
	files, err := loadResolvedConfigFiles(configPath, homeDirFn, getwdFn)
	if err != nil {
		return Config{}, err
	}
	envCfg, err := loadEnv(lookupEnv)
	if err != nil {
		return Config{}, err
	}
	overlays := make([]Overlay, 0, len(files)+2)
	for _, file := range files {
		overlays = append(overlays, file.Overlay)
	}
	overlays = append(overlays, envCfg, cli)
	cfg := Merge(Default(), overlays...)
	// Only validate Telegram fields.
	if err := ValidateTelegram(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func loadFileIfExists(path string) (LoadedConfig, bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LoadedConfig{}, false, nil
		}
		return LoadedConfig{}, false, fmt.Errorf("stat config file: %w", err)
	}
	loaded, err := LoadFile(path)
	if err != nil {
		return LoadedConfig{}, false, err
	}
	return loaded, true, nil
}
