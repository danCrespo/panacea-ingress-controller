package panaceaingresscontroller

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	controllerconfig "github.com/danCrespo/panacea-ingress-controller/config"
	h "github.com/danCrespo/panacea-ingress-controller/helpers"
	"github.com/danCrespo/panacea-ingress-controller/routing"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	log   = logf.Log.WithName("panacea-ingress-controller")
	utils = h.New()
)

func main() {
	runtime.Must(nil) // Ensure k8s runtime panics turn  into stack traces

	ctcfg := controllerconfig.LoadControllerConfig()

	cfg, err := utils.InClusterOrKubeconfig(ctcfg.Kubeconfig)
	if err != nil {
		log.Error(err, "failed to get kubeconfig: ", ctcfg.Kubeconfig)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Error(err, "failed to create kubernetes clientset")
		os.Exit(1)
	}

	router := routing.New()

	factory := informers.NewSharedInformerFactory(clientset, 0)
	ingInformer := factory.Networking().V1().Ingresses()

	ingInformer.Informer().AddEventHandler(utils.GetOnAnyHandler(func() {
		utils.Sync.SyncIngresses(clientset, router, ctcfg.IngressClass)
	}))

	stop := make(chan struct{})
	defer close(stop)

	factory.Start(stop)

	if !utils.Sync.CacheSync(factory, stop) {
		log.Error(fmt.Errorf("failed to sync caches"), "")
		return
	}

	utils.Sync.SyncIngresses(clientset, router, ctcfg.IngressClass)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host

		if i := strings.IndexByte(host, ':'); i > 0 {
			host = host[:i]
		}

		if rt := router.Match(host, r.URL.Path); rt != nil {
			r.Host = rt.Backend.Host
			r.Header.Set("X-Forwarded-Host", host)
			r.Header.Set("X-Forwarded-Proto", r.Proto)
			r.Header.Set("X-Forwarded-For", r.RemoteAddr)
			rt.Proxy.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("panacea-ingress-controller: no route found\n"))
	})

	srv := &http.Server{
		Addr:    ctcfg.ListenAddress,
		Handler: h,
	}
	log.Info("panacea-ingress-controller listening on " + ctcfg.ListenAddress)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error(err, "HTTP server failed")
		os.Exit(1)
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to set up overall controller manager: %v\n", err)
		os.Exit(1)
	}

	// Start the manager in a separate goroutine
	go func() {
		if err := mgr.Start(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "unable to run the manager: %v\n", err)
			os.Exit(1)
		}
	}()

	c := mgr.GetClient()

	go func() {
		for {
			var ingresses networkingv1.IngressList
			if err := c.List(context.TODO(), &ingresses); err != nil {
				log.Error(err, "unable to list ingresses")
			} else {
				for _, ingress := range ingresses.Items {
					log.Info(fmt.Sprintf("Found ingress: %s/%s", ingress.Namespace, ingress.Name))
				}
			}

			// Sleep for a while before checking again
			time.Sleep(30 * time.Second)
		}
	}()

	http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Panacea Ingress Controller is running\n"))
	}))
}
