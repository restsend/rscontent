package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2/v6"
	"github.com/gin-gonic/gin"
	"github.com/sevlyar/go-daemon"
)

var fileMode os.FileMode = 0666

var serverAddr string
var rootDir string
var logFile string
var err error
var runDaemon bool
var m *ContentManager

func LoadContext() map[string]interface{} {
	fname := filepath.Join(rootDir, "config.json")
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil
	}
	ctx := make(map[string]interface{})
	err = json.Unmarshal(data, &ctx)
	if err != nil {
		return nil
	}
	return ctx
}

func HandleMarkdownContent() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m == nil {
			return
		}
		ctx := LoadContext()
		buf, err := m.Get(c.Request.RequestURI, ctx)
		if err != nil {
			return
		}
		c.Data(http.StatusOK, "text/html", buf)
	}
}

func main() {

	flag.StringVar(&rootDir, "r", ".", "root dir")
	flag.StringVar(&serverAddr, "s", ":8080", "listen addr")
	flag.StringVar(&logFile, "l", "", "log file")
	flag.BoolVar(&runDaemon, "d", false, "run as daemon")

	flag.Parse()

	var lw io.Writer
	if logFile != "" {
		lw, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, fileMode)
		if err != nil {
			log.Printf("open %s fail, %v\n", logFile, err)
		}
	}

	r := gin.New()
	r.Use(gin.LoggerWithWriter(lw),
		gin.Recovery())

	r.Use(HandleMarkdownContent())

	staticDir := filepath.Join(rootDir, "static")
	contentDir := filepath.Join(rootDir, "content")
	templateDir := filepath.Join(rootDir, "template")

	r.StaticFS("/static", http.Dir(staticDir))

	m = &ContentManager{}
	m.AddLoader(http.Dir(contentDir))
	m.Sets = pongo2.NewSet("rscontent", pongo2.MustNewLocalFileSystemLoader(templateDir))

	log.Println("static dir:", staticDir)
	log.Println("content dir:", contentDir)
	log.Println("template dir:", templateDir)

	if serverAddr[0] == ':' || strings.HasPrefix(serverAddr, "0.0.0.0:") {
		log.Printf("Serving HTTP on (http://localhost:%s/) ... \n", strings.Split(serverAddr, ":")[1])
	} else {
		log.Printf("Serving HTTP on (http://%s/) ... \n", serverAddr)
	}
	if runDaemon {
		cntxt := &daemon.Context{
			WorkDir: ".",
		}
		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatal("Unable to run: ", err)
		}
		if d != nil {
			return
		}
		defer cntxt.Release()
		r.Run(serverAddr)
	} else {
		r.Run(serverAddr)
	}

}
