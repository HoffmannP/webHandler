package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type assets []string

type button struct {
	Icon string // icon file name
	Text string // button text
	Cmd  string // command to execute
}

var bs map[string]button
var t template.Template
var as assets
var tn, h, cf string

func (as *assets) String() string {
	return strings.Join(*as, ", ")
}

func (as *assets) Set(a string) error {
	*as = append(*as, a)
	return nil
}

func main() {
	flag.StringVar(&tn, "template", "index.go.html", "Template für die Indexdatei")
	flag.StringVar(&h, "host", "1112", "Serveradresse in der Form [IP:]Port")
	flag.StringVar(&cf, "config", "config.json", "Konfigurationsdatei")
	flag.Var(&as, "assets", "Zusätzliche Dateien die ausgeliefert werden sollen (default: \"script.js\", \"style.css\")")
	flag.Parse()
	if len(as) == 0 {
		as = []string{"style.css", "script.js", "MSReferenceSansSerif.woff2", "MSReferenceSansSerif.woff", "MSReferenceSansSerif.tff"}
	}

	tt, err := template.New("index.go.html").ParseFiles("index.go.html")
	if err != nil {
		panic(err)
	}
	t = *tt

	if !strings.Contains(":", h) {
		h = ":" + h
	}

	f, err := os.Open(cf)
	if err != nil {
		panic(err)
	}

	err = json.NewDecoder(f).Decode(&bs)
	if err != nil {
		panic(err)
	}

	fmt.Println("listening at", h)
	http.HandleFunc("/", serve)
	if err := http.ListenAndServe(h, nil); err != nil {
		panic(err)
	}
}

func serve(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path[1:]

	for _, a := range as {
		if p == a {
			serveFile(w, p)
			return
		}
	}

	for n, b := range bs {
		if p == n {
			execute(w, b)
			return
		}
		if p == b.Icon {
			serveFile(w, p)
			return
		}
	}

	t.Execute(w, bs)
}

func serveFile(w http.ResponseWriter, fn string) {
	fh, err := os.Open(fn)
	if err != nil {
		fmt.Println(err)
		return
	}
	m := mime.TypeByExtension(fn[len(fn)-4:])
	w.Header().Set("Content-Type", m)
	io.Copy(w, fh)
}

func execute(w http.ResponseWriter, b button) {
	err := exec.Command("/bin/sh", "-c", b.Cmd).Run()
	if err != nil {
		w.Write([]byte(fmt.Sprintln(err)))
		return
	}
	w.Write([]byte(b.Cmd))
}
