package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	log "gopkg.in/inconshreveable/log15.v2"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Listen   string
	Projects []Project
}

type User struct {
	Email    string
	Password string
	Premium  bool
}

type Project struct {
	User         string   `yaml:user,omitempty`
	Token        string   `yaml:token,omitempty`
	Organization string   `yaml:organization,omitempty`
	Owner        string   `yaml:owner,omitempty`
	Repository   string   `yaml:repositor,omitempty`
	Labels       []string `yaml:labels,omitempty`
	Assignee     string   `yaml:assignee,omitempty`
	State        string   `yaml:state,omitempty`
	Milestone    int      `yaml:milestone,omitempty`
	Origins      []string `yaml:origins,omitempty`
}

var cfgFile = flag.String("config", "/etc/gh-issue-collector/config.yml", "The file to load configuration from")
var cfg Config

func init() {
	// example with short version for long flag
	flag.StringVar(cfgFile, "c", "", "The file to load configuration from")
}

func main() {

	flag.Parse()

	cfgContents, err := ioutil.ReadFile(*cfgFile)
	if err != nil {
		panic(fmt.Sprintf("Could not read configuration file: %s", *cfgFile))
	}

	log.Info(fmt.Sprintf("Loaded configuration from %s", *cfgFile))

	err = yaml.Unmarshal(cfgContents, &cfg)
	if err != nil {
		logCritical(err, "Could not parse configuration: %s")
		panic("Configuration error")
	}

	r := mux.NewRouter()
	r.HandleFunc("/{organization}/{project}", IssueHandler).Methods("POST", "GET")
	r.HandleFunc("/{organization}/{project}/script", ScriptHandler).Methods("GET")
	http.Handle("/", r)

	log.Info(fmt.Sprintf("About to listen on %s", cfg.Listen))
	logCritical(http.ListenAndServe(cfg.Listen, nil), "Could not start server: %s")

}

func logCritical(err error, msg string) {
	if err != nil {
		log.Crit(fmt.Sprintf(msg, err.Error()))
	}
}

// Finds a project with the owner/organization and repository supplied.
func findProject(org string, repo string) (err error, project Project) {

	err = errors.New("Project not registered")

	if cfg.Projects == nil {
		return
	}

	for _, p := range cfg.Projects {
		if p.Organization == org && p.Repository == repo {
			err = nil
			project = p
		}
	}

	return

}

// Sets CORS headers for JavaScript JSON requests.
func setCorsAcl(w http.ResponseWriter, r *http.Request, p Project) {

	origin := r.Header.Get("Origin")

	if len(p.Origins) < 1 {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		return
	}

	for _, host := range p.Origins {
		if host == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			return
		}
	}

}

func IssueHandler(w http.ResponseWriter, r *http.Request) {

	v := mux.Vars(r)

	title := r.FormValue("title")
	body := r.FormValue("body")

	if title == "" || body == "" {
		http.Error(w, "Title and body required", 400)
		return
	}

	err, p := findProject(v["organization"], v["project"])
	if err != nil {
		log.Info("Project github.com/%s/%s not registered", v["organization"], v["project"])
		http.Error(w, "Project not registered", 400)
		return
	}

	setCorsAcl(w, r, p)

	if !canPostToProject(r, p) {
		http.Error(w, "Sorry, you can't post to this project", 401)
		return
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: p.Token})
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	issue := github.IssueRequest{Title: &title, Body: &body}

	if len(p.Labels) > 0 {
		issue.Labels = &p.Labels
	}

	if p.Assignee != "" {
		issue.Assignee = &p.Assignee
	}

	if p.State != "" {
		issue.State = &p.State
	}

	if p.Milestone != 0 {
		issue.Milestone = &p.Milestone
	}

	i, resp, err := client.Issues.Create(p.Organization, p.Repository, &issue)
	if err != nil {
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	writeJson(w, i)

	return

}

// Checks if a request is allowed to post a project. Does this
// by comparing the Origin, Referer, and Remote Address to the
// values allowed in the Project.Origins array.
func canPostToProject(r *http.Request, p Project) (allowed bool) {

	origin := r.Header.Get("Origin")
	referer := r.Referer()
	remote := r.RemoteAddr

	if len(p.Origins) < 1 {
		allowed = true
		return
	}

	for _, host := range p.Origins {
		if host == origin || host == referer || host == remote {
			allowed = true
			return
		}
	}

	log.Warn(fmt.Sprintf("User from %s (referer: %s, ip: %s) tried to post to %s/%s", origin, referer, remote, p.Organization, p.Repository))

	return

}

func ScriptHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/javascript")
	fmt.Fprintf(w, "/* Inject the collector in any page - coming soon! */")

	return

}

// Writes the JSON representation of i to the io.Writer
func writeJson(w io.Writer, i interface{}) {

	enc := json.NewEncoder(w)
	err := enc.Encode(&i)
	if err != nil {
		log.Error(fmt.Sprintf("Error encoding JSON: %s", err.Error()))
	}

}
