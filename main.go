package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/gin-gonic/gin"
	"github.com/sevlyar/go-daemon"
)

var fileMode os.FileMode = 0666
var dirMode os.FileMode = 0766

var serverAddr string
var rootDir string
var outputDir string
var logFile string
var err error
var runDaemon bool
var runBuild bool
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
		uri := strings.TrimSuffix(c.Request.RequestURI, ".html")
		buf, err := m.Get(uri, ctx)
		if err != nil {
			return
		}
		c.Data(http.StatusOK, "text/html", buf)
	}
}

func SizeReadable(v int) string {
	if v < 1024 {
		return fmt.Sprintf("%d B", v)
	}
	x := float64(v / 1024.0)
	if x < 1024 {
		return fmt.Sprintf("%.1f KB", x)
	}
	x = x / 1024.0
	if x < 1024 {
		return fmt.Sprintf("%.1f MB", x)
	}
	return fmt.Sprintf("%.1f GB", x)
}

func IsDir(name string) bool {
	st, err := os.Stat(name)
	if err != nil {
		return false
	}
	return st.IsDir()
}

func WalkContents(dir, suffix string) []string {
	gs, err := filepath.Glob(dir + "/*")
	var flist []string
	if err != nil {
		log.Printf("glob error in '%s' %v", dir, err)
		return flist
	}
	for _, v := range gs {
		if IsDir(v) {
			flist = append(flist, WalkContents(v, suffix)...)
		}
		if strings.HasSuffix(v, suffix) {
			flist = append(flist, v)
		}
	}
	return flist
}

func main() {
	flag.StringVar(&rootDir, "r", ".", "root dir")
	flag.StringVar(&serverAddr, "s", ":8080", "listen addr")
	flag.StringVar(&logFile, "l", "", "log file")
	flag.BoolVar(&runDaemon, "d", false, "run as daemon")
	flag.BoolVar(&runBuild, "b", false, "build static html, default: false")
	flag.StringVar(&outputDir, "o", "dist", "output dir, default: dist")

	flag.Parse()

	var lw io.Writer
	if logFile != "" {
		lw, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, fileMode)
		if err != nil {
			log.Printf("open %s fail, %v\n", logFile, err)
		}
	}

	if !IsDir(rootDir) {
		log.Printf("the given path '%s' is not a directory", rootDir)
		return
	}

	staticDir := filepath.Join(rootDir, "static")
	contentDir := filepath.Join(rootDir, "content")
	templateDir := filepath.Join(rootDir, "template")

	log.Println("---------------------------")
	log.Println("static dir:", staticDir)
	log.Println("content dir:", contentDir)
	log.Println("template dir:", templateDir)
	log.Println("---------------------------")
	//
	// create default layout
	os.MkdirAll(staticDir, fileMode)
	os.MkdirAll(contentDir, fileMode)
	os.MkdirAll(templateDir, fileMode)

	m = &ContentManager{}
	m.AddLoader(http.Dir(contentDir))
	m.Sets = pongo2.NewSet("rscontent", pongo2.MustNewLocalFileSystemLoader(templateDir))

	if runBuild {
		log.Printf("Build '%s'....", contentDir)
		//
		flist := WalkContents(contentDir, ".md")

		ctx := LoadContext()
		for _, v := range flist {
			st := time.Now()
			buf, err := m.Get(strings.TrimPrefix(v, contentDir), ctx)
			if err != nil {
				log.Printf("render '%s' fail, %v", v, err)
				return
			}
			outname := strings.TrimSuffix(strings.TrimPrefix(v, contentDir), ".md") + ".html"
			outname = filepath.Join(outputDir, outname)
			os.MkdirAll(filepath.Dir(outname), dirMode)

			err = os.WriteFile(outname, buf, fileMode)
			if err != nil {
				log.Printf("write '%s' fail, %v", v, err)
				return
			}
			log.Printf(" '%s' => '%s' size %s  usage %d ms", v, outname, SizeReadable(len(buf)), time.Since(st).Milliseconds())
		}
		log.Printf("Done, total: %d files", len(flist))
		return
	}

	r := gin.New()
	r.Use(gin.LoggerWithWriter(lw),
		gin.Recovery())
	r.Use(HandleMarkdownContent())
	r.StaticFS("/static", http.Dir(staticDir))

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
