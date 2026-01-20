package config

import (
	"flag"
	"os"
	strutil "strconv"
)

type ControllerConfig struct {
  IngressClass  string
  ListenAddress string
  Kubeconfig    string
  ResyncPeriod  string
  Namespace     string
}

func LoadControllerConfig() *ControllerConfig {
  var cfg ControllerConfig
  flag.StringVar(&cfg.IngressClass, "ingress-class", envDefault("INGRESS_CLASS", "panacea-ingress-class"), "IngressClass name to reconcile")
  flag.StringVar(&cfg.ListenAddress, "listen", envDefault("LISTEN_ADDR", "0.0.0.0:80"), "Address to listen on for HTTP requests")
  flag.StringVar(&cfg.Kubeconfig, "kubeconfig", envDefault("KUBECONFIG", ""), "Path to a kubeconfig. Only required if out-of-cluster.")
  flag.StringVar(&cfg.ResyncPeriod, "resync-period", envDefault("CONTROLLER_RSYNC_PERIOD",  30), "Resync period in seconds")
  flag.StringVar(&cfg.Namespace, "namespace", envDefault("NAMESPACE", "default"), "Namespace to watch for Ingress resources. Leave empty to watch all namespaces.")
  flag.Parse()
  return &cfg
}

func envDefault(key string, def any) string {
  var defStr string

  switch v := def.(type) {
  case string:
    defStr = v
    case int:
    defStr = strutil.Itoa(v)
  case bool:
    defStr = strutil.FormatBool(v)
  default:
    defStr = def.(string)
  }
 
  if value := os.Getenv(key); value != "" {
    return value
  }
  return defStr
}

