package storage

import (
	"fmt"
	"os"
	"strings"
	"time"

	"cms/model"
)

func WriteMarkdownWithFrontmatter(path string, post model.BlogPost) error {
	// fallback to current date if not supplied
	date := post.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	frontmatter := fmt.Sprintf(`---
title: "%s"
excerpt: "%s"
coverImage: "%s"
date: "%s"
ogImage:
  url: "%s"
tags: [%s]
---

`, post.Title, post.Excerpt, post.CoverImage, date, post.OGImage.URL, strings.Join(post.Tags, ", "))

	fullContent := frontmatter + post.Content
	return os.WriteFile(path, []byte(fullContent), 0644)
}

// WriteContent writes any content type to markdown with frontmatter
func WriteContent(path string, content model.Content) error {
	date := content.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	frontmatter := fmt.Sprintf(`---
title: "%s"
excerpt: "%s"
coverImage: "%s"
date: "%s"
ogImage:
  url: "%s"
tags: [%s]
---

`, content.Title, content.Excerpt, content.CoverImage, date, content.OGImage.URL, strings.Join(content.Tags, ", "))

	fullContent := frontmatter + content.Content
	return os.WriteFile(path, []byte(fullContent), 0644)
}
