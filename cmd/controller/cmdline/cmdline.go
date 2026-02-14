package cmdline

import (
	"fmt"
	"os"
	"strings"

	. "github.com/danCrespo/panacea-ingress-controller/config"
	"github.com/danCrespo/panacea-ingress-controller/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CmdLine interface {
	Execute() error
	initConfiguration()
	newPanaceaIngressCommand() *cobra.Command
	newRunCommand() *cobra.Command
	newVersionCommand() *cobra.Command
}

type cmdline struct {
	flags  ControllerFlags
	config Config
}

var _ CmdLine = (*cmdline)(nil)

func New() CmdLine {
	return &cmdline{
		flags: ControllerFlags{
			IngressClass: "",
			Listen:       "",
			Kubeconfig:   "",
			ResyncPeriod: "",
			Namespace:    "",
			Help:         false,
			Verbosity:    0,
		},
		config: Config{},
	}
}

func (c *cmdline) Execute() error {
	rootCmd := c.newPanaceaIngressCommand()
	rootCmd.Execute()
	return nil
}

func (c *cmdline) newPanaceaIngressCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "controller",
		Args:              cobra.ExactArgs(1),
		DisableAutoGenTag: true,
		Short:             "Panacea Ingress Controller startup command",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmd.UsageString())
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			c.initConfiguration()
		},
	}

	cmd.SetVersionTemplate(`Panacea Ingress Controller:
	Version: {{.Version}}
	Git Commit: {{.GitCommit}}
	Build Date: {{.BuildDate}}
`)

	c.flags.BindFlags(cmd)

	cmd.AddCommand(
		c.newRunCommand(),
		c.newVersionCommand(),
	)

	return cmd
}

func (c *cmdline) initConfiguration() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	c.flags.BindViper()
	c.config = Config{
		IngressClass: c.flags.IngressClass,
		Listen:       c.flags.Listen,
		Kubeconfig:   c.flags.Kubeconfig,
		ResyncPeriod: c.flags.ResyncPeriod,
		Namespace:    c.flags.Namespace,
		Verbosity:    c.flags.Verbosity,
	}
}

func (c *cmdline) newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "start",
		Short:             "Start the Panacea Ingress Controller",
		DisableAutoGenTag: true,
		ValidArgs:         []string{},
		ArgAliases: []string{
			"s",
			"run",
			"init",
			"execute",
			"launch",
			"do",
			"begin",
		},
		DisableFlagsInUseLine: false,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ctrl := controller.NewController(&c.config)
			if err := ctrl.Run(); err != nil {
				ctrl.Log(fmt.Sprintf("failed to run Panacea Ingress Controller: %v", err))
				os.Exit(1)
			}
		},
	}

	c.flags.BindFlags(cmd)

	return cmd
}

func (c *cmdline) newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "version",
		Short:                 "Print the version number of Panacea Ingress Controller",
		DisableAutoGenTag:     true,
		ValidArgs:             []string{},
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Panacea Ingress Controller:")
			fmt.Printf("\tVersion: %s\n", Version)
			fmt.Printf("\tGit Commit: %s\n", GitCommit)
			fmt.Printf("\tBuild Date: %s\n", BuildDate)
			os.Exit(0)
		},
	}
	return cmd
}
