package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var Config = &Conf{}

type Conf struct {
	LogConfig  *LogConfig  `mapstructure:"log"`
	SyncConfig *SyncConfig `mapstructure:"sync"`
}

type LogConfig struct {
	Level    string `mapstructure:"level"`
	FilePath string `mapstructure:"file_path"`
}

type SyncConfig struct {
	BatchSize int `mapstructure:"batch_size"`
}

func Init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	for _, dir := range configSearchPaths() {
		viper.AddConfigPath(dir)
	}

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("read job config: %w (searched: %v)", err, configSearchPaths()))
	}
	if err := viper.Unmarshal(&Config); err != nil {
		panic(err)
	}
	if Config.SyncConfig == nil {
		Config.SyncConfig = &SyncConfig{}
	}
	if Config.SyncConfig.BatchSize <= 0 {
		Config.SyncConfig.BatchSize = 500
	}
}

func configSearchPaths() []string {
	workDir, _ := os.Getwd()
	exePath, err := os.Executable()
	exeDir := ""
	if err == nil {
		exeDir = filepath.Dir(exePath)
	}

	candidates := []string{
		filepath.Join(workDir, "resources"),
		filepath.Join(workDir, "..", "resources"),        // cwd = job/cmd
		filepath.Join(workDir, "job", "resources"),       // cwd = repo root
		filepath.Join(workDir, "..", "job", "resources"), // cwd = server/ etc.
	}
	if exeDir != "" {
		candidates = append(candidates,
			filepath.Join(exeDir, "resources"),
			filepath.Join(exeDir, "..", "resources"),
		)
	}

	seen := make(map[string]struct{}, len(candidates))
	paths := make([]string, 0, len(candidates))
	for _, p := range candidates {
		abs, absErr := filepath.Abs(p)
		if absErr != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		paths = append(paths, abs)
	}
	return paths
}
