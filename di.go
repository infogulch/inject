// Package inject is a simple dependency injector.
//
// Injection occurrs by calling functions passed to `Injector.Inject` using
// values passed duing the construction of the Injector. Values may be passed
// or required in any order, but there may only be one of each exact type. Use
// new named types to pass multiple values of the same underlying value, as in
// `type A int; inject.New(0, A(1))`. Also, injected functions may only return
// one value (and optionally an error).
//
// With these heavy restrictions, you might wonder how this could be useful.
// The answer is that this is intended to inject long-lived values into functions
// **that return closures**. Specifically this intended as an alternative to
// global variables or giant `server` structs with everything under the sun in them
// and methods on it for handing http requests.
//
// Instead of this:
//
//     type server struct {
//         db  *sql.DB
//         log *log.Logger
//         t   *template.Template
//         mut sync.RWMutex
//         // ...
//     }
//     // home has access to *all* fields, even ones it doesn't need,
//     // breaking componentation and making it harder to mock and test.
//     func (s *server) home(w http.ResponseWriter, req *http.Request) {
//         s.t.ExecuteTemplate(w, "home.html", stuff(s.db))
//     }
//     func main() {
//         s := setupServer()
//         http.HandleFunc("/", s.home)
//         http.ListendAndServe("", nil)
//     }
//
// You can do this:
//
//     // it's perfectly clear what home needs, and it's easy to mock its' dependencies
//     func home(t *template.Template, db *sql.DB) http.Handler {
//         return http.HandleFunc(func(w http.ResponseWriter, req *http.Request) {
//             s.t.ExecuteTemplate(w, "home.html", stuff(s.db))
//         })
//     }
//     func main() {
//         db, tmpl := deps()
//         di := inject.New(db, tmpl)
//         http.Handle("/", di(home)) // even better: make your router do this!
//         http.ListeAndServe(":80", nil)
//     }
//
// Injector is an interface so you can accept an Injector in your library without
// directly depending on a package that uses reflect. Just copy the definition of
// the `Injector` interface into your library and use it instead. Keep your
// dependency tree clean, but allow your users to excercise this power if they want!
//
// Under MIT license.
//
package inject

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// Injector can dynamically inject (call) functions.
type Injector interface {
	// Inject takes a function as an argument and tries to
	// call it, returning the result, or an error if there
	// was a problem.
	//
	// fn can take any number of arguments but it can only
	// return one value in addition to an optional error.
	Inject(fn interface{}) (interface{}, error)
}

// New returns a new Injector using the args for injection.
//
// vaccines are the values injected into functions passed to Inject.
// There can only be one value of a given type per Injector
func New(vaccines ...interface{}) (Injector, error) {
	n := needle{}
	for _, v := range vaccines {
		typ, val := reflect.TypeOf(v), reflect.ValueOf(v)
		if old, ok := n[typ]; ok {
			return nil, fmt.Errorf("cannot inject two values of the same type. first: %#v (%T), second: %#v (%T)",
				old.Interface(),
				old.Interface(),
				val.Interface(),
				val.Interface(),
			)
		}
		n[typ] = val
	}
	return &n, nil
}

type needle map[reflect.Type]reflect.Value

func (n needle) Inject(fn interface{}) (interface{}, error) {
	typ := reflect.TypeOf(fn)
	val := reflect.ValueOf(fn)
	var err error
	// check the function for compatibility
	if val.Kind() != reflect.Func {
		return nil, fmt.Errorf("arg is a %s, not a Func: %v", val.Kind().String(), fn)
	}
	if typ.NumOut() > 2 {
		return nil, fmt.Errorf("cannot inject function with more than 2 return values: %#v", fn)
	}
	if typ.NumOut() == 2 && typ.Out(1) != reflect.TypeOf(&err).Elem() {
		return nil, fmt.Errorf("cannot inject function with a non-error second return value: %s. %#v", typ.Out(1).String(), fn)
	}
	if typ.NumOut() < 1 {
		return nil, fmt.Errorf("cannot inject function with no return values: %#v", fn)
	}
	// build the arguments
	args := make([]reflect.Value, typ.NumIn())
	for i := range args {
		arg, ok := n[typ.In(i)]
		if !ok {
			return nil, fmt.Errorf("fn requires a type that's missing: %s. %#v", typ.In(i).String(), fn)
		}
		args[i] = arg
	}
	// call the function
	results := val.Call(args)
	// extract the optional error
	if len(results) == 2 {
		// we already know that results[1] has type error
		if i := results[1].Interface(); i != nil {
			err = errors.Wrap(i.(error), "cannot inject")
		}
	}
	return results[0].Interface(), err
}
