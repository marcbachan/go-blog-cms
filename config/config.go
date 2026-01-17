package config

import (
	"encoding/json"
	"log"
	"os"
)

// ContentTypeConfig defines a single content type
type ContentTypeConfig struct {
	Name      string `json:"name"`      // Display name: "Posts", "Photos"
	Slug      string `json:"slug"`      // URL slug: "posts", "photos"
	Directory string `json:"directory"` // Content dir: "../_posts"
	ImagesDir string `json:"imagesDir"` // Image storage: "../public/assets/img"
	Icon      string `json:"icon"`      // UI icon: "📝", "📷"
}

type Settings struct {
	// Legacy fields for backward compatibility
	PostsDir  string `json:"postsDir"`
	ImagesDir string `json:"imagesDir"`

	// Content types array
	ContentTypes []ContentTypeConfig `json:"contentTypes"`
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

	// Auto-generate content types from legacy config if not defined
	if len(AppConfig.ContentTypes) == 0 && AppConfig.PostsDir != "" {
		AppConfig.ContentTypes = []ContentTypeConfig{
			{
				Name:      "Posts",
				Slug:      "posts",
				Directory: AppConfig.PostsDir,
				ImagesDir: AppConfig.ImagesDir,
				Icon:      "📝",
			},
		}
	}
}

// GetContentType returns config for a content type by slug
func GetContentType(slug string) (*ContentTypeConfig, bool) {
	for i := range AppConfig.ContentTypes {
		if AppConfig.ContentTypes[i].Slug == slug {
			return &AppConfig.ContentTypes[i], true
		}
	}
	return nil, false
}
