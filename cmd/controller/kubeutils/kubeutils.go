package kubeutils

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danCrespo/panacea-ingress-controller/config"
	"github.com/danCrespo/panacea-ingress-controller/logger"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type IKubeutils interface {
	GetClusterConfig() (*rest.Config, error)
	GetClusterName() (string, error)
	GetNamespace() (string, error)
	GetClusterDomain() (string, error)
	GetResource(namespace, name, kind string) (any, error)
	GetServicePortByName(namespace string, name string, param3 string) (int32, error)
	SetLogger(logger logr.Logger)
}

type kubeutils struct {
	config     config.Config
	clientSet  *kubernetes.Clientset
	kubeconfig *rest.Config
}

var (
	l = logger.NewLogger().WithValues("component", "kubeutils")
)

func NewKubeutils(cfg config.Config) IKubeutils {

	kc := getConfig(cfg.Kubeconfig)
	clientSet, err := kubernetes.NewForConfig(kc)
	if err != nil {
		panic(err.Error())
	}
	k := &kubeutils{
		config:     cfg,
		clientSet:  clientSet,
		kubeconfig: kc,
	}

	return k
}

func (k *kubeutils) SetLogger(logger logr.Logger) {
	l = logger
}

func (k *kubeutils) GetClusterConfig() (*rest.Config, error) {
	var err error
	if k.kubeconfig == nil {
		err = fmt.Errorf("no kubeconfig available")
	}

	return k.kubeconfig, err
}

func (k *kubeutils) GetClusterName() (string, error) {
	cfg, err := k.GetClusterConfig()
	if err != nil {
		return "", err
	}

	return cfg.Host, nil
}

func (k *kubeutils) GetClusterDomain() (string, error) {
	ctx := context.Background()

	cmi := k.clientSet.CoreV1().ConfigMaps("kube-system")
	if cmi == nil {
		return "", fmt.Errorf("error getting cluster-info configmap: %v", "kube-system")

	}

	cm, err := cmi.Get(ctx, "kubelet-config",
		metav1.GetOptions{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1"},
			ResourceVersion: "0"})

	if err != nil {
		return "", fmt.Errorf("error getting cluster-info configmap: %v", err)
	}

	kubelet, ok := cm.Data["kubelet"]
	if !ok || kubelet == "" {
		return "cluster.local", nil
	}

	if strings.Contains(kubelet, "clusterDomain:") == false {
		return "cluster.local", nil
	}

	_, aft, success := strings.Cut(kubelet, "clusterDomain: ")
	if !success {
		return "cluster.local", nil
	}

	domain := ""

	for i, c := range aft {
		if c == '\n' || c == '\r' || c == ' ' {
			break
		}

		domain += string(c)
		if i > 253 {
			break
		}

	}
	if domain == "" {
		return "cluster.local", nil
	}

	return domain, nil
}

func (k *kubeutils) GetNamespace() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pods, err := k.clientSet.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: labels.Everything().String(),
	})
	if err != nil {
		return "", fmt.Errorf("error listing pods: %v", err)
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found in the cluster")
	}

	return pods.Items[0].Namespace, nil
}

func (k *kubeutils) GetResource(namespace, name, kind string) (any, error) {
	ctx := context.Background()

	l.Info("Getting resource", "kind", kind, "namespace", namespace, "name", name)

	switch strings.ToLower(kind) {
	case "pod", "pods":
		pod, err := k.clientSet.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting pod %s/%s: %v", namespace, name, err)
		}
		return pod, nil
	case "service", "services":
		svc, err := k.clientSet.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting service %s/%s: %v", namespace, name, err)
		}
		return svc, nil
	case "deployment", "deployments":
		deploy, err := k.clientSet.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting deployment %s/%s: %v", namespace, name, err)
		}
		return deploy, nil
	case "ingress", "ingresses":
		ingress, err := k.clientSet.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting ingress %s/%s: %v", namespace, name, err)
		}
		return ingress, nil

	case "configmap", "configmaps":
		cm, err := k.clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting configmap %s/%s: %v", namespace, name, err)
		}
		return cm, nil

	case "secret", "secrets":
		secret, err := k.clientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting secret %s/%s: %v", namespace, name, err)
		}
		return secret, nil

	case "statefulset", "statefulsets":
		ss, err := k.clientSet.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting statefulset %s/%s: %v", namespace, name, err)
		}
		return ss, nil

	case "daemonset", "daemonsets":
		ds, err := k.clientSet.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting daemonset %s/%s: %v", namespace, name, err)
		}
		return ds, nil

	case "job", "jobs":
		job, err := k.clientSet.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting job %s/%s: %v", namespace, name, err)
		}
		return job, nil

	case "cronjob", "cronjobs":
		cronjob, err := k.clientSet.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting cronjob %s/%s: %v", namespace, name, err)
		}
		return cronjob, nil
	default:
		return nil, fmt.Errorf("unsupported resource kind: %s", kind)
	}
}

func (k *kubeutils) GetServicePortByName(namespace string, name string, portName string) (int32, error) {
	ctx := context.Background()

	svc, err := k.clientSet.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("error getting service %s/%s: %v", namespace, name, err)
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == portName {
			return port.Port, nil
		}
	}

	return 0, fmt.Errorf("port name %s not found in service %s/%s", portName, namespace, name)
}

func getConfig(kc string) *rest.Config {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		if kc != "" {
			cfg, err = clientcmd.BuildConfigFromFlags("", kc)
			if err != nil {
				panic(err.Error())
			}
			return cfg
		}
		if home, err := os.UserHomeDir(); err == nil {
			kubeconfig := home + "/.kube/config"
			cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				panic(err.Error())
			}
			return cfg
		}
		panic("no kubeconfig available")
	}
	return cfg
}
