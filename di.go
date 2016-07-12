/*
Package inject is a simple dependency injector.

Injection occurrs by passing a function to Injector.Inject. That function is
then called dynamically, using values passed duing the construction of the
Injector as arguments. Values may be passed or required in any order, but there
may only be one of each exact type. Use new named types to pass multiple
instances of the same underlying value, as in `type A int; inject.New(0,A(1))`.
Also, injected functions may only return one value (and optionally an error).

With such heavy restrictions, one might wonder if it's this useless on purpose.
The answer is that this is intended to inject long-lived values into functions
*that return closures*. For example, this can be an alternative to global
variables or giant server structs with everything under the sun in them and
methods on it for handing http requests. With this pattern injection happens
only once, at startup, so the cost of reflection is not paid on each request.
You can find a fleshed out example http server in the example dir.

Injector is an interface so you can accept an Injector in your library without
directly depending on a package that uses reflect. Just copy the definition of
the `Injector` interface into your library and your users can pass any
compatible implementation (like this one) to use it. Keep your dependency tree
clean, and still give your users injection!

Under MIT license.
*/
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
// args are the values injected into functions passed to Inject.
// There can only be one value of a given type per Injector.
func New(args ...interface{}) (Injector, error) {
	n := needle{}
	for _, v := range args {
		typ, val := reflect.TypeOf(v), reflect.ValueOf(v)
		if old, ok := n[typ]; ok {
			oi, vi := old.Interface(), val.Interface()
			return nil, fmt.Errorf("cannot inject two values of the same type. first: %#v (%T), second: %#v (%T)", oi, oi, vi, vi)
		}
		n[typ] = val
	}
	return &n, nil
}

// Must can wrap Inject and panics if err is not nil
func Must(i interface{}, err error) interface{} {
	if err != nil {
		panic(err)
	}
	return i
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
