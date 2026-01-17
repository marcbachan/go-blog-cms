package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cms/config"
	"cms/model"
	"cms/storage"

	"github.com/google/uuid"

	"github.com/gorilla/mux"
)

func clearTempFolder(path string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !file.IsDir() {
			os.Remove(filepath.Join(path, file.Name()))
		}
	}
	return nil
}

func CreatePost(w http.ResponseWriter, r *http.Request) {
	var post model.BlogPost
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("%s.md", post.Slug)
	fullPath := filepath.Join(config.AppConfig.PostsDir, filename)

	// Assume: post.CoverImage was "/tmp-preview/scottishcow.jpg"

	tmpPath := strings.TrimPrefix(post.CoverImage, "/")             // => "tmp-preview/scottishcow.jpg"
	src := filepath.Join("public", tmpPath)                         // => "public/tmp-preview/scottishcow.jpg"
	destDir := filepath.Join(config.AppConfig.ImagesDir, post.Slug) // => e.g. "public/assets/img/cow"
	os.MkdirAll(destDir, os.ModePerm)

	destFilename := filepath.Base(tmpPath)       // => "scottishcow.jpg"
	dest := filepath.Join(destDir, destFilename) // => "public/assets/img/cow/scottishcow.jpg"

	err := os.Rename(src, dest)
	if err != nil {
		log.Printf("Failed to move image: %v", err)
		http.Error(w, "Failed to move image", http.StatusInternalServerError)
		return
	}

	// Update the image paths to their final public-facing URLs
	finalPath := fmt.Sprintf("/assets/img/%s/%s", post.Slug, destFilename)
	post.CoverImage = finalPath
	post.OGImage.URL = finalPath

	// Now write the frontmatter with the correct path
	if err := storage.WriteMarkdownWithFrontmatter(fullPath, post); err != nil {
		http.Error(w, "Failed to write post", http.StatusInternalServerError)
		return
	}

	// Clear temp preview folder
	if err := clearTempFolder("./public/tmp-preview"); err != nil {
		log.Printf("Warning: failed to clear tmp-preview folder: %v", err)
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "🎉 Post written to %s\n", fullPath)
}

func UploadImage(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image not provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate a unique filename
	ext := filepath.Ext(header.Filename)
	tempName := uuid.New().String() + ext
	tmpDir := "./public/tmp-preview"
	os.MkdirAll(tmpDir, os.ModePerm)

	tmpPath := filepath.Join(tmpDir, tempName)
	out, err := os.Create(tmpPath)
	if err != nil {
		http.Error(w, "Failed to save temporary image", http.StatusInternalServerError)
		return
	}
	defer out.Close()
	io.Copy(out, file)

	// Public URL served by Next.js from /public
	webPath := fmt.Sprintf("/tmp-preview/%s", tempName)

	log.Printf("Uploaded temp image: %s (served as %s)", tmpPath, webPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": webPath,
	})
}

func UpdatePost(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	if slug == "" {
		http.Error(w, "Missing slug", http.StatusBadRequest)
		return
	}

	var post model.BlogPost
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	path := filepath.Join(config.AppConfig.PostsDir, slug+".md")
	if err := storage.WriteMarkdownWithFrontmatter(path, post); err != nil {
		http.Error(w, "Failed to update post", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	currentTime := time.Now()
	fmt.Fprintf(w, "✅ Post updated at: %s\n", currentTime.Format(time.RFC3339))
}

func DeletePost(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	if slug == "" {
		http.Error(w, "Missing slug", http.StatusBadRequest)
		return
	}

	postPath := filepath.Join(config.AppConfig.PostsDir, slug+".md")
	imgPath := filepath.Join(config.AppConfig.ImagesDir, slug)

	err := os.Remove(postPath)
	if err != nil {
		http.Error(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}
	err = os.RemoveAll(imgPath)
	if err != nil {
		log.Printf("Warning: failed to delete images for post %s: %v", slug, err)
	}

	w.WriteHeader(http.StatusOK)
}
