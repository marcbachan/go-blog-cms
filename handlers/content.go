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

// Dashboard shows overview of all content types
func Dashboard(w http.ResponseWriter, r *http.Request) {
	allTypes := config.AppConfig.ContentTypes

	// Get counts for each type
	typeCounts := make(map[string]int)
	for _, ct := range allTypes {
		files, err := os.ReadDir(ct.Directory)
		if err == nil {
			count := 0
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
					count++
				}
			}
			typeCounts[ct.Slug] = count
		}
	}

	tmpl := template.Must(template.ParseFiles("templates/dashboard.html"))
	tmpl.Execute(w, map[string]any{
		"ContentTypes": allTypes,
		"Counts":       typeCounts,
	})
}

// ListContent handles /{type} - lists all items of a content type
func ListContent(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	ct, ok := config.GetContentType(typeSlug)
	if !ok {
		http.Error(w, "Unknown content type", http.StatusNotFound)
		return
	}

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
			items = append(items, item)
		}
	}

	// Sort by date descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].Date > items[j].Date
	})

	// Tag filtering
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

	// Collect all tags
	tagSet := map[string]struct{}{}
	for _, item := range items {
		for _, t := range item.Tags {
			tagSet[t] = struct{}{}
		}
	}
	var allTags []string
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}
	sort.Strings(allTags)

	// Get all content types for navigation
	allTypes := config.AppConfig.ContentTypes

	tmpl := template.Must(template.ParseFiles("templates/listcontent.html"))
	tmpl.Execute(w, map[string]any{
		"Items":       filtered,
		"Tags":        allTags,
		"ContentType": ct,
		"AllTypes":    allTypes,
	})
}

// GetPreview returns HTML preview of a content item (for HTMX expandable preview)
func GetPreview(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	slug := mux.Vars(r)["slug"]

	ct, ok := config.GetContentType(typeSlug)
	if !ok {
		http.Error(w, "Unknown content type", http.StatusNotFound)
		return
	}

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
	ct, ok := config.GetContentType(typeSlug)
	if !ok {
		http.Error(w, "Unknown content type", http.StatusNotFound)
		return
	}

	allTypes := config.AppConfig.ContentTypes

	tmpl := template.Must(template.ParseFiles("templates/newcontent.html"))
	tmpl.Execute(w, map[string]any{
		"ContentType": ct,
		"AllTypes":    allTypes,
	})
}

// EditContentForm handles GET /{type}/edit/{slug}
func EditContentForm(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	slug := mux.Vars(r)["slug"]

	ct, ok := config.GetContentType(typeSlug)
	if !ok {
		http.Error(w, "Unknown content type", http.StatusNotFound)
		return
	}

	path := filepath.Join(ct.Directory, slug+".md")
	item, body, err := storage.ReadContent(path)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	allTypes := config.AppConfig.ContentTypes

	tmpl := template.New("editcontent.html").Funcs(template.FuncMap{
		"join": strings.Join,
	})
	tmpl = template.Must(tmpl.ParseFiles("templates/editcontent.html"))

	tmpl.Execute(w, map[string]interface{}{
		"Item":        item,
		"Slug":        slug,
		"Body":        body,
		"ContentType": ct,
		"AllTypes":    allTypes,
	})
}

// CreateContent handles POST /api/{type}
func CreateContent(w http.ResponseWriter, r *http.Request) {
	typeSlug := mux.Vars(r)["type"]
	ct, ok := config.GetContentType(typeSlug)
	if !ok {
		http.Error(w, "Unknown content type", http.StatusNotFound)
		return
	}

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

	ct, ok := config.GetContentType(typeSlug)
	if !ok {
		http.Error(w, "Unknown content type", http.StatusNotFound)
		return
	}

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

	ct, ok := config.GetContentType(typeSlug)
	if !ok {
		http.Error(w, "Unknown content type", http.StatusNotFound)
		return
	}

	contentPath := filepath.Join(ct.Directory, slug+".md")
	imgPath := filepath.Join(ct.ImagesDir, slug)

	if err := os.Remove(contentPath); err != nil {
		http.Error(w, "Failed to delete content", http.StatusInternalServerError)
		return
	}

	os.RemoveAll(imgPath) // Best effort for images

	w.WriteHeader(http.StatusOK)
}
