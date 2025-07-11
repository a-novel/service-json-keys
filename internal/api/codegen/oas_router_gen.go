// Code generated by ogen, DO NOT EDIT.

package codegen

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ogen-go/ogen/uri"
)

func (s *Server) cutPrefix(path string) (string, bool) {
	prefix := s.cfg.Prefix
	if prefix == "" {
		return path, true
	}
	if !strings.HasPrefix(path, prefix) {
		// Prefix doesn't match.
		return "", false
	}
	// Cut prefix from the path.
	return strings.TrimPrefix(path, prefix), true
}

// ServeHTTP serves http request as defined by OpenAPI v3 specification,
// calling handler that matches the path or returning not found error.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	elem := r.URL.Path
	elemIsEscaped := false
	if rawPath := r.URL.RawPath; rawPath != "" {
		if normalized, ok := uri.NormalizeEscapedPath(rawPath); ok {
			elem = normalized
			elemIsEscaped = strings.ContainsRune(elem, '%')
		}
	}

	elem, ok := s.cutPrefix(elem)
	if !ok || len(elem) == 0 {
		s.notFound(w, r)
		return
	}

	// Static code generated router with unwrapped path search.
	switch {
	default:
		if len(elem) == 0 {
			break
		}
		switch elem[0] {
		case '/': // Prefix: "/"

			if l := len("/"); len(elem) >= l && elem[0:l] == "/" {
				elem = elem[l:]
			} else {
				break
			}

			if len(elem) == 0 {
				break
			}
			switch elem[0] {
			case 'h': // Prefix: "healthcheck"

				if l := len("healthcheck"); len(elem) >= l && elem[0:l] == "healthcheck" {
					elem = elem[l:]
				} else {
					break
				}

				if len(elem) == 0 {
					// Leaf node.
					switch r.Method {
					case "GET":
						s.handleHealthcheckRequest([0]string{}, elemIsEscaped, w, r)
					default:
						s.notAllowed(w, r, "GET")
					}

					return
				}

			case 'p': // Prefix: "p"

				if l := len("p"); len(elem) >= l && elem[0:l] == "p" {
					elem = elem[l:]
				} else {
					break
				}

				if len(elem) == 0 {
					break
				}
				switch elem[0] {
				case 'a': // Prefix: "ayload/sign"

					if l := len("ayload/sign"); len(elem) >= l && elem[0:l] == "ayload/sign" {
						elem = elem[l:]
					} else {
						break
					}

					if len(elem) == 0 {
						// Leaf node.
						switch r.Method {
						case "POST":
							s.handleSignClaimsRequest([0]string{}, elemIsEscaped, w, r)
						default:
							s.notAllowed(w, r, "POST")
						}

						return
					}

				case 'i': // Prefix: "ing"

					if l := len("ing"); len(elem) >= l && elem[0:l] == "ing" {
						elem = elem[l:]
					} else {
						break
					}

					if len(elem) == 0 {
						// Leaf node.
						switch r.Method {
						case "GET":
							s.handlePingRequest([0]string{}, elemIsEscaped, w, r)
						default:
							s.notAllowed(w, r, "GET")
						}

						return
					}

				case 'u': // Prefix: "ublic-keys"

					if l := len("ublic-keys"); len(elem) >= l && elem[0:l] == "ublic-keys" {
						elem = elem[l:]
					} else {
						break
					}

					if len(elem) == 0 {
						switch r.Method {
						case "GET":
							s.handleGetPublicKeyRequest([0]string{}, elemIsEscaped, w, r)
						default:
							s.notAllowed(w, r, "GET")
						}

						return
					}
					switch elem[0] {
					case '/': // Prefix: "/list"

						if l := len("/list"); len(elem) >= l && elem[0:l] == "/list" {
							elem = elem[l:]
						} else {
							break
						}

						if len(elem) == 0 {
							// Leaf node.
							switch r.Method {
							case "GET":
								s.handleListPublicKeysRequest([0]string{}, elemIsEscaped, w, r)
							default:
								s.notAllowed(w, r, "GET")
							}

							return
						}

					}

				}

			}

		}
	}
	s.notFound(w, r)
}

// Route is route object.
type Route struct {
	name        string
	summary     string
	operationID string
	pathPattern string
	count       int
	args        [0]string
}

// Name returns ogen operation name.
//
// It is guaranteed to be unique and not empty.
func (r Route) Name() string {
	return r.name
}

// Summary returns OpenAPI summary.
func (r Route) Summary() string {
	return r.summary
}

// OperationID returns OpenAPI operationId.
func (r Route) OperationID() string {
	return r.operationID
}

// PathPattern returns OpenAPI path.
func (r Route) PathPattern() string {
	return r.pathPattern
}

// Args returns parsed arguments.
func (r Route) Args() []string {
	return r.args[:r.count]
}

// FindRoute finds Route for given method and path.
//
// Note: this method does not unescape path or handle reserved characters in path properly. Use FindPath instead.
func (s *Server) FindRoute(method, path string) (Route, bool) {
	return s.FindPath(method, &url.URL{Path: path})
}

// FindPath finds Route for given method and URL.
func (s *Server) FindPath(method string, u *url.URL) (r Route, _ bool) {
	var (
		elem = u.Path
		args = r.args
	)
	if rawPath := u.RawPath; rawPath != "" {
		if normalized, ok := uri.NormalizeEscapedPath(rawPath); ok {
			elem = normalized
		}
		defer func() {
			for i, arg := range r.args[:r.count] {
				if unescaped, err := url.PathUnescape(arg); err == nil {
					r.args[i] = unescaped
				}
			}
		}()
	}

	elem, ok := s.cutPrefix(elem)
	if !ok {
		return r, false
	}

	// Static code generated router with unwrapped path search.
	switch {
	default:
		if len(elem) == 0 {
			break
		}
		switch elem[0] {
		case '/': // Prefix: "/"

			if l := len("/"); len(elem) >= l && elem[0:l] == "/" {
				elem = elem[l:]
			} else {
				break
			}

			if len(elem) == 0 {
				break
			}
			switch elem[0] {
			case 'h': // Prefix: "healthcheck"

				if l := len("healthcheck"); len(elem) >= l && elem[0:l] == "healthcheck" {
					elem = elem[l:]
				} else {
					break
				}

				if len(elem) == 0 {
					// Leaf node.
					switch method {
					case "GET":
						r.name = HealthcheckOperation
						r.summary = "Check the health of the service."
						r.operationID = "healthcheck"
						r.pathPattern = "/healthcheck"
						r.args = args
						r.count = 0
						return r, true
					default:
						return
					}
				}

			case 'p': // Prefix: "p"

				if l := len("p"); len(elem) >= l && elem[0:l] == "p" {
					elem = elem[l:]
				} else {
					break
				}

				if len(elem) == 0 {
					break
				}
				switch elem[0] {
				case 'a': // Prefix: "ayload/sign"

					if l := len("ayload/sign"); len(elem) >= l && elem[0:l] == "ayload/sign" {
						elem = elem[l:]
					} else {
						break
					}

					if len(elem) == 0 {
						// Leaf node.
						switch method {
						case "POST":
							r.name = SignClaimsOperation
							r.summary = "Sign a payload using the configuration for the target usage."
							r.operationID = "signClaims"
							r.pathPattern = "/payload/sign"
							r.args = args
							r.count = 0
							return r, true
						default:
							return
						}
					}

				case 'i': // Prefix: "ing"

					if l := len("ing"); len(elem) >= l && elem[0:l] == "ing" {
						elem = elem[l:]
					} else {
						break
					}

					if len(elem) == 0 {
						// Leaf node.
						switch method {
						case "GET":
							r.name = PingOperation
							r.summary = "Check the status of the service."
							r.operationID = "ping"
							r.pathPattern = "/ping"
							r.args = args
							r.count = 0
							return r, true
						default:
							return
						}
					}

				case 'u': // Prefix: "ublic-keys"

					if l := len("ublic-keys"); len(elem) >= l && elem[0:l] == "ublic-keys" {
						elem = elem[l:]
					} else {
						break
					}

					if len(elem) == 0 {
						switch method {
						case "GET":
							r.name = GetPublicKeyOperation
							r.summary = "Get the public keys used for JSON Web Algorithms."
							r.operationID = "getPublicKey"
							r.pathPattern = "/public-keys"
							r.args = args
							r.count = 0
							return r, true
						default:
							return
						}
					}
					switch elem[0] {
					case '/': // Prefix: "/list"

						if l := len("/list"); len(elem) >= l && elem[0:l] == "/list" {
							elem = elem[l:]
						} else {
							break
						}

						if len(elem) == 0 {
							// Leaf node.
							switch method {
							case "GET":
								r.name = ListPublicKeysOperation
								r.summary = "List all public keys used for JSON Web Algorithms."
								r.operationID = "listPublicKeys"
								r.pathPattern = "/public-keys/list"
								r.args = args
								r.count = 0
								return r, true
							default:
								return
							}
						}

					}

				}

			}

		}
	}
	return r, false
}
