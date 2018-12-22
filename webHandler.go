package main

import (
	"crypto/rand"
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

var cd string

var authCode string

const authFileName = "authcode"

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
	flag.StringVar(&cd, "confdir", ".", "Directory for all assets, configs, etc")
	flag.StringVar(&tn, "template", "index.go.html", "Template für die Indexdatei")
	flag.StringVar(&h, "host", "1112", "Serveradresse in der Form [IP:]Port")
	flag.StringVar(&cf, "config", "config.json", "Konfigurationsdatei")
	flag.Var(&as, "assets", "Zusätzliche Dateien die ausgeliefert werden sollen (default: \"script.js\", \"style.css\")")
	flag.Parse()
	if len(as) == 0 {
		as = []string{"style.css", "script.js", "MSReferenceSansSerif.woff2", "MSReferenceSansSerif.woff", "MSReferenceSansSerif.tff"}
	}

	authCode = loadAuthcode()
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

func loadAuthcode() string {
	f, err := os.Open(cd + "/" + authFileName)
	if err != nil {
		return createAuthcode()
	}
	defer f.Close()
	ac := make([]byte, 40)
	if n, err := f.Read(ac); n != 40 || err != nil {
		fmt.Println(cd+"/"+authFileName, "file invalid")
		f.Close()
		return createAuthcode()
	}
	fmt.Println("load authCode", string(ac))
	return string(ac)
}

func createAuthcode() string {
	ab := make([]byte, 20)
	n, err := rand.Read(ab)
	if n != 20 || err != nil {
		fmt.Println("couldn't create random AuthCode")
	}
	ac := fmt.Sprintf("%x", ab)
	fmt.Println("created new authCode", ac)
	f, err := os.OpenFile(cd+"/"+authFileName, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Println("couldn't save", cd+"/"+authFileName)
		return ""
	}
	defer f.Close()
	if n, err := f.Write([]byte(ac)); n != 40 || err != nil {
		fmt.Println("couldn't write to", cd+"/"+authFileName)
		return ""
	}
	return ac
}

func parseTemplate() {
	tt, err := template.New(tn).ParseFiles(cd + "/" + tn)
	if err != nil {
		panic(err)
	}
	t = *tt
}

func parseConfig() {
	f, err := os.Open(cd + "/" + cf)
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

	if c, err := r.Cookie("auth"); err != nil || c == nil || c.Value != authCode {
		if p == authCode {
			http.SetCookie(w, &http.Cookie{Name: "auth", Value: authCode})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - You are not allowed to access this server"))
		return
	}

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
	fh, err := os.Open(cd + "/" + fn)
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
