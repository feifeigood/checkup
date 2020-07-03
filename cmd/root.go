package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/feifeigood/checkup"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log            = logrus.WithField("component", "checkup")
	configFile     string
	storageResults bool
	rootCmd        = &cobra.Command{
		Use:   "checkup",
		Short: "Perform checks on your services and sites",
		Long: `Checkup is health checks of any endpoints over HTTP,TCP,DNS,ICMP,TLS and Exec

Checkup will always look for a checkup.json file in
the current working directory by default and use it.
You can specify a different file location using the
--config/-c flag.

Running checkup without any arguments will invoke
a single checkup and print results to stdout. To
store the results of the check, use --store.
`,
		Run: func(cmd *cobra.Command, args []string) {
			allHealthy := true

			c := loadCheckup()

			if len(c.Checkers) == 0 {
				log.Fatal("no checkers configured")
			}

			if storageResults {
				if c.Storage == nil {
					log.Fatal("no storage configured")
				}
			}

			results, err := c.Check()
			if err != nil {
				log.Fatal(err)
			}

			if storageResults {
				err := c.Storage.Store(results)
				if err != nil {
					log.Fatal(err)
				}
				return
			}

			for _, result := range results {
				log.Info(result.String())
				if !result.Healthy {
					allHealthy = false
				}
			}

			if !allHealthy {
				os.Exit(1)
			}
		},
	}
)

func loadCheckup() checkup.Checkup {
	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	var c checkup.Checkup
	err = json.Unmarshal(configBytes, &c)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "checkup.json", "JSON config file")
	rootCmd.Flags().BoolVar(&storageResults, "store", false, "Store checkup results")

	logrus.SetFormatter(&nested.Formatter{
		FieldsOrder: []string{"component"},
		HideKeys:    true,
	})
}
