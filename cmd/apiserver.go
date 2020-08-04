package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/feifeigood/checkup"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	listenAddress string
	metricsPath   string
	every         string
	basicAuth     bool
	username      string
	password      string

	serverCmd = &cobra.Command{
		Use:   "apiserver",
		Short: "Run checkup with HTTP server",
		Long: `Running checkup with HTTP server for expose metrics to endpoints.
The result of each check is saved to storage. If checkup configuration file 
had been update, you need post request to '/-/reload' for reload it. For safety, 
It only accept localhost requests.
 `,
		Run: func(cmd *cobra.Command, args []string) {
			interval, err := time.ParseDuration(every)
			if err != nil {
				log.Fatal(err)
			}

			lvl, err := logrus.ParseLevel(logLevel)
			if err != nil {
				log.Fatal(err)
			}
			logrus.SetLevel(lvl)

			// running http server
			stop := make(chan struct{})
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-quit
				close(stop)
			}()

			apiserver, err := checkup.NewAPIServer(checkup.Config{
				ConfigPath:    configFile,
				BasicAuth:     basicAuth,
				MetricsPath:   metricsPath,
				Interval:      interval,
				ListenAddress: listenAddress,
				Password:      password,
				Username:      username,
			})

			failOnError(err)

			failOnError(apiserver.Run(stop))
		},
	}
)

func failOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	serverCmd.Flags().StringVar(&listenAddress, "web.listen-address", ":9193", "Address on which to expose metrics and web interface.")
	serverCmd.Flags().StringVar(&metricsPath, "web.telementry-path", "/metrics", "Path under which to expose metrics.")
	serverCmd.Flags().StringVar(&every, "every", "30s", "Runs checkups at the interval")
	serverCmd.Flags().StringVar(&logLevel, "log-level", "info", "Setting log level")
	serverCmd.Flags().BoolVar(&basicAuth, "basic-auth", false, "Enable basic authentication for apiserver")
	serverCmd.Flags().StringVar(&username, "username", "", "Basic authentication username")
	serverCmd.Flags().StringVar(&password, "password", "", "Passwords are hashed with bcrypt")
	rootCmd.AddCommand(serverCmd)
}
