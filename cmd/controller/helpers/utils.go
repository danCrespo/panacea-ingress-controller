package helpers

import (
	"context"
	"time"

	"github.com/danCrespo/panacea-ingress-controller/kubeutils"
	"github.com/danCrespo/panacea-ingress-controller/logger"
	"github.com/danCrespo/panacea-ingress-controller/routing"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type utils struct {
	GetOnAnyHandler func(fn func()) *onAnyHandler
	Sync            *sync
}

var (
	l = logger.Log
)

func New() *utils {
	return &utils{
		GetOnAnyHandler: newOnAnyHandler,
		Sync:            &sync{},
	}
}

func (u *utils) InClusterOrKubeconfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return kubeutils.NewKubeutils().GetClusterConfig()
}

type onAnyHandler struct {
	fn func()
}

// OnAdd implements onAnyHandler.
func (o *onAnyHandler) OnAdd(obj any, _ bool) {
	o.fn()
}

// OnDelete implements onAnyHandler.
func (o *onAnyHandler) OnDelete(obj any) {
	o.fn()
}

// OnUpdate implements onAnyHandler.
func (o *onAnyHandler) OnUpdate(oldObj any, newObj any) {
	o.fn()
}

func newOnAnyHandler(fn func()) *onAnyHandler {
	return &onAnyHandler{fn: fn}
}

type sync struct{}

func (s *sync) CacheSync(factory informers.SharedInformerFactory, stop <-chan struct{}) bool {
	okMap := factory.WaitForCacheSync(stop)
	for _, ok := range okMap {
		if !ok {
			return false
		}
	}
	return true
}

func (s *sync) SyncIngresses(clientSet *kubernetes.Clientset, router routing.RoutingTable, ingressClass string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	l.Info("Syncing ingresses")
	lists, err := clientSet.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{
		LabelSelector: labels.Everything().String(),
	})
	if err != nil {
		l.Error(err, "error listing ingresses")
		return
	}

	ings := make([]*networkingv1.Ingress, 0, len(lists.Items))
	for i := range lists.Items {
		ing := &lists.Items[i]
		l.Info("Found ingress", "name", ing.Name, "namespace", ing.Namespace)
		ings = append(ings, ing)
	}

	router.UpdateFromIngresses(ings, ingressClass)
	l.Info("Synchronized %d ingresses\n", len(ings))
}
