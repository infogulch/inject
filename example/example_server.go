package inject_test

import (
	"database/sql"
	"html/template"
	"net/http"

	"github.com/infogulch/inject"
	_ "github.com/mattn/go-sqlite3"
)

// it's perfectly clear what home needs, and it's easy to mock its' dependencies
func home(t *template.Template, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var tm string
		db.QueryRow(`select datetime('now')`).Scan(&tm)
		t.ExecuteTemplate(w, "home.html", tm)
	}
}

func Example() {
	// make dependencies
	tmpl := template.Must(template.New("example").Parse(`{{define "home.html"}}Hello, now it's {{.}}!{{end}}`))
	db, _ := sql.Open("sqlite3", ":memory:") // this is an example, please don't ignore your errors :)
	// make injector
	di, _ := inject.New(db, tmpl)
	// start http server
	http.Handle("/", parse(di.Inject(home))) // even better: make your router do this!
	http.ListenAndServe(":8080", nil)
}

func parse(i interface{}, e error) http.Handler {
	if e != nil {
		panic(e)
	}
	return i.(http.HandlerFunc)
}
