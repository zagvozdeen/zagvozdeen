package converter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/google/uuid"
	"github.com/zagvozdeen/zagvozdeen/config"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"time"
)

type Article struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Author string `json:"author"`

	html  []byte
	files []string
}

type Converter struct {
	config  config.Config
	version string
	logger  *slog.Logger
	vite    Vite
}

func New(cfg config.Config) *Converter {
	return &Converter{
		config: cfg,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (c *Converter) Run() {
	if !c.config.IsProduction {
		c.logger.Info("You are running in development mode")
	}
	uid, err := uuid.NewV7()
	if err != nil {
		c.logger.Error("Failed to generate new version uuid", "err", err)
		return
	}
	c.version = uid.String()
	b, err := os.ReadFile("blog/blog.json")
	if err != nil {
		c.logger.Error("Failed to read file", "err", err)
		return
	}
	var articles []Article
	err = json.Unmarshal(b, &articles)
	if err != nil {
		c.logger.Error("Failed to unmarshal json", "err", err)
		return
	}
	for i := range articles {
		err = c.handleArticle(&articles[i])
		if err != nil {
			c.logger.Error("Failed to handle article", "err", err)
			return
		}
	}
	err = os.MkdirAll(fmt.Sprintf("dist/%s", c.version), os.ModePerm)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		c.logger.Error("Failed to create dist directory", "err", err)
		return
	}
	err = os.MkdirAll(fmt.Sprintf("dist/assets/%s", c.version), os.ModePerm)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		c.logger.Error("Failed to create assets directory", "err", err)
		return
	}
	err = c.InitVite()
	if err != nil {
		c.logger.Error("Failed to init vite", "err", err)
		return
	}
	for i := range articles {
		err = c.createFiles(&articles[i])
		if err != nil {
			c.logger.Error("Failed to create files", "err", err, "article", articles[i].ID)
			return
		}
	}
	b, err = os.ReadFile("dist/version")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		c.logger.Error("Failed to read version file", "err", err)
		return
	}
	err = os.WriteFile("dist/version", []byte(c.version), os.ModePerm)
	if err != nil {
		c.logger.Error("Failed to write version file", "err", err)
		return
	}
	if b != nil {
		old := string(b)
		err = os.RemoveAll(fmt.Sprintf("dist/%s", old))
		if err != nil {
			c.logger.Error("Failed to remove old version", "err", err)
			return
		}
		c.logger.Info("Old version removed", "version", old)
	}
	c.logger.Info("Conversion completed", "version", c.version)
}

func (c *Converter) handleArticle(a *Article) error {
	md, err := os.ReadFile(fmt.Sprintf("blog/%s/index.md", a.ID))
	if err != nil {
		return err
	}
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if img, ok := node.(*ast.Image); ok && entering {
			a.files = append(a.files, string(img.Destination))
			img.Destination = []byte(fmt.Sprintf("%s/assets/%s/%s/%s", c.config.AppURL, c.version, a.ID, img.Destination))
			img.Attribute = &ast.Attribute{
				Attrs: map[string][]byte{
					"loading": []byte("lazy"),
				},
			}
		}
		return ast.GoToNext
	})
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	a.html = markdown.Render(doc, renderer)
	return nil
}

func (c *Converter) createFiles(a *Article) error {
	err := os.Mkdir(fmt.Sprintf("dist/%s/%s", c.version, a.Slug), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.Mkdir(fmt.Sprintf("dist/assets/%s/%s", c.version, a.ID), os.ModePerm)
	if err != nil {
		return err
	}
	for _, f := range a.files {
		err = c.copyFile(
			fmt.Sprintf("blog/%s/%s", a.ID, f),
			fmt.Sprintf("dist/assets/%s/%s/%s", c.version, a.ID, f),
		)
		if err != nil {
			c.logger.Error("Failed to copy file", "err", err, "file", f)
		}
	}
	t, err := template.ParseFiles("web/layout.html")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(
		fmt.Sprintf("dist/%s/%s/index.html", c.version, a.Slug),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		os.ModePerm,
	)
	if err != nil {
		return err
	}
	type page struct {
		Title     string
		Author    string
		Published string
		Content   template.HTML
		Vite      Vite
	}
	published, err := time.Parse(time.DateOnly, a.ID)
	if err != nil {
		return fmt.Errorf("failed to parse date: %w", err)
	}
	err = t.Execute(f, page{
		Title:     a.Title,
		Author:    a.Author,
		Published: published.Format("2 January 2006"),
		Content:   template.HTML(a.html),
		Vite:      c.vite,
	})
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func (c *Converter) copyFile(from, to string) error {
	source, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() {
		err := source.Close()
		if err != nil {
			c.logger.Error("Failed to close source file", "err", err)
		}
	}()
	destination, err := os.Create(to)
	if err != nil {
		return err
	}
	defer func() {
		err := destination.Close()
		if err != nil {
			c.logger.Error("Failed to close destination file", "err", err)
		}
	}()
	_, err = io.Copy(destination, source)
	return err
}
