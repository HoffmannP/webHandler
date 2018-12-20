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
	"os/signal"
	"strings"
	"syscall"
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

	parseTemplate()
	parseConfig()

	if !strings.Contains(":", h) {
		h = ":" + h
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGUSR2)
	go reloadTemplate(s)

	fmt.Println("listening at", h)
	http.HandleFunc("/", serve)
	if err := http.ListenAndServe(h, nil); err != nil {
		panic(err)
	}

	close(s)
}

func parseTemplate() {
	tt, err := template.New(tn).ParseFiles(tn)
	if err != nil {
		panic(err)
	}
	t = *tt
}

func parseConfig() {
	f, err := os.Open(cf)
	if err != nil {
		panic(err)
	}

	err = json.NewDecoder(f).Decode(&bs)
	if err != nil {
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

func reloadTemplate(s chan os.Signal) {
	for sig := range s {
		fmt.Println(sig)
		fmt.Println("refreshing config")
		parseConfig()
		fmt.Println("refreshing template")
		parseTemplate()
	}
}
