package checkup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config is represents APIServer configure structure
type Config struct {
	ListenAddress string
	MetricsPath   string
	ConfigPath    string
	Interval      time.Duration
	BasicAuth     bool
	Username      string
	Password      string
}

// APIServer performs a routine checkup on endpoints or services and manage checkup controller by HTTP endpoints.
type APIServer struct {
	config Config

	server    *http.Server
	container *restful.Container

	controller *Controller

	logger *logrus.Entry
}

// NewAPIServer creates a checkup APIServer object
func NewAPIServer(config Config) (*APIServer, error) {
	apiserver := &APIServer{
		config: config,
		logger: logrus.WithField("component", "apiserver"),
	}

	apiserver.controller = NewController(config.ConfigPath, config.Interval)
	cr := restful.NewContainer()
	cr.Router(restful.CurlyRouter{})
	cr.RecoverHandler(func(panicReason interface{}, httpWriter http.ResponseWriter) {
		apiserver.logStackOnRecover(panicReason, httpWriter)
	})

	if config.BasicAuth {
		cr.Filter(apiserver.basicAuthenticate)
		cr.Handle(config.MetricsPath, apiserver.handleAuth(promhttp.Handler()))
	} else {
		cr.Handle(config.MetricsPath, promhttp.Handler())
	}

	apiWs := new(restful.WebService)
	apiWs.Route(apiWs.POST("/-/reload").To(apiserver.Reload))
	cr.Add(apiWs)

	apiserver.container = cr

	apiserver.server = &http.Server{
		Addr:    config.ListenAddress,
		Handler: apiserver.container,
	}

	return apiserver, nil
}

// Reload reload checkup configuration file
func (a *APIServer) Reload(request *restful.Request, response *restful.Response) {
	go a.controller.Reload()
	io.WriteString(response, "OK")
}

// Run run checkup server
func (a *APIServer) Run(stop <-chan struct{}) error {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-stop
		_ = a.server.Shutdown(ctx)
	}()

	go a.controller.Run()

	a.logger.Infof("listening on %s", a.config.ListenAddress)
	if err = a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (a *APIServer) logStackOnRecover(panicReason interface{}, w http.ResponseWriter) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("recover from panic situation: - %v\r\n", panicReason))
	for i := 2; ; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		buffer.WriteString(fmt.Sprintf("    %s:%d\r\n", file, line))
	}
	a.logger.Errorln(buffer.String())

	headers := http.Header{}
	if ct := w.Header().Get("Content-Type"); len(ct) > 0 {
		headers.Set("Accept", ct)
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal server error"))
}

func (a *APIServer) basicAuthenticate(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	u, p, ok := req.Request.BasicAuth()
	if !ok || u != a.config.Username || bcrypt.CompareHashAndPassword([]byte(a.config.Password), []byte(p)) != nil {
		resp.AddHeader("WWW-Authenticate", "Basic realm=Protected Area")
		resp.WriteErrorString(401, "401: Not Authorized")
		return
	}
	chain.ProcessFilter(req, resp)
}

func (a *APIServer) handleAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != a.config.Username || bcrypt.CompareHashAndPassword([]byte(a.config.Password), []byte(p)) != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "401: Not Authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
