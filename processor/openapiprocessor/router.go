// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Base implementation taken from https://github.com/getkin/kin-openapi/blob/master/routers/gorillamux/router.go

package openapiprocessor

import (
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/gorilla/mux"
)

var _ routers.Router = &Router{}

// Router helps link http.Request.s and an OpenAPIv3 spec
type Router struct {
	muxes  []routeMux
	routes []*routers.Route
}

type varsf func(vars map[string]string)

type routeMux struct {
	muxRoute    *mux.Route
	varsUpdater varsf
}

type srv struct {
	schemes     []string
	host, base  string
	server      *openapi3.Server
	varsUpdater varsf
}

const (
	serviceNameExtensionAttribute = "x-service-name"
)

var singleVariableMatcher = regexp.MustCompile(`^\{([^{}]+)\}$`)

// TODO: Handle/HandlerFunc + ServeHTTP (When there is a match, the route variables can be retrieved calling mux.Vars(request))

// NewRouter creates a gorilla/mux router.
// Assumes spec is .Validate()d
// Note that a variable for the port number MUST have a default value and only this value will match as the port (see issue #367).
func addRoutersFromAPI(doc *openapi3.T, existingRouters *map[string]*Router, serviceHostMapping *map[string]string, allowHTTPAndHTTPS bool) error {
	servers, err := makeServers(doc.Servers)
	if err != nil {
		return err
	}

	muxRouter := mux.NewRouter().UseEncodedPath()
	for _, path := range doc.Paths.InMatchingOrder() {
		pathItem := doc.Paths.Value(path)
		if len(pathItem.Servers) > 0 {
			if servers, err = makeServers(pathItem.Servers); err != nil {
				return err
			}
		}

		operations := pathItem.Operations()
		methods := make([]string, 0, len(operations))
		for method := range operations {
			methods = append(methods, method)
		}
		sort.Strings(methods)

		for _, s := range servers {
			muxRoute := muxRouter.Path(s.base + path).Methods(methods...)
			if allowHTTPAndHTTPS {
				muxRoute.Schemes("http", "https")
			} else {
				if schemes := s.schemes; len(schemes) != 0 {
					muxRoute.Schemes(schemes...)
				}
			}

			host := s.host
			if host != "" {
				muxRoute.Host(host)
				serviceNameExtension, ok := s.server.Extensions[serviceNameExtensionAttribute]
				if ok {
					if str, ok := serviceNameExtension.(string); ok {
						(*serviceHostMapping)[str] = host
					}
				}
			}

			if err := muxRoute.GetError(); err != nil {
				return err
			}

			// Find router based on host
			r := (*existingRouters)[host]
			if r == nil {
				r = &Router{}
				(*existingRouters)[host] = r
			}

			r.muxes = append(r.muxes, routeMux{
				muxRoute:    muxRoute,
				varsUpdater: s.varsUpdater,
			})
			r.routes = append(r.routes, &routers.Route{
				Spec:      doc,
				Server:    s.server,
				Path:      path,
				PathItem:  pathItem,
				Method:    "",
				Operation: nil,
			})
		}
	}

	return nil
}

// FindRoute extracts the route and parameters of an http.Request
func (r *Router) FindRoute(req *http.Request) (*routers.Route, map[string]string, error) {
	for i, m := range r.muxes {
		var match mux.RouteMatch
		if m.muxRoute.Match(req, &match) {
			vars := match.Vars
			if f := m.varsUpdater; f != nil {
				f(vars)
			}
			route := *r.routes[i]
			route.Method = req.Method
			route.Operation = route.Spec.Paths.Value(route.Path).GetOperation(route.Method)
			return &route, vars, nil
		}
		switch match.MatchErr { //nolint
		case nil:
		case mux.ErrMethodMismatch:
			return nil, nil, routers.ErrMethodNotAllowed
		default:
		}
	}
	return nil, nil, routers.ErrPathNotFound
}

func makeServers(in openapi3.Servers) ([]srv, error) {
	servers := make([]srv, 0, len(in))
	for _, server := range in {
		serverURL := server.URL
		if submatch := singleVariableMatcher.FindStringSubmatch(serverURL); submatch != nil {
			sVar := submatch[1]
			sVal := server.Variables[sVar].Default
			serverURL = strings.ReplaceAll(serverURL, "{"+sVar+"}", sVal)
			var varsUpdater varsf
			if lhs := strings.TrimSuffix(serverURL, server.Variables[sVar].Default); lhs != "" {
				varsUpdater = func(vars map[string]string) { vars[sVar] = lhs }
			}
			svr, err := newSrv(serverURL, server, varsUpdater)
			if err != nil {
				return nil, err
			}

			servers = append(servers, svr)
			continue
		}

		// If a variable represents the port "http://domain.tld:{port}/bla"
		// then url.Parse() cannot parse "http://domain.tld:`bEncode({port})`/bla"
		// and mux is not able to set the {port} variable
		// So we just use the default value for this variable.
		// See https://github.com/getkin/kin-openapi/issues/367
		var varsUpdater varsf
		if lhs := strings.Index(serverURL, ":{"); lhs > 0 {
			rest := serverURL[lhs+len(":{"):]
			rhs := strings.Index(rest, "}")
			portVariable := rest[:rhs+1]
			portValue := server.Variables[portVariable].Default
			serverURL = strings.ReplaceAll(serverURL, "{"+portVariable+"}", portValue)
			varsUpdater = func(vars map[string]string) {
				vars[portVariable] = portValue
			}
		}

		svr, err := newSrv(serverURL, server, varsUpdater)
		if err != nil {
			return nil, err
		}
		servers = append(servers, svr)
	}
	if len(servers) == 0 {
		servers = append(servers, srv{})
	}

	return servers, nil
}

func newSrv(serverURL string, server *openapi3.Server, varsUpdater varsf) (srv, error) {
	var schemes []string
	if strings.Contains(serverURL, "://") {
		scheme0 := strings.Split(serverURL, "://")[0]
		schemes = permutePart(scheme0, server)
		serverURL = strings.Replace(serverURL, scheme0+"://", schemes[0]+"://", 1)
	}

	u, err := url.Parse(bEncode(serverURL))
	if err != nil {
		return srv{}, err
	}
	path := bDecode(u.EscapedPath())
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	svr := srv{
		host:        bDecode(u.Host), // u.Hostname()?
		base:        path,
		schemes:     schemes, // scheme: []string{scheme0}, TODO: https://github.com/gorilla/mux/issues/624
		server:      server,
		varsUpdater: varsUpdater,
	}
	return svr, nil
}

// Magic strings that temporarily replace "{}" so net/url.Parse() works
var blURL, brURL = strings.Repeat("-", 50), strings.Repeat("_", 50)

func bEncode(s string) string {
	s = strings.ReplaceAll(s, "{", blURL)
	s = strings.ReplaceAll(s, "}", brURL)
	return s
}
func bDecode(s string) string {
	s = strings.ReplaceAll(s, blURL, "{")
	s = strings.ReplaceAll(s, brURL, "}")
	return s
}

func permutePart(part0 string, srv *openapi3.Server) []string {
	type mapAndSlice struct {
		m map[string]struct{}
		s []string
	}
	var2val := make(map[string]mapAndSlice)
	max := 0
	for name0, v := range srv.Variables {
		name := "{" + name0 + "}"
		if !strings.Contains(part0, name) {
			continue
		}
		m := map[string]struct{}{v.Default: {}}
		for _, value := range v.Enum {
			m[value] = struct{}{}
		}
		if l := len(m); l > max {
			max = l
		}
		s := make([]string, 0, len(m))
		for value := range m {
			s = append(s, value)
		}
		var2val[name] = mapAndSlice{m: m, s: s}
	}
	if len(var2val) == 0 {
		return []string{part0}
	}

	partsMap := make(map[string]struct{}, max*len(var2val))
	for i := 0; i < max; i++ {
		part := part0
		for name, mas := range var2val {
			part = strings.ReplaceAll(part, name, mas.s[i%len(mas.s)])
		}
		partsMap[part] = struct{}{}
	}
	parts := make([]string, 0, len(partsMap))
	for part := range partsMap {
		parts = append(parts, part)
	}
	sort.Strings(parts)
	return parts
}
