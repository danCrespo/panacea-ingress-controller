package cmdline

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ControllerFlags struct {
	IngressClass string `flag:"ingress-class,c" help:"IngressClass name to reconcile" default:"panacea-ingress-class"`
	Listen       string `flag:"listen,l" help:"Address to listen on for HTTP requests" default:"0.0.0.0:80"`
	Kubeconfig   string `flag:"kubeconfig,k" help:"Path to a kubeconfig. Only required if out-of-cluster" default:""`
	ResyncPeriod string `flag:"resync-period,r" help:"Resync period in seconds" default:"30"`
	Namespace    string `flag:"namespace,n" help:"Namespace to watch for Ingress resources. Leave empty to watch all namespaces." default:""`
	Help         bool   `flag:"help,h" help:"Help for panacea-ingress-controller" default:"false"`
	Verbosity    int    `flag:"verbosity,v" help:"Logging verbosity level" default:"0"`
}

func bindFlags(cmd *cobra.Command, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		flagTag := field.Tag.Get("flag")
		if flagTag == "" {
			continue
		}

		// Parse flag tag: "name,shorthand"
		parts := strings.SplitN(flagTag, ",", 2)
		name := parts[0]
		shorthand := ""
		if len(parts) > 1 {
			shorthand = parts[1]
		}

		help := field.Tag.Get("help")
		defaultVal := field.Tag.Get("default")

		fieldPtr := v.Field(i).Addr().Interface()

		switch field.Type.Kind() {
		case reflect.String:
			cmd.Flags().StringVarP(fieldPtr.(*string), name, shorthand, defaultVal, help)
		case reflect.Bool:
			def, _ := strconv.ParseBool(defaultVal)
			cmd.Flags().BoolVarP(fieldPtr.(*bool), name, shorthand, def, help)
		case reflect.Int:
			def, _ := strconv.Atoi(defaultVal)
			cmd.Flags().IntVarP(fieldPtr.(*int), name, shorthand, def, help)
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.String {
				cmd.Flags().StringSliceVarP(fieldPtr.(*[]string), name, shorthand, nil, help)
			}
		case reflect.Map:
			if field.Type.Key().Kind() == reflect.String {
				cmd.Flags().StringToStringVarP(fieldPtr.(*map[string]string), name, shorthand, nil, help)
			}
		}
	}

	return nil
}

func (cf *ControllerFlags) BindFlags(cmd *cobra.Command) error {
	return bindFlags(cmd, cf)
}

func (cf *ControllerFlags) BindViper() error {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	viper.SetDefault("ingress-class", cf.IngressClass)
	viper.SetDefault("listen", cf.Listen)
	viper.SetDefault("kubeconfig", cf.Kubeconfig)
	viper.SetDefault("resync-period", cf.ResyncPeriod)
	viper.SetDefault("namespace", cf.Namespace)
	viper.SetDefault("verbosity", cf.Verbosity)

	return nil
}
