package converter

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
)

type Sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []SitemapRow `xml:"url"`
}

type SitemapRow struct {
	Location        string  `xml:"loc"`
	LastModified    string  `xml:"lastmod"`
	Priority        float32 `xml:"priority"`
	ChangeFrequency string  `xml:"changefreq"`
}

func (c *Converter) NewSitemap(articles []Article) error {
	s := Sitemap{
		Xmlns: "https://www.sitemaps.org/schemas/sitemap/0.9",
	}
	for _, article := range articles {
		u, err := url.JoinPath(c.config.AppURL, "blog", article.Slug, "/")
		if err != nil {
			return err
		}
		s.URLs = append(s.URLs, SitemapRow{
			Location:        u,
			LastModified:    article.Updated,
			Priority:        0.5,
			ChangeFrequency: "monthly",
		})
	}
	file, err := os.Create("public/sitemap.xml")
	if err != nil {
		return fmt.Errorf("error creating XML file: %w", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			c.logger.Error("Failed to close XML file", "err", err)
		}
	}()
	_, err = file.WriteString(xml.Header)
	if err != nil {
		return err
	}
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "\t")
	err = encoder.Encode(&s)
	if err != nil {
		return fmt.Errorf("error encoding XML to file: %w", err)
	}
	return nil
}
