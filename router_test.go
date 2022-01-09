// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package wrmatch

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParams(t *testing.T) {
	ps := Params{
		Param{"param1", "value1"},
		Param{"param2", "value2"},
		Param{"param3", "value3"},
	}
	for i := range ps {
		assert.Equal(t, ps[i].Value, ps.Param(ps[i].Key))
	}
	require.Empty(t, ps.Param("noKey"))
}

func TestRouterInvalidInput(t *testing.T) {
	value := struct{}{}
	router := New()

	require.Panics(t, func() {
		router.Add("", "/", value)
	})
	require.Panics(t, func() {
		router.GET("", value)
	})
	require.Panics(t, func() {
		router.GET("noSlashRoot", value)
	})
	require.Panics(t, func() {
		router.GET("/", nil)
	})
}

func TestRouterParam(t *testing.T) {
	router := New()
	router.GET("/user/:name", "aaa")

	value, ps, matched := router.Match(http.MethodGet, "/user/gopher")
	require.True(t, matched)
	require.Equal(t, ps, Params{Param{"name", "gopher"}})
	require.Equal(t, value, "aaa")
}

func TestRouterMach(t *testing.T) {
	var matched bool
	var params Params
	var value interface{}

	router := New()
	router.GET("/GET", "get")
	router.HEAD("/GET", "head")
	router.OPTIONS("/GET", "options")
	router.POST("/POST", "post")
	router.PUT("/PUT", "put")
	router.PATCH("/PATCH", "patch")
	router.DELETE("/DELETE", "delete")
	router.Add(http.MethodGet, "/HANDLE", "handle")
	router.Any("/ANY", "any")

	value, params, matched = router.Match(http.MethodGet, "/GET")
	require.True(t, matched)
	require.Nil(t, params)
	require.Equal(t, value, "get")

	value, params, matched = router.Match(http.MethodHead, "/GET")
	require.True(t, matched)
	require.Nil(t, params)
	require.Equal(t, value, "head")

	value, params, matched = router.Match(http.MethodOptions, "/GET")
	require.True(t, matched)
	require.Nil(t, params)
	require.Equal(t, value, "options")

	value, params, matched = router.Match(http.MethodPost, "/POST")
	require.True(t, matched)
	require.Nil(t, params)
	require.Equal(t, value, "post")

	value, params, matched = router.Match(http.MethodPut, "/PUT")
	require.True(t, matched)
	require.Nil(t, params)
	require.Equal(t, value, "put")

	value, params, matched = router.Match(http.MethodPatch, "/PATCH")
	require.True(t, matched)
	require.Nil(t, params)
	require.Equal(t, value, "patch")

	value, params, matched = router.Match(http.MethodDelete, "/DELETE")
	require.True(t, matched)
	require.Nil(t, params)
	require.Equal(t, value, "delete")

	value, params, matched = router.Match(http.MethodGet, "/HANDLE")
	require.True(t, matched)
	require.Nil(t, params)
	require.Equal(t, value, "handle")

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodConnect, http.MethodTrace}
	for _, method := range methods {
		value, params, matched = router.Match(method, "/ANY")
		require.True(t, matched)
		require.Nil(t, params)
		require.Equal(t, value, "any")
	}

	value, params, matched = router.Match(http.MethodGet, "/notfound")
	require.False(t, matched)
	require.Nil(t, params)
	require.Nil(t, value)
}

func TestRouterMatchRedirectTrailingSlash(t *testing.T) {
	var matched bool

	router := New()
	router.GET("/GET", "get")
	router.GET("/POST/", "get")

	_, _, matched = router.Match(http.MethodGet, "/GET/")
	require.True(t, matched)

	_, _, matched = router.Match(http.MethodGet, "/POST")
	require.True(t, matched)
}

func TestRouterRedirect(t *testing.T) {
	router := New()
	router.GET("/path", "/path")
	router.GET("/dir/", "/dir/")
	router.GET("/", "/")

	tests := []struct {
		name     string
		path     string
		location string
	}{
		{"", "/path/", "/path"},   // TSR -/
		{"", "/dir", "/dir/"},     // TSR +/
		{"", "", "/"},             // TSR +/
		{"", "/PATH", "/path"},    // Fixed Case
		{"", "/DIR/", "/dir/"},    // Fixed Case
		{"", "/PATH/", "/path"},   // Fixed Case -/
		{"", "/DIR", "/dir/"},     // Fixed Case +/
		{"", "/../path", "/path"}, // CleanPath
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, params, matched := router.Match(http.MethodGet, tt.path)
			assert.True(t, matched)
			assert.Equal(t, tt.location, v)
			assert.Nil(t, params)
		})
	}
}

func TestRouterDisableRedirect(t *testing.T) {
	router := New(WithDisableRedirectFixedPath(), WithDisableRedirectTrailingSlash())
	router.GET("/path", "/path")
	router.GET("/dir/", "/dir/")
	router.GET("/", "/")

	tests := []struct {
		name     string
		path     string
		location string
	}{
		{"", "/path/", "/path"},   // TSR -/
		{"", "/dir", "/dir/"},     // TSR +/
		{"", "", "/"},             // TSR +/
		{"", "/PATH", "/path"},    // Fixed Case
		{"", "/DIR/", "/dir/"},    // Fixed Case
		{"", "/PATH/", "/path"},   // Fixed Case -/
		{"", "/DIR", "/dir/"},     // Fixed Case +/
		{"", "/../path", "/path"}, // CleanPath
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, params, matched := router.Match(http.MethodGet, tt.path)
			assert.False(t, matched)
			assert.Nil(t, v)
			assert.Nil(t, params)
		})
	}
}

func TestRouterLookup(t *testing.T) {
	wantHandle := struct{}{}
	wantParams := Params{Param{"name", "gopher"}}

	router := New()

	// try empty router first
	value, _, tsr := router.Lookup(http.MethodGet, "/nope")
	require.Nil(t, value)
	require.False(t, tsr)

	// insert route and try again
	router.GET("/user/:name", wantHandle)
	value, params, tsr := router.Lookup(http.MethodGet, "/user/gopher")
	require.NotNil(t, value)
	require.False(t, tsr)
	require.Equal(t, params, wantParams)
	require.True(t, reflect.DeepEqual(params, wantParams))

	// route without param
	router.GET("/user", wantHandle)
	value, params, _ = router.Lookup(http.MethodGet, "/user")
	require.NotNil(t, value)
	require.False(t, tsr)
	require.Nil(t, params)

	value, _, tsr = router.Lookup(http.MethodGet, "/user/gopher/")
	require.Nil(t, value)
	require.True(t, tsr)

	value, _, tsr = router.Lookup(http.MethodGet, "/nope")
	require.Nil(t, value)
	require.False(t, tsr)
}

func TestRouterMatchedRoutePath(t *testing.T) {
	router := New(WithSaveMatchedRoutePath())
	router.Add(http.MethodGet, "/user/:name", "handle1")
	router.Add(http.MethodGet, "/user/:name/details", "handle2")
	router.Add(http.MethodGet, "/", "handle3")

	v, params, matched := router.Match(http.MethodGet, "/user/gopher")
	require.True(t, matched)
	require.Equal(t, "/user/:name", params.MatchedRoutePath())
	require.Equal(t, Params{Param{"name", "gopher"}, {MatchedRoutePathParam, "/user/:name"}}, params)
	require.Equal(t, "handle1", v)

	v, params, matched = router.Match(http.MethodGet, "/user/gopher/details")
	require.True(t, matched)
	require.Equal(t, "/user/:name/details", params.MatchedRoutePath())
	require.Equal(t, Params{Param{"name", "gopher"}, {MatchedRoutePathParam, "/user/:name/details"}}, params)
	require.Equal(t, "handle2", v)

	v, params, matched = router.Match(http.MethodGet, "/")
	require.True(t, matched)
	require.Equal(t, "/", params.MatchedRoutePath())
	require.Equal(t, Params{{MatchedRoutePathParam, "/"}}, params)
	require.Equal(t, "handle3", v)
}

func TestRouterEnableSaveMatchedRouterPathPanicShouldNotHappen(t *testing.T) {
	router := New()
	router.Add(http.MethodGet, "/user/:name", "handle1")
	router.saveMatchedRoutePath = true
	require.Panics(t, func() {
		router.Match(http.MethodGet, "/user/gopher")
	})
}

func TestRouterMatchURL(t *testing.T) {
	router := New()

	router.GET("/GET/:name", "get")

	v, matchPath, matched := router.MatchURL(http.MethodGet, "/GET/myName")
	require.True(t, matched)
	require.Equal(t, "", matchPath)
	require.Equal(t, "get", v)
}

func TestRouterMatchURLEnableSaveMatchedRouterPath(t *testing.T) {
	router := New(WithSaveMatchedRoutePath())

	router.GET("/GET/:name", "get")

	v, matchPath, matched := router.MatchURL(http.MethodGet, "/GET/myName")
	require.True(t, matched)
	require.Equal(t, "/GET/:name", matchPath)
	require.Equal(t, "get", v)
}

func BenchmarkMatch(b *testing.B) {
	router := New()
	router.GET("/GET/:name", "get")
	for i := 0; i < b.N; i++ {
		router.Match(http.MethodGet, "/GET/myName")
	}
}

func BenchmarkMatchURL(b *testing.B) {
	router := New()
	router.GET("/GET/:name", "get")
	for i := 0; i < b.N; i++ {
		router.MatchURL(http.MethodGet, "/GET/myName")
	}
}
