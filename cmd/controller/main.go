package main

import (
	"fmt"
	"os"

	"github.com/danCrespo/panacea-ingress-controller/cmdline"
	"github.com/danCrespo/panacea-ingress-controller/config"
	l "github.com/danCrespo/panacea-ingress-controller/logger"
	"k8s.io/apimachinery/pkg/util/runtime"

	// "k8s.io/klog/v2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log       = l.NewLogger().WithName("main")
	version   string
	gitCommit string
	buildDate string
)

func init() {
	config.Version = version
	config.GitCommit = gitCommit
	config.BuildDate = buildDate
	logf.SetLogger(log.WithName("panacea-ingress-controller"))
}

func main() {
	runtime.Must(nil) // Ensure k8s runtime panics turn  into stack traces
	if err := cmdline.New().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
