package model

// Content represents any content type with common fields
type Content struct {
	Title      string   `yaml:"title" json:"title"`
	Excerpt    string   `yaml:"excerpt" json:"excerpt"`
	CoverImage string   `yaml:"coverImage" json:"coverImage"`
	Date       string   `yaml:"date" json:"date"`
	OGImage    OGImage  `yaml:"ogImage" json:"ogImage"`
	Tags       []string `yaml:"tags" json:"tags"`
	Content    string   `yaml:"-" json:"content"`
	Slug       string   `yaml:"-" json:"slug"`

	// Content type metadata (not in frontmatter)
	TypeSlug string `yaml:"-" json:"typeSlug"`
}

type OGImage struct {
	URL string `yaml:"url" json:"url"`
}

// ToContent converts BlogPost to generic Content
func (p BlogPost) ToContent(typeSlug string) Content {
	return Content{
		Title:      p.Title,
		Excerpt:    p.Excerpt,
		CoverImage: p.CoverImage,
		Date:       p.Date,
		OGImage:    OGImage{URL: p.OGImage.URL},
		Tags:       p.Tags,
		Content:    p.Content,
		Slug:       p.Slug,
		TypeSlug:   typeSlug,
	}
}
