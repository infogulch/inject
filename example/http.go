package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/infogulch/inject"
	_ "github.com/mattn/go-sqlite3"
)

func home(t *template.Template, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var tm string
		db.QueryRow(`select datetime('now')`).Scan(&tm)
		t.ExecuteTemplate(w, "home.html", tm)
	}
}

type middleware func(http.Handler) http.Handler

func logMiddleware(lg *log.Logger) middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			lg.Printf("before: %s\n", req.URL.Path)
			h.ServeHTTP(w, req)
			lg.Println("after")
		})
	}
}

// deps gets all the dependencies and returns them in an Injector
func deps() inject.Injector {
	// this is an example, please don't ignore your errors :)
	tmpl := template.Must(template.New("example").Parse(`{{define "home.html"}}Hello, now it's {{.}}!{{end}}`))
	db, _ := sql.Open("sqlite3", ":memory:")
	lg := log.New(os.Stderr, "CUSTOM LOGGER: ", log.LstdFlags)
	di, _ := inject.New(db, tmpl, lg)
	return di
}

func main() {
	di := deps() // get dependencies
	// If any dependencies change, main does't have to be updated.
	//
	// This could all be done by your router to make it cleaner. See goji.go for
	// an example.
	h := inject.Must(di.Inject(home)).(http.HandlerFunc)
	mid := inject.Must(di.Inject(logMiddleware)).(middleware)
	http.Handle("/", mid(h))
	http.ListenAndServe(":8080", nil)
}
