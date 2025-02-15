package converter

import (
	"fmt"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"io"
)

// styleName is the name of the style to use for highlighting.
//
// See all styles: https://github.com/alecthomas/chroma/tree/master/styles.
const styleName = "monokailight"

type Highlighter struct {
	formatter *html.Formatter
	style     *chroma.Style
}

func NewHighlighter(w io.Writer) (*Highlighter, error) {
	htmlFormatter := html.New(
		html.WithClasses(true),
		html.WithAllClasses(true),
		html.WithLineNumbers(true),
		html.TabWidth(2),
	)
	highlightStyle := styles.Get(styleName)
	_, err := fmt.Fprint(w, "<style>")
	if err != nil {
		return nil, err
	}
	err = htmlFormatter.WriteCSS(w, highlightStyle)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprint(w, "</style>")
	if err != nil {
		return nil, err
	}
	return &Highlighter{
		formatter: htmlFormatter,
		style:     highlightStyle,
	}, nil
}

func (h *Highlighter) HTMLHighlight(w io.Writer, source, lang string) error {
	l := lexers.Get(lang)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)
	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}
	return h.formatter.Format(w, h.style, it)
}
