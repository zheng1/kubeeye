package web

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"os"

	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

//go:embed dist/*
var assets embed.FS

type errorResponder struct{}

func (e *errorResponder) Error(w http.ResponseWriter, req *http.Request, err error) {
	klog.Error(err)
	responsewriters.InternalError(w, req, err)
}

func loadConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}
	return clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")

}

func RunWebService(ctx context.Context) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	subFS, _ := fs.Sub(assets, "dist")
	assetsFs := http.FileServer(http.FS(subFS))
	kubernetes, _ := url.Parse(config.Host)
	defaultTransport, err := rest.TransportFor(config)
	if err != nil {
		return err
	}

	var handleKubeAPIfunc = func(w http.ResponseWriter, req *http.Request) {
		klog.Info(req.URL)
		s := *req.URL
		s.Host = kubernetes.Host
		s.Scheme = kubernetes.Scheme

		// make sure we don't override kubernetes's authorization
		req.Header.Del("Authorization")
		httpProxy := proxy.NewUpgradeAwareHandler(&s, defaultTransport, true, false, &errorResponder{})
		httpProxy.UpgradeTransport = proxy.NewUpgradeRequestRoundTripper(defaultTransport, defaultTransport)
		httpProxy.ServeHTTP(w, req)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/apis/", handleKubeAPIfunc)
	mux.HandleFunc("/api/", handleKubeAPIfunc)
	mux.Handle("/", assetsFs)

	klog.Info("server at :8000")
	return http.ListenAndServe(":8000", mux)
}
