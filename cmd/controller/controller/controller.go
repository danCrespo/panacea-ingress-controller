package controller

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/danCrespo/panacea-ingress-controller/config"
	"github.com/danCrespo/panacea-ingress-controller/helpers"
	"github.com/danCrespo/panacea-ingress-controller/logger"
	"github.com/danCrespo/panacea-ingress-controller/routing"
	"github.com/go-logr/logr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Controller interface {
	Run() error
	Log(msg ...string)
}

type controller struct {
	*config.Config
	log logr.Logger
}

var _ Controller = (*controller)(nil)

var (
	utils = helpers.New()
)

func NewController(cfg *config.Config) Controller {
	log := logger.NewLoggerWithLevel(cfg.Verbosity)
	ctrlr := &controller{
		cfg,
		log,
	}
	return ctrlr
}

func (c *controller) Log(msg ...string) {
	for _, m := range msg {
		c.log.Info(m)
	}
}

func (c *controller) Run() error {
	cfg, err := utils.InClusterOrKubeconfig(*c.Config)
	if err != nil {
		c.Log(fmt.Sprintf("failed to get kubeconfig: %s", c.Kubeconfig))
	} else {
		c.Log(fmt.Sprintf("Using kubeconfig: %s", c.Kubeconfig))
	}

  
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		c.Log(fmt.Sprintf("failed to create kubernetes clientset: %v", err))
		os.Exit(1)
	}
  
  utils.SetLogger(c.log)
	router := routing.New(*c.Config)
	router.SetLogger(c.log)

	c.Log("Routing table created.")

	factory := informers.NewSharedInformerFactory(clientset, 0)

	c.Log("Informer factory created.")
	ingInformer := factory.Networking().V1().Ingresses()

	c.Log("Ingress informer created.")

	ingInformer.Informer().AddEventHandlerWithOptions(utils.GetOnAnyHandler(func() {
		utils.Sync.SyncIngresses(clientset, router, c.IngressClass)
	}), cache.HandlerOptions{
		Logger:       &c.log,
		ResyncPeriod: nil,
	})

	c.Log("Event handlers added to informer.")

	stop := make(chan struct{})
	defer close(stop)

	factory.Start(stop)

	c.Log("Informer factory started.")

	if !utils.Sync.CacheSync(factory, stop) {
		c.Log("failed to sync caches")
		return fmt.Errorf("failed to sync caches")
	}

	c.Log("Caches synced.")
	utils.Sync.SyncIngresses(clientset, router, c.IngressClass)

	c.Log(fmt.Sprintf("Ingress class %s sync complete.", c.IngressClass))

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host

		if i := strings.IndexByte(host, ':'); i > 0 {
			host = host[:i]
		}

		c.Log(fmt.Sprintf("Received request host=%s path=%s", host, r.URL.Path))

		if rt := router.Match(host, r.URL.Path); rt != nil {
			r.Host = rt.Backend.Host
			r.Header.Set("X-Forwarded-Host", host)
			r.Header.Set("X-Forwarded-Proto", rt.Backend.Scheme)
			r.Header.Set("X-Forwarded-For", rt.Backend.Host)
			w.Header().Set("X-Proxy-By", "panacea-controller")
			rt.Proxy.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("panacea-controller: no route found\n"))
	})

	srv := &http.Server{
		Addr:    c.Listen,
		Handler: h,
	}

	c.Log(fmt.Sprintf("panacea-controller listening on %s", c.Listen))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		c.Log(fmt.Sprintf("HTTP server failed: %v", err))
		os.Exit(1)
	}
	return nil
}
