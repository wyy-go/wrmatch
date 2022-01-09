package wrmatch

// Pattern is a via configurable url pattern
type Pattern struct {
	root *node
	Options
}

// NewPattern returns a new initialized Router.
// Path auto-correction, including trailing slashes, is enabled by default.
func NewPattern(opts ...Option) *Pattern {
	r := &Pattern{
		root: new(node),
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

// Add registers a new request value with the given path.
func (r *Pattern) Add(path string, value interface{}) *Pattern {
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}
	if value == nil {
		panic("value must not be nil")
	}

	if r.saveMatchedRoutePath {
		value = matchValue{path, value}
	}
	r.root.addRoute(path, value)
	return r
}

// MatchURL match method and path return matched or not and store value and matched route path if Router.saveMatchedRoutePath enabled.
func (r *Pattern) MatchURL(path string) (interface{}, string, bool) {
	value, _, tsr := r.root.getValue(path, nil)
	if value != nil {
		if r.saveMatchedRoutePath {
			vv, ok := value.(matchValue)
			if !ok {
				panic("enabled saveMatchedRoutePath, value should be struct(matchValue)")
			}
			return vv.Value, vv.matchedPath, true
		}
		return value, "", true
	}
	if path != "/" {
		if tsr && r.redirectTrailingSlash {
			if len(path) > 1 && path[len(path)-1] == '/' {
				path = path[:len(path)-1]
			} else {
				path += "/"
			}
			return r.MatchURL(path)
		}
		// Try to fix the request path
		if r.redirectFixedPath {
			fixedPath, found := r.root.findCaseInsensitivePath(CleanPath(path), r.redirectTrailingSlash)
			if found {
				return r.MatchURL(fixedPath)
			}
		}
	}
	return nil, "", false
}
