// Command example is a sample application built with Goji. Its goal is to give
// you a taste for what Goji looks like in the real world by artificially using
// all of its features.
//
// In particular, this is a complete working site for gritter.com, a site where
// users can post 140-character "greets". Any resemblance to real websites,
// alive or dead, is purely coincidental.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"io/ioutil"
	"net/http"

	// "path/filepath"

	"regexp"
	"strconv"
	"time"

	"github.com/goji/param"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"github.com/zenazn/goji/web/middleware"
	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/v2"

	"go.elastic.co/apm/module/apmsql/v2"
	_ "go.elastic.co/apm/module/apmsql/v2/sqlite3"

	// "github.com/olivere/elastic"
	// "go.elastic.co/apm/module/apmelasticsearch/v2"

	// apmgin "go.elastic.co/apm/module/apmgin/v2"
	"apmgoji"
)

var client *http.Client

var db *sql.DB

// var elasticClient, _ = elastic.NewClient(elastic.SetHttpClient(&http.Client{
// 	Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport),
// }), elastic.SetBasicAuth("elastic", "pass123"))

// func init() {
// 	filePath, _ := filepath.Abs("C:\\Users\\Sonika.Prakash\\GitHub\\goji web app\\web.log")
// 	openLogFile, _ := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
// 	Info = log.New(openLogFile, "\tINFO\t", log.Ldate|log.Ltime|log.Lmsgprefix|log.Lshortfile)
// 	Debug = log.New(openLogFile, "\tDEBUG\t", log.Ldate|log.Ltime|log.Lmsgprefix|log.Lshortfile)
// 	Error = log.New(openLogFile, "\tERROR\t", log.Ldate|log.Ltime|log.Lmsgprefix|log.Lshortfile)
// 	Warn = log.New(openLogFile, "\tWARN\t", log.Ldate|log.Ltime|log.Lmsgprefix|log.Lshortfile)
// }

// Note: the code below cuts a lot of corners to make the example app simple.

func main() {
	var err error
	db, err = apmsql.Open("sqlite3", ":memory:")
	if err != nil {
		Error.Println(err)
	}
	if _, err := db.Exec("CREATE TABLE stats (name TEXT PRIMARY KEY, count INTEGER);"); err != nil {
		Error.Println(err)
	}

	client = apmhttp.WrapClient(http.DefaultClient)

	goji.Use(goji.DefaultMux.Router)
	goji.Use(apmgoji.Middleware())
	goji.Get("/", Root)
	// goji.Get("/hello", func(c web.C, w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintf(w, "Why hello there!")
	// })

	goji.Get("/hello/:name", HelloHandler)

	// Fully backwards compatible with net/http's Handlers
	goji.Get("/greets", http.RedirectHandler("/", 301))
	// Use your favorite HTTP verbs
	goji.Post("/greets", NewGreet)
	// Use Sinatra-style patterns in your URLs
	goji.Get("/users/:name", GetUser)
	// Goji also supports regular expressions with named capture groups.
	goji.Get(regexp.MustCompile(`^/greets/(?P<id>\d+)$`), GetGreet)

	// Middleware can be used to inject behavior into your app. The
	// middleware for this application are defined in middleware.go, but you
	// can put them wherever you like.
	goji.Use(PlainText)

	// If the patterns ends with "/*", the path is treated as a prefix, and
	// can be used to implement sub-routes.
	admin := web.New()
	goji.Handle("/admin/*", admin)

	// The standard SubRouter middleware helps make writing sub-routers
	// easy. Ordinarily, Goji does not manipulate the request's URL.Path,
	// meaning you'd have to repeat "/admin/" in each of the following
	// routes. This middleware allows you to cut down on the repetition by
	// eliminating the shared, already-matched prefix.
	admin.Use(middleware.SubRouter)
	// You can also easily attach extra middleware to sub-routers that are
	// not present on the parent router. This one, for instance, presents a
	// password prompt to users of the admin endpoints.
	admin.Use(SuperSecure)

	admin.Get("/", AdminRoot)
	admin.Get("/finances", AdminFinances)

	// Goji's routing, like Sinatra's, is exact: no effort is made to
	// normalize trailing slashes.
	goji.Get("/admin", http.RedirectHandler("/admin/", 301))

	// Use a custom 404 handler
	goji.NotFound(NotFound)

	// Sometimes requests take a long time.
	goji.Get("/waitforit", WaitForIt)

	goji.Get("/zip", GetZip)
	goji.Get("/test", TestHandler)

	// external API calls
	goji.Get("/randomuser", GetRandomUser)
	goji.Get("/getregion", GetRegion)
	goji.Get("/getzipcode", GetZipCodeInfo)
	// goji.Get("/elastic", ElasticHandler)

	// Call Serve() at the bottom of your main() function, and it'll take
	// care of everything else for you, including binding to a socket (with
	// automatic support for systemd and Einhorn) and supporting graceful
	// shutdown on SIGINT. Serve() is appropriate for both development and
	// production.
	goji.Serve()
}

// func ElasticHandler(w http.ResponseWriter, r *http.Request) {
// 	// result, err := elasticClient.Search("index").Query(elastic.NewMatchAllQuery()).Do(r.Context())
// 	exists, err := elasticClient.IndexExists("index-01").Do(r.Context())
// 	if err != nil {
// 		Error.Println("elastic search error: ", err)
// 	}
// 	if exists {
// 		io.WriteString(w, "index index-01 exists")
// 	} else {
// 		io.WriteString(w, "index index-01 does not exists")
// 	}
// 	Info.Println("index exists: ", exists)
// }

// GetRandomUser makes an outgoing http request
func GetRandomUser(c web.C, w http.ResponseWriter, r *http.Request) {
	time.Sleep(100 * time.Millisecond)
	span, ctx := apm.StartSpan(r.Context(), "getRandomUser", "custom")
	defer span.End()
	req, _ := http.NewRequest("GET", "https://randomuser.me/api/", nil)
	// client := apmhttp.WrapClient(http.DefaultClient)
	resp, _ := client.Do(req.WithContext(ctx))
	defer resp.Body.Close() // this is mandatory for a span to be completed and sent to server
	body, _ := ioutil.ReadAll(resp.Body)
	sb := string(body)
	io.WriteString(w, sb)
	// resp, _ := http.Get("https://randomuser.me/api/")
	// body, _ := ioutil.ReadAll(resp.Body)
	// sb := string(body)
	// // Info.Println("API response: ", sb)
	// io.WriteString(w, sb)
}

func GetRegion(w http.ResponseWriter, r *http.Request) {
	time.Sleep(100 * time.Millisecond)
	span, ctx := apm.StartSpan(r.Context(), "getRegion", "custom")
	defer span.End()
	time.Sleep(150 * time.Millisecond)
	req, _ := http.NewRequest("GET", "https://ipinfo.io/161.185.160.93/geo", nil)
	resp, _ := client.Do(req.WithContext(ctx))
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	sb := string(body)
	io.WriteString(w, sb)
}

func GetZip(w http.ResponseWriter, r *http.Request) {
	// In the real world you'd probably use a template or something.
	time.Sleep(200 * time.Millisecond)
	io.WriteString(w, "33162")
}

// GetZipCodeInfo gives related transactions as there is one internal http request and the other is a remote api
func GetZipCodeInfo(w http.ResponseWriter, r *http.Request) {
	span, ctx := apm.StartSpan(r.Context(), "getZipCodeInfo", "custom")
	defer span.End()
	req, _ := http.NewRequest("GET", "http://127.0.0.1:8000/zip", nil)
	resp, _ := client.Do(req.WithContext(ctx))
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	zipCode := string(body)
	time.Sleep(100 * time.Millisecond)
	reqNew, _ := http.NewRequest("GET", "https://api.zippopotam.us/us/"+zipCode, nil)
	respNew, _ := client.Do(reqNew.WithContext(ctx))
	defer respNew.Body.Close()
	bodyNew, _ := ioutil.ReadAll(respNew.Body)
	sb := string(bodyNew)
	io.WriteString(w, sb)
}

func HelloHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	userName := c.URLParams["name"]
	getTraceLabels(r.Context())
	Debug.Print("Name: ", userName)
	requestCount, _ := updateRequestCount(r.Context(), userName)
	Debug.Printf("Request count: %d", requestCount)
	fmt.Fprintf(w, "Hello, %s! (#%d)\n", userName, requestCount)
}

// updateRequestCount increments a count for name in db, returning the new count.
func updateRequestCount(ctx context.Context, name string) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	row := tx.QueryRowContext(ctx, "SELECT count FROM stats WHERE name=?", name)
	var count int
	switch err := row.Scan(&count); err {
	case nil:
		count++
		if _, err := tx.ExecContext(ctx, "UPDATE stats SET count=? WHERE name=?", count, name); err != nil {
			return -1, err
		}
	case sql.ErrNoRows:
		count = 1
		if _, err := tx.ExecContext(ctx, "INSERT INTO stats (name, count) VALUES (?, ?)", name, count); err != nil {
			return -1, err
		}
	default:
		return -1, err
	}
	return count, tx.Commit()
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	// this will be traced but seperately, not as a span
	// to get it traced as a span, use apm's startspan method
	// it will then be traced as a related transaction
	resp, _ := http.Get("http://127.0.0.1:8000/zip")
	body, _ := ioutil.ReadAll(resp.Body)
	sb := string(body)
	// Info.Println("API response: ", sb)
	io.WriteString(w, sb)
}

// Root route (GET "/"). Print a list of greets.
func Root(w http.ResponseWriter, r *http.Request) {
	// labels := getTraceLabels(r.Context())
	getTraceLabels(r.Context())
	// Debug.Println(fmt.Sprintf(logFormat, "User has hit the url 127.0.0.1:8000/", labels["transaction.id"], labels["trace.id"], labels["span.id"]))
	Info.Println("User has hit the url 127.0.0.1:8000/")
	// In the real world you'd probably use a template or something.
	Debug.Println("no. of greets:", len(Greets))
	io.WriteString(w, "Gritter\n======\n\n")
	for i := len(Greets) - 1; i >= 0; i-- {
		Greets[i].Write(w)
	}
}

// NewGreet creates a new greet (POST "/greets"). Creates a greet and redirects
// you to the created greet.
//
// To post a new greet, try this at a shell:
// $ now=$(date +'%Y-%m-%dT%H:%M:%SZ')
// $ curl -i -d "user=carl&message=Hello+World&time=$now" localhost:8000/greets
// example: curl -i -d "user=sonika&message=Hello+World&time=2022-07-07T16:14:10Z" localhost:8000/greets
func NewGreet(w http.ResponseWriter, r *http.Request) {
	var greet Greet

	// Parse the POST body into the Greet struct. The format is the same as
	// is emitted by (e.g.) jQuery.param.
	r.ParseForm()
	err := param.Parse(r.Form, &greet)

	if err != nil || len(greet.Message) > 140 {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// We make no effort to prevent races against other insertions.
	Greets = append(Greets, greet)
	url := fmt.Sprintf("/greets/%d", len(Greets)-1)
	http.Redirect(w, r, url, http.StatusCreated)
}

// GetUser finds a given user and her greets (GET "/user/:name")
func GetUser(c web.C, w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Gritter\n======\n\n")
	handle := c.URLParams["name"]
	user, ok := Users[handle]
	if !ok {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	user.Write(w, handle)

	io.WriteString(w, "\nGreets:\n")
	for i := len(Greets) - 1; i >= 0; i-- {
		if Greets[i].User == handle {
			Greets[i].Write(w)
		}
	}
}

// GetGreet finds a particular greet by ID (GET "/greets/\d+"). Does no bounds
// checking, so will probably panic.
func GetGreet(c web.C, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(c.URLParams["id"])
	if err != nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}
	// This will panic if id is too big. Try it out!
	greet := Greets[id]

	io.WriteString(w, "Gritter\n======\n\n")
	greet.Write(w)
}

// WaitForIt is a particularly slow handler (GET "/waitforit"). Try loading this
// endpoint and initiating a graceful shutdown (Ctrl-C) or Einhorn reload. The
// old server will stop accepting new connections and will attempt to kill
// outstanding idle (keep-alive) connections, but will patiently stick around
// for this endpoint to finish. How kind of it!
func WaitForIt(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "This is going to be legend... (wait for it)\n")
	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}
	time.Sleep(15 * time.Second)
	io.WriteString(w, "...dary! Legendary!\n")
}

// AdminRoot is root (GET "/admin/root"). Much secret. Very administrate. Wow.
func AdminRoot(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Gritter\n======\n\nSuper secret admin page!\n")
}

// AdminFinances would answer the question 'How are we doing?'
// (GET "/admin/finances")
func AdminFinances(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Gritter\n======\n\nWe're broke! :(\n")
}

// NotFound is a 404 handler.
func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Umm... have you tried turning it off and on again?", 404)
}
