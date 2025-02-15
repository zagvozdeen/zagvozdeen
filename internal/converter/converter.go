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
	"net/url"
	"os"
	"strings"
	"time"
)

type Article struct {
	ID      string `json:"id"`
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Lead    string `json:"lead"`
	Author  string `json:"author"`
	Image   string `json:"image"`
	Updated string `json:"updated"`

	html  []byte
	files []string
}

type Converter struct {
	config      config.Config
	version     string
	logger      *slog.Logger
	vite        Vite
	head        *strings.Builder
	highlighter *Highlighter
}

func New(cfg config.Config) *Converter {
	return &Converter{
		config: cfg,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
		head:   &strings.Builder{},
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
	c.highlighter, err = NewHighlighter(c.head)
	if err != nil {
		c.logger.Error("Failed to create highlighter", "err", err)
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
	err = c.NewSitemap(articles)
	if err != nil {
		c.logger.Error("Failed to create sitemap", "err", err)
		return
	}
	c.logger.Info("Sitemap created")
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
			img.Destination = []byte(fmt.Sprintf("%s/assets/%s/%s", c.config.AppURL, a.ID, img.Destination))
			img.Attribute = &ast.Attribute{
				Attrs: map[string][]byte{
					"loading": []byte("lazy"),
				},
			}
		}
		return ast.GoToNext
	})
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	renderer := html.NewRenderer(html.RendererOptions{
		Flags: htmlFlags,
		RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
			if code, ok := node.(*ast.CodeBlock); ok {
				err = c.highlighter.HTMLHighlight(w, string(code.Literal), string(code.Info))
				if err != nil {
					c.logger.Error("Failed to highlight code", "err", err)
				}
				return ast.GoToNext, true
			}
			return ast.GoToNext, false
		},
	})
	a.html = markdown.Render(doc, renderer)
	return nil
}

func (c *Converter) createFiles(a *Article) error {
	err := os.Mkdir(fmt.Sprintf("dist/%s/%s", c.version, a.Slug), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.Mkdir(fmt.Sprintf("dist/assets/%s", a.ID), os.ModePerm)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	for _, f := range a.files {
		err = c.copyFile(
			fmt.Sprintf("blog/%s/%s", a.ID, f),
			fmt.Sprintf("dist/assets/%s/%s", a.ID, f),
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
		Lead      string
		Author    string
		Published string
		CreatedAt string
		UpdatedAt string
		URL       template.URL
		ImageURL  template.URL
		Content   template.HTML
		LD        template.HTML
		Head      template.HTML
		Vite      Vite
	}
	published, err := time.Parse(time.DateOnly, a.ID)
	if err != nil {
		return fmt.Errorf("failed to parse published date: %w", err)
	}
	updated, err := time.Parse(time.DateOnly, a.Updated)
	if err != nil {
		return fmt.Errorf("failed to parse updated date: %w", err)
	}
	u, err := url.JoinPath(c.config.AppURL, "assets", a.ID, a.Image)
	if err != nil {
		return fmt.Errorf("failed to join image url: %w", err)
	}
	ld, err := c.NewLD(a)
	if err != nil {
		return fmt.Errorf("failed to create ld: %w", err)
	}
	err = t.Execute(f, page{
		Title:     a.Title,
		Lead:      a.Lead,
		Author:    a.Author,
		Published: published.Format("2.01.2006"),
		CreatedAt: published.Format(time.DateOnly),
		UpdatedAt: updated.Format(time.DateOnly),
		URL:       template.URL(c.config.AppURL),
		ImageURL:  template.URL(u),
		Content:   template.HTML(a.html),
		Head:      template.HTML(c.head.String()),
		LD:        ld,
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
		if errors.Is(err, os.ErrExist) {
			return nil
		}
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
