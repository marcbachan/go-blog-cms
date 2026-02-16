package handlers

import (
	"cms/config"
	"cms/model"
	"cms/storage"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// hasTag checks if a tag list contains a specific tag
func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// discoverTags scans all content files and returns unique tags with counts
func discoverTags() map[string]int {
	tagCounts := make(map[string]int)
	dir := config.AppConfig.ContentDir

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Failed to read content dir %s: %v", dir, err)
		return tagCounts
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		fullPath := filepath.Join(dir, f.Name())
		item, _, err := storage.ReadContent(fullPath)
		if err != nil {
			log.Printf("Failed to read %s: %v", f.Name(), err)
			continue
		}
		for _, tag := range item.Tags {
			tagCounts[tag]++
		}
	}

	return tagCounts
}

// Dashboard shows overview of all content types discovered from tags
func Dashboard(w http.ResponseWriter, r *http.Request) {
	tagCounts := discoverTags()

	// Build sorted list of content types
	var tags []string
	for tag := range tagCounts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	var contentTypes []config.ContentTypeConfig
	typeCounts := make(map[string]int)
	for _, tag := range tags {
		ct := config.BuildContentType(tag)
		contentTypes = append(contentTypes, ct)
		typeCounts[ct.Slug] = tagCounts[tag]
	}

	tmpl := template.Must(template.ParseFiles("templates/dashboard.html"))
	tmpl.Execute(w, map[string]any{
		"ContentTypes": contentTypes,
		"Counts":       typeCounts,
	})
}

// ListContent handles /{type} - lists all items of a content type
func ListContent(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	ct := config.BuildContentType(typeSlug)

	files, err := os.ReadDir(ct.Directory)
	if err != nil {
		http.Error(w, "Failed to list content", http.StatusInternalServerError)
		return
	}

	var items []model.Content
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			fullPath := filepath.Join(ct.Directory, f.Name())

			item, _, err := storage.ReadContent(fullPath)
			if err != nil {
				log.Printf("Failed to read %s: %v", f.Name(), err)
				continue
			}

			item.Slug = strings.TrimSuffix(f.Name(), ".md")
			item.TypeSlug = typeSlug

			// Filter by content type's tag
			if !hasTag(item.Tags, ct.FilterTag) {
				continue
			}

			items = append(items, item)
		}
	}

	// Sort by date descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].Date > items[j].Date
	})

	// User tag filtering (on top of auto-filter)
	filterTag := r.URL.Query().Get("tag")
	var filtered []model.Content
	if filterTag != "" {
		for _, item := range items {
			for _, tag := range item.Tags {
				if tag == filterTag {
					filtered = append(filtered, item)
					break
				}
			}
		}
	} else {
		filtered = items
	}

	// Collect all tags, excluding the content type's own FilterTag
	tagSet := map[string]struct{}{}
	for _, item := range items {
		for _, t := range item.Tags {
			if t != ct.FilterTag {
				tagSet[t] = struct{}{}
			}
		}
	}
	var allTags []string
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}
	sort.Strings(allTags)

	tmpl := template.Must(template.ParseFiles("templates/listcontent.html"))
	tmpl.Execute(w, map[string]any{
		"Items":       filtered,
		"Tags":        allTags,
		"ContentType": ct,
	})
}

// GetPreview returns HTML preview of a content item (for HTMX expandable preview)
func GetPreview(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	slug := mux.Vars(r)["slug"]

	ct := config.BuildContentType(typeSlug)

	path := filepath.Join(ct.Directory, slug+".md")
	item, body, err := storage.ReadContent(path)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	item.Content = body
	item.Slug = slug
	item.TypeSlug = typeSlug

	tmpl := template.Must(template.ParseFiles("templates/partials/preview.html"))
	tmpl.Execute(w, map[string]any{
		"Item":        item,
		"ContentType": ct,
	})
}

// NewContentForm handles GET /{type}/new
func NewContentForm(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	ct := config.BuildContentType(typeSlug)

	tmpl := template.Must(template.ParseFiles("templates/newcontent.html"))
	tmpl.Execute(w, map[string]any{
		"ContentType": ct,
		"FilterTag":   ct.FilterTag,
	})
}

// EditContentForm handles GET /{type}/edit/{slug}
func EditContentForm(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	slug := mux.Vars(r)["slug"]

	ct := config.BuildContentType(typeSlug)

	path := filepath.Join(ct.Directory, slug+".md")
	item, body, err := storage.ReadContent(path)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	tmpl := template.New("editcontent.html").Funcs(template.FuncMap{
		"join": strings.Join,
	})
	tmpl = template.Must(tmpl.ParseFiles("templates/editcontent.html"))

	tmpl.Execute(w, map[string]interface{}{
		"Item":        item,
		"Slug":        slug,
		"Body":        body,
		"ContentType": ct,
	})
}

// CreateContent handles POST /api/{type}
func CreateContent(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	ct := config.BuildContentType(typeSlug)

	var item model.Content
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("%s.md", item.Slug)
	fullPath := filepath.Join(ct.Directory, filename)

	// Handle image moving from temp folder
	if strings.HasPrefix(item.CoverImage, "/tmp-preview/") {
		tmpPath := strings.TrimPrefix(item.CoverImage, "/")
		src := filepath.Join("public", tmpPath)
		destDir := filepath.Join(ct.ImagesDir, item.Slug)
		os.MkdirAll(destDir, os.ModePerm)

		destFilename := filepath.Base(tmpPath)
		dest := filepath.Join(destDir, destFilename)

		if err := os.Rename(src, dest); err != nil {
			log.Printf("Failed to move image: %v", err)
			http.Error(w, "Failed to move image", http.StatusInternalServerError)
			return
		}

		// Determine the public path based on content type
		publicPath := strings.TrimPrefix(ct.ImagesDir, "../public")
		finalPath := fmt.Sprintf("%s/%s/%s", publicPath, item.Slug, destFilename)
		item.CoverImage = finalPath
		item.OGImage.URL = finalPath
	}

	if err := storage.WriteContent(fullPath, item); err != nil {
		http.Error(w, "Failed to write content", http.StatusInternalServerError)
		return
	}

	clearTempFolder("./public/tmp-preview")

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Content created: %s\n", fullPath)
}

// UpdateContent handles PUT /api/{type}/{slug}
func UpdateContent(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	slug := mux.Vars(r)["slug"]

	ct := config.BuildContentType(typeSlug)

	var item model.Content
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	path := filepath.Join(ct.Directory, slug+".md")
	if err := storage.WriteContent(path, item); err != nil {
		http.Error(w, "Failed to update content", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Updated at: %s\n", time.Now().Format(time.RFC3339))
}

// DeleteContent handles DELETE /api/{type}/{slug}
func DeleteContent(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	slug := mux.Vars(r)["slug"]

	ct := config.BuildContentType(typeSlug)

	contentPath := filepath.Join(ct.Directory, slug+".md")
	imgPath := filepath.Join(ct.ImagesDir, slug)

	if err := os.Remove(contentPath); err != nil {
		http.Error(w, "Failed to delete content", http.StatusInternalServerError)
		return
	}

	os.RemoveAll(imgPath) // Best effort for images

	w.WriteHeader(http.StatusOK)
}
