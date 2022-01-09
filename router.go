// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// Package urlmatch is a trie match url.
//
// A trivial example is:
//
//  package main
//
//  import (
//      "github.com/things-go/urlmatch"
//      "log"
//  )
//
//  func main() {
//      router := urlmatch.New()
//      router.GET("/", "/")
//      router.GET("/hello/:name", "Hello")
//
//      v, _, matched := router.Match(http.MethodGet, "/")
//      if matched {
//          log.Println(v)
//      }
//      v, ps, matched := router.Match(http.MethodGet, "/hello/myname")
//      if matched {
//          log.Println(v)
//          log.Println(ps.Param("name"))
//      }
//  }
//
// The router matches the request method and the path.
// For the methods GET, POST, PUT, PATCH, DELETE and OPTIONS shortcut functions exist to
// register value, for all other methods router.Value can be used.
//
// The registered path, against which the router matches incoming requests, can
// contain two types of parameters:
//  Syntax    Type
//  :name     named parameter
//  *name     catch-all parameter
//
// Named parameters are dynamic path segments. They match anything until the
// next '/' or the path end:
//  Path: /blog/:category/:post
//
//  Requests:
//   /blog/go/request-routers            match: category="go", post="request-routers"
//   /blog/go/request-routers/           no match, but the router would redirect
//   /blog/go/                           no match
//   /blog/go/request-routers/comments   no match
//
// Catch-all parameters match anything until the path end, including the
// directory index (the '/' before the catch-all). Since they match anything
// until the end, catch-all parameters must always be the final path element.
//  Path: /files/*filepath
//
//  Requests:
//   /files/                             match: filepath="/"
//   /files/LICENSE                      match: filepath="/LICENSE"
//   /files/templates/article.html       match: filepath="/templates/article.html"
//   /files                              no match, but the router would redirect
//
// The value of parameters is saved as a slice of the Param struct, consisting
// each of a key and a value. The slice is passed to the Add func as a third
// parameter.
// There are two ways to retrieve the value of a parameter:
//  // by the name of the parameter
//  user := ps.Param("user") // defined by :user or *user
//
//  // by the index of the parameter. This way you can also get the name (key)
//  thirdKey   := ps[2].Key   // the name of the 3rd parameter
//  thirdValue := ps[2].Value // the value of the 3rd parameter
package wrmatch

import (
	"net/http"
)

// MatchedRoutePathParam is the Param name under which the path of the matched
// route is stored, if Router.saveMatchedRoutePath is set.
var MatchedRoutePathParam = "$matchedRoutePath"

// matchValue store matchedPath and real value when Router.saveMatchedRoutePath = true.
type matchValue struct {
	matchedPath string
	Value       interface{}
}

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// Param returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) Param(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

// MatchedRoutePath retrieves the path of the matched route.
// Router.saveMatchedRoutePath must have been enabled when the respective
// handler was added, otherwise this function always returns an empty string.
func (ps Params) MatchedRoutePath() string {
	return ps.Param(MatchedRoutePathParam)
}

// Router is a via configurable routes
type Router struct {
	trees map[string]*node

	paramsNew func() *Params
	maxParams uint16

	Options
}

// New returns a new initialized Router.
// Path auto-correction, including trailing slashes, is enabled by default.
func New(opts ...Option) *Router {
	r := &Router{
		Options: Options{
			redirectTrailingSlash: true,
			redirectFixedPath:     true,
		},
	}
	for _, opt := range opts {
		opt(&r.Options)
	}
	return r
}

// GET is a shortcut for router.Add(http.MethodGet, path, handle)
func (r *Router) GET(path string, value interface{}) *Router {
	return r.Add(http.MethodGet, path, value)
}

// HEAD is a shortcut for router.Add(http.MethodHead, path, value)
func (r *Router) HEAD(path string, value interface{}) *Router {
	return r.Add(http.MethodHead, path, value)
}

// OPTIONS is a shortcut for router.Add(http.MethodOptions, path, value)
func (r *Router) OPTIONS(path string, value interface{}) *Router {
	return r.Add(http.MethodOptions, path, value)
}

// POST is a shortcut for router.Add(http.MethodPost, path, value)
func (r *Router) POST(path string, value interface{}) *Router {
	return r.Add(http.MethodPost, path, value)
}

// PUT is a shortcut for router.Add(http.MethodPut, path, value)
func (r *Router) PUT(path string, value interface{}) *Router {
	return r.Add(http.MethodPut, path, value)
}

// PATCH is a shortcut for router.Add(http.MethodPatch, path, value)
func (r *Router) PATCH(path string, value interface{}) *Router {
	return r.Add(http.MethodPatch, path, value)
}

// DELETE is a shortcut for router.Add(http.MethodDelete, path, value)
func (r *Router) DELETE(path string, value interface{}) *Router {
	return r.Add(http.MethodDelete, path, value)
}

// Any registers a route that matches all the HTTP methods.
// GET, POST, PUT, PATCH, HEAD, OPTIONS, DELETE, CONNECT, TRACE.
func (r *Router) Any(path string, value interface{}) *Router {
	return r.Add(http.MethodGet, path, value).
		Add(http.MethodPost, path, value).
		Add(http.MethodPut, path, value).
		Add(http.MethodPatch, path, value).
		Add(http.MethodHead, path, value).
		Add(http.MethodOptions, path, value).
		Add(http.MethodDelete, path, value).
		Add(http.MethodConnect, path, value).
		Add(http.MethodTrace, path, value)
}

// Add registers a new request value with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) Add(method, path string, value interface{}) *Router {
	varsCount := uint16(0)

	if method == "" {
		panic("method must not be empty")
	}
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}
	if value == nil {
		panic("value must not be nil")
	}

	if r.saveMatchedRoutePath {
		value = matchValue{path, value}
		varsCount++
	}

	if r.trees == nil {
		r.trees = make(map[string]*node)
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root
	}

	root.addRoute(path, value)

	// Update maxParams
	if paramsCount := countParams(path); paramsCount+varsCount > r.maxParams {
		r.maxParams = paramsCount + varsCount
	}

	// Lazy-init paramsNew alloc func
	if r.paramsNew == nil && r.maxParams > 0 {
		r.paramsNew = func() *Params {
			ps := make(Params, 0, r.maxParams)
			return &ps
		}
	}
	return r
}

// Lookup allows the manual lookup of a method + path combo.
// This is e.g. useful to build a framework around this router.
// If the path was found, it returns the value function and the path parameter
// values. Otherwise the third return value indicates whether a redirection to
// the same path with an extra / without the trailing slash should be performed.
func (r *Router) Lookup(method, path string) (interface{}, Params, bool) {
	if root := r.trees[method]; root != nil {
		value, ps, tsr := root.getValue(path, r.paramsNew)
		if value == nil {
			return nil, nil, tsr
		}
		if ps == nil {
			return value, nil, tsr
		}
		return value, *ps, tsr
	}
	return nil, nil, false
}

// Match match method and path return matched or not and store value and url params.
func (r *Router) Match(method, path string) (interface{}, Params, bool) {
	return r.match(method, path, r.paramsNew)
}

// MatchURL match method and path return matched or not and store value and matched route path if Router.saveMatchedRoutePath enabled.
func (r *Router) MatchURL(method, path string) (interface{}, string, bool) {
	v, params, matched := r.match(method, path, nil)
	return v, params.MatchedRoutePath(), matched
}

// match match method and path return matched or not and store value and url params.
func (r *Router) match(method, path string, paramsNew func() *Params) (interface{}, Params, bool) {
	if root := r.trees[method]; root != nil {
		value, ps, tsr := root.getValue(path, paramsNew)
		if value != nil {
			if r.saveMatchedRoutePath {
				vv, ok := value.(matchValue)
				if !ok {
					panic("enabled saveMatchedRoutePath, value should be struct(matchValue)")
				}
				if ps == nil {
					return vv.Value, Params{Param{MatchedRoutePathParam, vv.matchedPath}}, true
				}
				*ps = append(*ps, Param{MatchedRoutePathParam, vv.matchedPath})
				return vv.Value, *ps, true
			}
			if ps == nil {
				return value, nil, true
			}
			return value, *ps, true
		}
		if method != http.MethodConnect && path != "/" {
			if tsr && r.redirectTrailingSlash {
				if len(path) > 1 && path[len(path)-1] == '/' {
					path = path[:len(path)-1]
				} else {
					path += "/"
				}
				return r.Match(method, path)
			}
			// Try to fix the request path
			if r.redirectFixedPath {
				fixedPath, found := root.findCaseInsensitivePath(CleanPath(path), r.redirectTrailingSlash)
				if found {
					return r.Match(method, fixedPath)
				}
			}
		}
	}
	return nil, nil, false
}
