package wrmatch

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatterInvalidInput(t *testing.T) {
	value := struct{}{}
	router := NewPattern()

	require.Panics(t, func() {
		router.Add("", value)
	})
	require.Panics(t, func() {
		router.Add("noSlashRoot", value)
	})
	require.Panics(t, func() {
		router.Add("/", nil)
	})
}

func TestPatternMach(t *testing.T) {
	var matched bool
	var matchedRoutePath string
	var value interface{}

	router := NewPattern()
	router.Add("/GET", "get")
	router.Add("/POST", "post")
	router.Add("/PUT", "put")
	router.Add("/PATCH", "patch")
	router.Add("/DELETE", "delete")
	router.Add("/HANDLE", "handle")
	router.Add("/ANY", "any")

	value, matchedRoutePath, matched = router.MatchURL("/GET")
	require.True(t, matched)
	require.Empty(t, matchedRoutePath)
	require.Equal(t, value, "get")

	value, matchedRoutePath, matched = router.MatchURL("/notfound")
	require.False(t, matched)
	require.Empty(t, matchedRoutePath)
	require.Nil(t, value)
}

func TestPatternMatchRedirectTrailingSlash(t *testing.T) {
	var matched bool

	router := NewPattern()
	router.Add("/GET", "get")
	router.Add("/POST/", "get")

	_, _, matched = router.MatchURL("/GET/")
	require.True(t, matched)

	_, _, matched = router.MatchURL("/POST")
	require.True(t, matched)
}

func TestPatternRedirect(t *testing.T) {
	router := NewPattern()
	router.Add("/path", "/path")
	router.Add("/dir/", "/dir/")
	router.Add("/", "/")

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
			v, matchedRoutePath, matched := router.MatchURL(tt.path)
			assert.True(t, matched)
			assert.Equal(t, tt.location, v)
			assert.Empty(t, matchedRoutePath)
		})
	}
}

func TestPatternDisableRedirect(t *testing.T) {
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
			v, matchedRoutePath, matched := router.Match(http.MethodGet, tt.path)
			assert.False(t, matched)
			assert.Nil(t, v)
			assert.Empty(t, matchedRoutePath)
		})
	}
}

func TestPatternMatchedRoutePath(t *testing.T) {
	router := NewPattern(WithSaveMatchedRoutePath())
	router.Add("/user/:name", "handle1")
	router.Add("/user/:name/details", "handle2")
	router.Add("/", "handle3")

	v, matchedRoutePath, matched := router.MatchURL("/user/gopher")
	require.True(t, matched)
	require.Equal(t, "/user/:name", matchedRoutePath)
	require.Equal(t, "handle1", v)

	v, matchedRoutePath, matched = router.MatchURL("/user/gopher/details")
	require.True(t, matched)
	require.Equal(t, "/user/:name/details", matchedRoutePath)
	require.Equal(t, "handle2", v)

	v, matchedRoutePath, matched = router.MatchURL("/")
	require.True(t, matched)
	require.Equal(t, "/", matchedRoutePath)
	require.Equal(t, "handle3", v)
}

func TestPatternEnableSaveMatchedRouterPathPanicShouldNotHappen(t *testing.T) {
	router := NewPattern()
	router.Add("/user/:name", "handle1")
	router.saveMatchedRoutePath = true
	require.Panics(t, func() {
		router.MatchURL("/user/gopher")
	})
}
