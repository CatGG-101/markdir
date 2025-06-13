package main

import (
	"errors"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/russross/blackfriday/v2"
)

var bind = flag.String("bind", "127.0.0.1:19000", "port to run the server on")
var dir = flag.String("d", ".", "directory to serve")

func main() {
	flag.Parse()

	httpdir := http.Dir(*dir)
	handler := renderer{httpdir, http.FileServer(httpdir)}

	log.Println("Serving on http://" + *bind)
	log.Fatal(http.ListenAndServe(*bind, handler))
}

var outputTemplate = template.Must(template.New("base").Parse(`
<html>
  <head>
    <title>{{ .Path }}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="stylesheet" href="index.css">
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/github-markdown-css/5.8.1/github-markdown-light.min.css">
	<style>
		.markdown-body {
			box-sizing: border-box;
			min-width: 200px;
			padding: 45px;
		}
		@media (max-width: 767px) {
			.markdown-body {
				padding: 15px;
			}
		}
	</style>
  </head>
  <body>
    <article class="markdown-body">
      {{ .Body }}
    </article>
  </body>
</html>
`))

type renderer struct {
	d http.Dir
	h http.Handler
}

func (r renderer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(req.URL.Path, ".md") && !strings.HasSuffix(req.URL.Path, "/guide") {
		r.h.ServeHTTP(rw, req)
		return
	}

	// net/http is already running a path.Clean on the req.URL.Path,
	// so this is not a directory traversal, at least by my testing
	var pathErr *os.PathError
	input, err := os.ReadFile(string(r.d) + req.URL.Path)
	if errors.As(err, &pathErr) {
		http.Error(rw, http.StatusText(http.StatusNotFound)+": "+req.URL.Path, http.StatusNotFound)
		log.Printf("file not found: %s", err)
		return
	}

	if err != nil {
		http.Error(rw, "Internal Server Error: "+err.Error(), 500)
		log.Printf("Couldn't read path %s: %v (%T)", req.URL.Path, err, err)
		return
	}

	output := blackfriday.Run(input)

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")

	outputTemplate.Execute(rw, struct {
		Path string
		Body template.HTML
	}{
		Path: req.URL.Path,
		Body: template.HTML(string(output)),
	})

}
