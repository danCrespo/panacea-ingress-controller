package routing

import (
	"crypto/tls"
	"fmt"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"sync"

	networkingv1 "k8s.io/api/networking/v1"
)

type RoutingTable interface {
	UpdateFromIngresses(ing []*networkingv1.Ingress, ingressClass string)
	Match(host, reqPath string) *Route
	GetRoutes(ingressKey string) []*Route
	SetRoutes(ingressKey string, routes []*Route)
	DeleteRoutes(ingressKey string)
	ListAllRoutes() map[string][]*Route
	Clear()
}

type Route struct {
	Path     string
	PathType string
	Backend  *url.URL
	Proxy    *httputil.ReverseProxy
}

type routingTable struct {
	mu   sync.RWMutex
	data map[string][]*Route // Keyed by namespace/name of the IngressClass
}

func New() RoutingTable {
	return newRoutingTable()
}

func newRoutingTable() *routingTable {
	return &routingTable{
		data: make(map[string][]*Route),
	}
}

func (rt *routingTable) UpdateFromIngresses(ing []*networkingv1.Ingress, ingressClass string) {
	newData := make(map[string][]*Route)

	for _, ingress := range ing {
		if ingress.Spec.IngressClassName == nil || *ingress.Spec.IngressClassName != ingressClass {
			continue
		}

		for _, rule := range ingress.Spec.Rules {
			if rule.Host == "" || rule.HTTP == nil {
				continue
			}

			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service == nil {
					continue
				}

				name := path.Backend.Service.Name
				port := int32(0)
				if path.Backend.Service.Port.Number != 0 {
					port = path.Backend.Service.Port.Number
				}
				if port == 0 {
					continue
				}

				backendUrl := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", name, ingress.Namespace, port)
				u, err := url.Parse(backendUrl)
				if err != nil {
					continue
				}

				proxy := httputil.NewSingleHostReverseProxy(u)
				proxy.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}

				route := &Route{
					Path:     path.Path,
					PathType: string(*path.PathType),
					Backend:  u,
					Proxy:    proxy,
				}
				newData[rule.Host] = append(newData[rule.Host], route)
			}
		}
	}

	for h := range newData {
		sort.Slice(newData[h], func(i, j int) bool {
			return len(newData[h][i].Path) > len(newData[h][j].Path)
		})
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.data = newData
}

func (rt *routingTable) Match(host, reqPath string) *Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	routes, exists := rt.data[host]
	if !exists {
		return nil
	}

	for _, route := range routes {
		switch route.PathType {
		case string(networkingv1.PathTypeExact):
			if reqPath == route.Path {
				return route
			}
		case string(networkingv1.PathTypePrefix):
			if strings.HasPrefix(reqPath, route.Path) {
				return route
			}
		default:
			if strings.HasPrefix(reqPath, route.Path) {
				return route
			}
		}
	}

	return nil
}

func (rt *routingTable) GetRoutes(ingressKey string) []*Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.data[ingressKey]
}

func (rt *routingTable) SetRoutes(ingressKey string, routes []*Route) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.data[ingressKey] = routes
}

func (rt *routingTable) DeleteRoutes(ingressKey string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.data, ingressKey)
}

func (rt *routingTable) ListAllRoutes() map[string][]*Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	// Create a copy to avoid external modification
	copy := make(map[string][]*Route)
	maps.Copy(copy, rt.data)
	return copy
}

func (rt *routingTable) Clear() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.data = make(map[string][]*Route)
}
