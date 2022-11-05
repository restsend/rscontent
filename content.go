package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2/v6"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type GetContext func(name string) map[string]interface{}
type GetExceptionContent func(name string, status int) string

type ContentManager struct {
	loaders             []http.FileSystem
	FallbackTemplate    string
	Sets                *pongo2.TemplateSet
	GetContext          GetContext
	GetExceptionContent GetExceptionContent
}

type MarkdownContent struct {
	ctx      map[string]interface{}
	htmlData []byte
}

type BlockJson struct {
	ast.Container
}

const (
	jsonFormaterDelim       = "---\n"
	keyLayout               = "layout"
	keyContent              = "content"
	tplSuffix               = ".html"
	defaultFallbackTemplate = `<html><head><title>{{title}}</title></head><body>{{content|safe}}</body><!--fallback render--></html>`
)

// for gomarkdown
func (c *MarkdownContent) blockHook(data []byte) (ast.Node, []byte, int) {
	i := 0
	s := len(jsonFormaterDelim)
	if len(data) < s {
		return nil, data, 0
	}

	if bytes.Index(data, []byte(jsonFormaterDelim)) != 0 {
		return nil, data, 0
	}

	i = bytes.Index(data[s:], []byte(jsonFormaterDelim))
	if i == -1 {
		return nil, data, 0
	}
	node := &BlockJson{}
	buf := data[s : s+i]

	if err := json.Unmarshal(buf, &c.ctx); err != nil {
		return node, nil, i
	}
	return node, nil, i + s + s
}

func (c *MarkdownContent) renderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	_, ok := node.(*BlockJson)
	if !ok {
		return ast.GoToNext, false
	}
	if entering {
		return ast.SkipChildren, true
	}
	return ast.GoToNext, true
}

func (c *MarkdownContent) Prepare(data []byte) {
	c.ctx = make(map[string]interface{})
	renderer := html.NewRenderer(html.RendererOptions{
		Flags:          html.CommonFlags,
		RenderNodeHook: c.renderHook,
	})
	p := parser.New()
	p.Opts = parser.Options{ParserHook: c.blockHook}
	c.htmlData = markdown.ToHTML(data, p, renderer)
}

// Merge Parent into content, overwrite when content not exist
func (m *MarkdownContent) MergeContext(parent map[string]interface{}) {
	if m.ctx == nil {
		m.ctx = make(map[string]interface{})
	}
	for k, v := range parent {
		_, ok := m.ctx[k]
		if !ok {
			m.ctx[k] = v
		}
	}
}

func (m *ContentManager) AddLoader(fs http.FileSystem) {
	m.loaders = append(m.loaders, fs)
}

func (m *ContentManager) Open(name string) (http.File, error) {
	if !strings.HasPrefix(name, "/") || strings.Contains(name, "/.") {
		return nil, errors.New("content: invalid character in file path")
	}
	for _, loader := range m.loaders {
		f, err := loader.Open(name)
		if err == nil {
			return f, nil
		}
	}
	return nil, errors.New("content: resource not found")
}

func (m *ContentManager) Get(name string, ctx map[string]interface{}) ([]byte, error) {
	if strings.HasSuffix(name, "/") {
		name += "index.md"
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		name += ".md"
		ext = ".md"
	}

	f, err := m.Open(name)
	if err != nil {
		return m.HandleException(name, err, http.StatusNotFound, ctx)
	}
	if ext == ".md" { // Markdown
		return m.RenderContent(name, f, ctx)
	}
	return io.ReadAll(f)
}

// Handle 404 404.md
func (m *ContentManager) HandleException(name string, prevErr error, status int, ctx map[string]interface{}) ([]byte, error) {
	dir := filepath.Dir(name)
	handle := m.GetExceptionContent

	if handle == nil {
		handle = func(name string, c int) string {
			switch c {
			case http.StatusNotFound:
				return "404.md"
			}
			return "500.md"
		}
	}

	fname := filepath.Join(dir, handle(name, status))
	f, err := m.Open(fname)
	if err != nil {
		return nil, prevErr
	}
	return m.RenderContent(name, f, ctx)
}

func (m *ContentManager) MatchLayout(name string) string {
	dir := filepath.Dir(name)
	if dir[0] == '/' {
		dir = dir[1:]
	}
	fname := strings.ToLower(filepath.Base(name))
	tplName := "page"
	if fname == "index.md" || fname == "readme.md" {
		tplName = "index"
	}
	return filepath.Join(dir, tplName+tplSuffix)
}

func (m *ContentManager) RenderContent(name string, f http.File, ctx map[string]interface{}) ([]byte, error) {
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	content := MarkdownContent{}
	// Markdown to html preprocess
	content.Prepare(data)

	if m.Sets == nil {
		return content.htmlData, err
	}

	if ctx != nil {
		content.MergeContext(ctx)
	}

	if m.GetContext != nil {
		content.MergeContext(m.GetContext(name))
	}

	layout := m.MatchLayout(name)
	if v, ok := content.ctx["layout"]; ok {
		if lv, ok := v.(string); ok {
			layout = lv
		}
	}

	content.ctx[keyContent] = string(content.htmlData)
	t, err := m.Sets.FromFile(layout)
	if err != nil {
		if !strings.Contains(err.Error(), "unable to resolve template") {
			return nil, err
		}
		fallback := defaultFallbackTemplate
		if m.FallbackTemplate != "" {
			fallback = m.FallbackTemplate
		}
		t, err = m.Sets.FromString(fallback)
		if err != nil {
			return nil, err
		}
	}

	return t.ExecuteBytes(content.ctx)
}
