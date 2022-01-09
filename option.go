package wrmatch

// Options Router and Pattern option
type Options struct {
	// If enabled, get the matched route path. that were
	// registered when this option was enabled.
	saveMatchedRoutePath bool

	// Enables automatic redirection if the current route can't be matched but a
	// value for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 308 for all other request methods.
	redirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// value is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a value can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 308 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// redirectTrailingSlash is independent of this option.
	redirectFixedPath bool
}

// Option for Router, Pattern
type Option func(*Options)

// WithDisableRedirectTrailingSlash disable automatic redirection if the current route can't be matched but a
// value for the path with (without) the trailing slash exists
// Default: enabled
func WithDisableRedirectTrailingSlash() Option {
	return func(r *Options) {
		r.redirectTrailingSlash = false
	}
}

// WithDisableRedirectFixedPath diable the router tries to fix the current request path, if no
// value is registered for it.
// Default: enabled
func WithDisableRedirectFixedPath() Option {
	return func(r *Options) {
		r.redirectFixedPath = false
	}
}

// WithSaveMatchedRoutePath adds the matched route path onto the Params.
// The matched route path is only added to Params of routes that were
// registered when this option was enabled.
// Default: disable
func WithSaveMatchedRoutePath() Option {
	return func(r *Options) {
		r.saveMatchedRoutePath = true
	}
}
