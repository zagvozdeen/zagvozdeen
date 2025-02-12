package converter

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
)

type Vite struct {
	Tags template.HTML
}

func (c *Converter) InitVite() error {
	if !c.config.IsProduction {
		c.vite = Vite{
			Tags: `<script type="module" src="http://localhost:5173/@vite/client"></script>
<script type="module" src="http://localhost:5173/web/index.css"></script>`,
		}
		return nil
	}
	b, err := os.ReadFile("dist/.vite/manifest.json")
	if err != nil {
		return fmt.Errorf("failed to manifest: %w", err)
	}
	type manifest struct {
		Entry struct {
			File string `json:"file"`
		} `json:"web/index.css"`
	}
	var m manifest
	err = json.Unmarshal(b, &m)
	if err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	c.vite = Vite{
		Tags: template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="%s/%s">`, c.config.AppURL, m.Entry.File)),
	}
	return nil
}
