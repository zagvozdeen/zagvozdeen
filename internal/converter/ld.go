package converter

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
)

type LD struct {
	Content       string   `json:"@context"`
	Type          string   `json:"@type"`
	Entity        LDEntity `json:"mainEntityOfPage"`
	Headline      string   `json:"headline"`
	Description   string   `json:"description"`
	Image         string   `json:"image"`
	Author        LDAuthor `json:"author"`
	DatePublished string   `json:"datePublished"`
	DateModified  string   `json:"dateModified"`
}

type LDEntity struct {
	Type string `json:"@type"`
	ID   string `json:"@id"`
}

type LDAuthor struct {
	Type string `json:"@type"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (c *Converter) NewLD(a *Article) (template.HTML, error) {
	u1, err := url.JoinPath(c.config.AppURL, "blog", a.Slug, "/")
	if err != nil {
		return "", fmt.Errorf("failed to join post url: %w", err)
	}
	u2, err := url.JoinPath(c.config.AppURL, "assets", c.version, a.ID, a.Image)
	if err != nil {
		return "", fmt.Errorf("failed to join image url: %w", err)
	}
	ld := LD{
		Content: "https://schema.org",
		Type:    "Article",
		Entity: LDEntity{
			Type: "WebPage",
			ID:   u1,
		},
		Headline:    a.Title,
		Description: a.Lead,
		Image:       u2,
		Author: LDAuthor{
			Type: "Person",
			Name: a.Author,
			URL:  c.config.AppURL,
		},
		DatePublished: a.ID,
		DateModified:  a.Updated,
	}
	js, err := json.Marshal(ld)
	if err != nil {
		return "", err
	}
	return template.HTML(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, js)), nil
}
