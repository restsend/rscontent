package main

import (
	"net/http"
	"os"
	"testing"

	"github.com/flosch/pongo2/v6"
	"github.com/stretchr/testify/assert"
)

func TestContentJsonBlock(t *testing.T) {
	{
		c := MarkdownContent{}
		data := `---
	{"layout":"hello.html"}
---
## hello`
		c.Prepare([]byte(data))
		assert.NotNil(t, c.ctx)
		assert.Equal(t, c.ctx[keyLayout], "hello.html")
		assert.Equal(t, string(c.htmlData), "<h2>hello</h2>\n")
	}
	{
		c := MarkdownContent{}
		data := `---
		{"layout":"pricing.html"}
---
## pricing`
		c.Prepare([]byte(data))
		assert.NotNil(t, c.ctx)
		assert.Equal(t, c.ctx[keyLayout], "pricing.html")
		assert.Equal(t, string(c.htmlData), "<h2>pricing</h2>\n")
	}
}
func TestContentException(t *testing.T) {
	m := ContentManager{}
	for _, v := range []string{"bad_dir_name.html", "/.env", "/blog/.env"} {
		_, err := m.Open(v)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid character")
	}
	for _, v := range []string{"/not_exist.html"} {
		_, err := m.Open(v)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	}
	{
		_, err := m.Get("/index.html", nil)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	}
	defer func() {
		os.RemoveAll("/tmp/404.md")
	}()
	os.WriteFile("/tmp/404.md", []byte("4o4"), 0666)
	m.AddLoader(http.Dir("/tmp/"))
	{
		d, err := m.Get("/index.html", nil)
		assert.Nil(t, err)
		assert.Contains(t, string(d), "4o4")
	}
}
func TestContentOpen(t *testing.T) {
	m := ContentManager{}
	defer func() {
		os.RemoveAll("/tmp/index.md")
		os.RemoveAll("/tmp/index.css")
	}()
	os.WriteFile("/tmp/index.md", []byte("index data"), 0666)
	os.WriteFile("/tmp/index.css", []byte(".index{}"), 0666)
	m.AddLoader(http.Dir("/tmp/"))
	{
		d, err := m.Get("/", nil)
		assert.Nil(t, err)
		assert.Equal(t, string(d), "<p>index data</p>\n")
	}
	{
		d, err := m.Get("/index.css", nil)
		assert.Nil(t, err)
		assert.Equal(t, string(d), ".index{}")
	}
}
func TestContentRender(t *testing.T) {
	m := ContentManager{}
	defer func() {
		os.RemoveAll("/tmp/index.md")
		os.RemoveAll("/tmp/about.md")
		os.RemoveAll("/tmp/pricing.md")
		os.RemoveAll("/tmp/index.html")
		os.RemoveAll("/tmp/page.html")
		os.RemoveAll("/tmp/pricing.html")
	}()
	os.WriteFile("/tmp/index.md", []byte("index data"), 0666)
	os.WriteFile("/tmp/about.md", []byte("about"), 0666)
	os.WriteFile("/tmp/pricing.md", []byte(`---
	{"layout":"pricing.html"}
---
## pricing`), 0666)
	os.WriteFile("/tmp/index.html", []byte("<html>{{content|safe}}</html>"), 0666)
	os.WriteFile("/tmp/page.html", []byte("<main>{{content|safe}}</main>"), 0666)
	os.WriteFile("/tmp/pricing.html", []byte("<pricing>{{content|safe}}</pricing>"), 0666)
	m.AddLoader(http.Dir("/tmp/"))
	m.Sets = pongo2.NewSet("unittest", pongo2.MustNewLocalFileSystemLoader("/tmp"))
	{
		d, err := m.Get("/", nil)
		assert.Nil(t, err)
		assert.Equal(t, string(d), "<html><p>index data</p>\n</html>")
	}
	{
		d, err := m.Get("/about", nil)
		assert.Nil(t, err)
		assert.Equal(t, string(d), "<main><p>about</p>\n</main>")
	}
	{
		d, err := m.Get("/pricing", nil)
		assert.Nil(t, err)
		assert.Equal(t, string(d), "<pricing><h2>pricing</h2>\n</pricing>")
	}
}

func TestMergeContext(t *testing.T) {
	c := MarkdownContent{}
	vals := map[string]interface{}{
		"title": "hello",
	}
	c.MergeContext(vals)
	assert.Contains(t, c.ctx, "title")
}
