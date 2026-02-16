package handlers

import (
	"cms/config"
	"cms/model"
	"cms/storage"
	"cms/utils"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func NewPostForm(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/newpost.html"))
	tmpl.Execute(w, nil)
}

func EditPostForm(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/edit/")
	if slug == "" {
		http.Error(w, "Missing slug", http.StatusBadRequest)
		return
	}

	path := filepath.Join(config.AppConfig.ContentDir, slug+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	post, body := utils.ParseFrontmatter(content)

	tmpl := template.New("editpost.html").Funcs(template.FuncMap{
		"join": strings.Join,
	})
	tmpl = template.Must(tmpl.ParseFiles("templates/editpost.html"))

	tmpl.Execute(w, map[string]interface{}{
		"Post": post,
		"Slug": slug,
		"Body": body,
	})
}

func ListPosts(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(config.AppConfig.ContentDir)
	if err != nil {
		http.Error(w, "Failed to list posts", http.StatusInternalServerError)
		return
	}

	var posts []model.BlogPost

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			fullPath := filepath.Join(config.AppConfig.ContentDir, f.Name())

			post, _, err := storage.ReadMarkdownWithFrontmatter(fullPath)
			if err != nil {
				log.Printf("Failed to read post %s: %v", f.Name(), err)
				continue
			}

			// Add slug (since it's not stored in frontmatter)
			post.Slug = strings.TrimSuffix(f.Name(), ".md")
			posts = append(posts, post)
		}
	}

	// Optional: support tag filtering via query param
	filterTag := r.URL.Query().Get("tag")
	var filtered []model.BlogPost
	if filterTag != "" {
		for _, p := range posts {
			for _, tag := range p.Tags {
				if tag == filterTag {
					filtered = append(filtered, p)
					break
				}
			}
		}
	} else {
		filtered = posts
	}

	// Collect unique tags for filtering UI
	tagSet := map[string]struct{}{}
	for _, p := range posts {
		for _, t := range p.Tags {
			tagSet[t] = struct{}{}
		}
	}
	var allTags []string
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}
	sort.Strings(allTags)

	tmpl := template.Must(template.ParseFiles("templates/listposts.html"))
	tmpl.Execute(w, map[string]any{
		"Posts": filtered,
		"Tags":  allTags,
	})
}
