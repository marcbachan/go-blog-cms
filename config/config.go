package config

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

// TagOverride provides optional display overrides for a tag category
type TagOverride struct {
	Name      string `json:"name,omitempty"`
	Icon      string `json:"icon,omitempty"`
	ImagesDir string `json:"imagesDir,omitempty"`
}

// ContentTypeConfig is a runtime display struct used by templates
type ContentTypeConfig struct {
	Name      string
	Slug      string
	Directory string
	ImagesDir string
	Icon      string
	FilterTag string
}

type Settings struct {
	ContentDir string                 `json:"contentDir"`
	ImagesDir  string                 `json:"imagesDir"`
	TagConfig  map[string]TagOverride `json:"tagConfig"`
}

var AppConfig Settings

func LoadConfig(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&AppConfig); err != nil {
		log.Fatalf("Failed to decode config JSON: %v", err)
	}

	if AppConfig.TagConfig == nil {
		AppConfig.TagConfig = make(map[string]TagOverride)
	}
}

// BuildContentType constructs a ContentTypeConfig for a given tag
func BuildContentType(tag string) ContentTypeConfig {
	ct := ContentTypeConfig{
		Name:      strings.ToUpper(tag[:1]) + tag[1:],
		Slug:      tag,
		Directory: AppConfig.ContentDir,
		ImagesDir: AppConfig.ImagesDir,
		Icon:      "📁",
		FilterTag: tag,
	}

	if override, ok := AppConfig.TagConfig[tag]; ok {
		if override.Name != "" {
			ct.Name = override.Name
		}
		if override.Icon != "" {
			ct.Icon = override.Icon
		}
		if override.ImagesDir != "" {
			ct.ImagesDir = override.ImagesDir
		}
	}

	return ct
}
