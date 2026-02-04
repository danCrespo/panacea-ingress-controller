package routing

import (
	"crypto/tls"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/danCrespo/panacea-ingress-controller/config"
	"github.com/danCrespo/panacea-ingress-controller/kubeutils"
	"github.com/danCrespo/panacea-ingress-controller/logger"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

var (
	l = logger.NewLogger().WithValues("component", "routing")
)

type RoutingTable interface {
	UpdateFromIngresses(ing []*networkingv1.Ingress, ingressClass string)
	Match(host, reqPath string) *Route
	GetRoutes(ingressKey string) []*Route
	SetRoutes(ingressKey string, routes []*Route)
	DeleteRoutes(ingressKey string)
	ListAllRoutes() map[string][]*Route
	Clear()
	SetLogger(logger logr.Logger)
}

type Route struct {
	Path     string
	PathType string
	Backend  *url.URL
	Proxy    *httputil.ReverseProxy
}

type routingTable struct {
	mu        sync.RWMutex
	data      map[string][]*Route // Keyed by namespace/name of the IngressClass
	kubeutils kubeutils.IKubeutils
	config    config.Config
}

func New(cfg config.Config) RoutingTable {
	return newRoutingTable(cfg)
}

func newRoutingTable(cfg config.Config) *routingTable {
	return &routingTable{
		data:      make(map[string][]*Route),
		kubeutils: kubeutils.NewKubeutils(cfg),
		config:    cfg,
	}
}

func (rt *routingTable) SetLogger(logger logr.Logger) {
	l = logger
}

func (rt *routingTable) UpdateFromIngresses(_ingresses []*networkingv1.Ingress, ingressClass string) {
	newData := make(map[string][]*Route)
	var (
		ingresses []networkingv1.Ingress
	)

	l.Info("Updating routing table from ingresses", _ingresses[len(_ingresses)-1])
	domain, err := rt.kubeutils.GetClusterDomain()

	if err != nil {
		l.Info("Error getting cluster domain, defaulting to cluster.local", "error", err)
		domain = "cluster.local"
	}

	if domain == "" {
		domain = "cluster.local"
	}

	ingresses = func() []networkingv1.Ingress {
		filtered := make([]networkingv1.Ingress, len(_ingresses))
		if ingressClass != "" {
			for _, ing := range _ingresses {
				if ing.Spec.IngressClassName != nil && *ing.Spec.IngressClassName == ingressClass {
					if len(ing.Spec.Rules) == 0 || ing.Spec.Rules == nil {
						continue
					}
					filtered = append(filtered, *ing)
				}
			}
		}
		return filtered
	}()

	for _, ingress := range ingresses {

		l.Info("Processing ingress", "name", ingress.Name, "namespace", ingress.Namespace)

		for _, rule := range ingress.Spec.Rules {

			if rule.Host == "" || rule.HTTP == nil {
				continue
			}

			l.Info("Processing rule", "host", rule.Host, "ingress", ingress.Name, "namespace", ingress.Namespace)

			for _, path := range rule.HTTP.Paths {

				if path.Backend.Service == nil && path.Backend.Resource == nil {
					continue
				}

				if path.Backend.Resource != nil {
					resource := path.Backend.Resource
					clusterResource, err := rt.kubeutils.GetResource(ingress.Namespace, resource.Kind, resource.Name)
					if err != nil {
						l.Info("Skipping resource backend due to error", "resource", path.Backend.Resource.Name, "kind", path.Backend.Resource.Kind, "error", err)
						continue
					}

					castResource, ok := clusterResource.(corev1.Service)
					if !ok {
						l.Info("Skipping resource backend due to invalid type", "resource", path.Backend.Resource.Name, "kind", path.Backend.Resource.Kind)
						continue
					}
					l.Info("Resolved resource backend to service", "resource", path.Backend.Resource.Name, "kind", path.Backend.Resource.Kind, "service", castResource.Name, "port", castResource.Spec.Ports[0].Port)
					path.Backend.Service = &networkingv1.IngressServiceBackend{
						Name: castResource.Name,
						Port: networkingv1.ServiceBackendPort{
							Number: castResource.Spec.Ports[0].Port,
						},
					}
					continue
				}

				if path.Backend.Service == nil {
					continue
				}

				l.Info("Processing path", "path", path.Path, "service", path.Backend.Service.Name, "namespace", ingress.Namespace)

				svc := path.Backend.Service
				if svc.Name == "" {
					continue
				}
				name := svc.Name
				port := int32(0)

				if path.Backend.Service.Port.Number != 0 {
					port = path.Backend.Service.Port.Number
				} else if path.Backend.Service.Port.Name != "" {
					svcPort, err := rt.kubeutils.GetServicePortByName(ingress.Namespace, name, svc.Port.Name)
					if err != nil {
						l.Info("Skipping path due to error getting service port by name", "service", name, "portName", svc.Port.Name, "error", err)
						continue
					}
					port = svcPort
				}

				if port == 0 {
					continue
				}

				l.Info("Adding route", "host", rule.Host, "path", path.Path, "service", name, "namespace", ingress.Namespace, "port", port)
				backendUrl := fmt.Sprintf("http://%s.%s.svc.%s:%d", name, ingress.Namespace, domain, port)
				u, err := url.Parse(backendUrl)
				if err != nil {
					continue
				}

				protos := &http.Protocols{}
				protos.SetHTTP1(true)
				protos.SetHTTP2(true)

				// proxy := httputil.NewSingleHostReverseProxy(u)
				proxy := &httputil.ReverseProxy{
					Rewrite: func(req *httputil.ProxyRequest) {
						defer req.Out.Context()
						headers := req.In.Header
						mustSkip := false

						for key, values := range headers {
							if slices.ContainsFunc(values, regexp.MustCompile(`[%^\s<>\\\(\)\[\]]`).MatchString) {
								mustSkip = true
							}

							if mustSkip {
								mustSkip = false
								continue
							}

							for _, value := range values {
								req.Out.Header.Add(key, value)
							}
						}
						req.Out.URL.Scheme = u.Scheme
						req.Out.URL.Host = u.Host
						req.Out.AddCookie(&http.Cookie{
							Name:  "panacea-auth",
							Value: "true",
						})
						resp := &http.Response{
							Status:        "200 OK",
							StatusCode:    200,
							Header:        req.Out.Header,
							Body:          req.Out.Body,
							ContentLength: req.Out.ContentLength,
						}
						req.Out.Response = resp

						req.SetXForwarded()
						req.SetURL(u)
						req.Out.Host = req.In.Host
					},

					Transport: &http.Transport{
						MaxIdleConns:          100,
						IdleConnTimeout:       90,
						TLSHandshakeTimeout:   10,
						ExpectContinueTimeout: 1,
						TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
						Protocols:             protos,
						MaxConnsPerHost:       100,
					},
					ErrorLog: log.Default(),
				}

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

	l.Info("Routing table updated", "data", rt.String())
}

func (rt *routingTable) String() string {
	var sb strings.Builder
	for host, routes := range rt.ListAllRoutes() {
		fmt.Fprintf(&sb, "Host: %s", host)
		fmt.Fprintln(&sb, "")
		for _, route := range routes {
			fmt.Fprintf(&sb, "  Path: %s,", route.Path)
			fmt.Fprintln(&sb, "")
			fmt.Fprintf(&sb, "  PathType: %s", route.PathType)
			fmt.Fprintln(&sb, "")
			fmt.Fprintf(&sb, "  Backend: %s", route.Backend.String())
			fmt.Fprintln(&sb, "")
		}
	}
	return sb.String()
}

func (rt *routingTable) Match(host, reqPath string) *Route {
	routes, exists := rt.ListAllRoutes()[host]
	if !exists {
		return nil
	}

	for _, route := range routes {
		switch route.PathType {
		case string(networkingv1.PathTypeExact):
			l.Info("Matched route", "type", route.PathType, "host", host, "path", route.Path, "backend", route.Backend)
			if reqPath == route.Path {
				return route
			}
		case string(networkingv1.PathTypePrefix):
			l.Info("Matched route", "type", route.PathType, "host", host, "path", route.Path, "backend", route.Backend)
			if strings.HasPrefix(reqPath, route.Path) {
				return route
			}
		default:
			l.Info("Matched route", "type", route.PathType, "host", host, "path", route.Path, "backend", route.Backend)
			if strings.HasPrefix(reqPath, route.Path) {
				return route
			}
		}
	}

	return nil
}

func (rt *routingTable) GetRoutes(ingressKey string) []*Route {
	if _, exists := rt.ListAllRoutes()[ingressKey]; !exists {
		return nil
	}

	return rt.ListAllRoutes()[ingressKey]
}

func (rt *routingTable) SetRoutes(ingressKey string, routes []*Route) {
	rt.data[ingressKey] = routes
}

func (rt *routingTable) DeleteRoutes(ingressKey string) {
	delete(rt.data, ingressKey)
}

func (rt *routingTable) ListAllRoutes() map[string][]*Route {
	// Create a copy to avoid external modification
	copy := make(map[string][]*Route)
	maps.Copy(copy, rt.data)
	return copy
}

func (rt *routingTable) Clear() {
	rt.data = make(map[string][]*Route)
}
