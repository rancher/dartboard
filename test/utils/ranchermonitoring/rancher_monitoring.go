package ranchermonitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"

	"github.com/git-ival/dartboard/test/utils/grafanautils"
	"github.com/git-ival/dartboard/test/utils/ranchermonitoring/dashboards/kubernetesapiserver"
	"github.com/git-ival/dartboard/test/utils/ranchermonitoring/dashboards/rancherclusternodes"
	"github.com/git-ival/dartboard/test/utils/ranchermonitoring/dashboards/rancherperformancedebugging"
	gapi "github.com/grafana/grafana-api-golang-client"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	PrometheusRoute            = "/api/v1/namespaces/cattle-monitoring-system/services/http:rancher-monitoring-prometheus:9090/proxy"
	GrafanaRoute               = "/api/v1/namespaces/cattle-monitoring-system/services/http:rancher-monitoring-grafana:80/proxy"
	GrafanaSnapshotRoute       = GrafanaRoute + "/" + "dashboard/snapshot/"
	PanelContentSelector       = "div.panel-content"
	IngressConfigMapName       = "ingress-nginx-controller"
	GrafanaConfigMapName       = "grafana-nginx-proxy-config"
	GrafanaDeploymentName      = "rancher-monitoring-grafana"
	RancherMonitoringNamespace = "cattle-monitoring-system"
)

func SetupClients(rancherHost string, adminToken string, adminPassword string) (*promv1.API, *gapi.Client, error) {
	baseURL := "https://" + rancherHost

	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: adminToken, TokenType: "Bearer"}))
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, nil, err
	}
	httpClient.Jar = jar
	promConfig := promapi.Config{
		Client:  httpClient,
		Address: baseURL + PrometheusRoute + "/",
	}
	promClient, err := promapi.NewClient(promConfig)
	if err != nil {
		log.Infof("error creating new prometheus client: %v", err)
		return nil, nil, err
	}
	promV1 := promv1.NewAPI(promClient)

	rancherLoginJSON, err := json.Marshal(map[string]string{"username": "admin", "password": adminPassword, "responseType": "cookie"})
	if err != nil {
		log.Infof("error marshalling rancher login json (%s): %v", string(rancherLoginJSON), err)
		return &promV1, nil, err
	}
	rancherLoginPayload := bytes.NewReader(rancherLoginJSON)
	resp, err := httpClient.Post(baseURL+"/v3-public/localProviders/local?action=login", "application/json", rancherLoginPayload)
	if err != nil {
		log.Infof("error failed to make request (%v) with response (%v): %v", resp.Request, resp, err)
		return &promV1, nil, err
	}

	grafanaLoginJSON, err := json.Marshal(map[string]string{"user": "admin", "password": "prom-operator"})
	if err != nil {
		log.Infof("error marshalling grafana login json (%s): %v", string(grafanaLoginJSON), err)
		return &promV1, nil, err
	}
	grafanaLoginPayload := bytes.NewReader(grafanaLoginJSON)

	resp, err = httpClient.Post(baseURL+GrafanaRoute+"/login", "application/json", grafanaLoginPayload)
	if err != nil {
		log.Infof("error failed to make request (%v) with response (%v): %v", resp.Request, resp, err)
		return &promV1, nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Infof("error failed to make request (%v) with response (%v): %v", resp.Request, resp, err)
		return &promV1, nil, err
	}
	fmt.Println(string(body))

	gapiConfig := gapi.Config{
		BasicAuth:  url.UserPassword("admin", "prom-operator"),
		NumRetries: 3,
		OrgID:      1,
		Client:     httpClient,
	}
	gapiClient, err := grafanautils.NewClient(baseURL+GrafanaRoute, gapiConfig)
	if err != nil {
		log.Infof("error creating new grafana client: %v", err)
		return &promV1, nil, err
	}

	return &promV1, gapiClient, err
}

func CustomKubernetesAPIServerUIDs() []string {
	return []string{kubernetesapiserver.UID1, kubernetesapiserver.UID2}
}

func CustomRancherPerformanceDebuggingUIDs() []string {
	return []string{rancherperformancedebugging.UIDFormatted1, rancherperformancedebugging.UIDFormatted2, rancherperformancedebugging.UIDFormatted3}
}

func RancherClusterNodesExprs() []string {
	return []string{
		rancherclusternodes.CPUUtilizationExpr,
		rancherclusternodes.LoadAverage1Expr,
		rancherclusternodes.LoadAverage5Expr,
		rancherclusternodes.LoadAverage15Expr,
		rancherclusternodes.MemUtilizationExpr,
		rancherclusternodes.DiskUtilizationExpr,
		rancherclusternodes.DiskReadBytesTotalExpr,
		rancherclusternodes.DiskWriteBytesTotalExpr,
		rancherclusternodes.NetworkReceiveErrorsTotalExpr,
		rancherclusternodes.NetworkReceivePacketsTotalExpr,
		rancherclusternodes.NetworkTransmitErrorsTotalExpr,
		rancherclusternodes.NetworkReceiveDropsTotalExpr,
		rancherclusternodes.NetworkTransmitDropsTotalExpr,
		rancherclusternodes.NetworkTransmitPacketsTotalExpr,
		rancherclusternodes.NetworkTransmitBytesTotalExpr,
		rancherclusternodes.NetworkReceiveBytesTotalExpr,
	}
}

func RancherPerformanceDebuggingExprs() []string {
	return []string{
		rancherperformancedebugging.HandlerAverageExecutionTimesExpr,
		rancherperformancedebugging.RancherAPIAverageRequestTimesExpr,
		rancherperformancedebugging.SubscribeAverageRequestTimesExpr,
		rancherperformancedebugging.LassoControllerWorkQueueDepthExpr,
		rancherperformancedebugging.NumberOfRancherRequestsExpr,
		rancherperformancedebugging.NumberOfFailedRancherRequestsExpr,
		rancherperformancedebugging.K8sProxyStoreAverageRequestTimesExpr,
		rancherperformancedebugging.K8sProxyClientAverageRequestTimesExpr,
		rancherperformancedebugging.CachedObjectsByGroupVersionKindExpr,
		rancherperformancedebugging.LassoHandlerExecutionsExpr,
		rancherperformancedebugging.LassoHandlerExecutionsByClusterNameExpr,
		rancherperformancedebugging.LassoHandlerExecutionsWithErrorExpr,
		rancherperformancedebugging.LassoHandlerExecutionsWithErrorByClusterNameExpr,
		rancherperformancedebugging.DataTransmittedByRemoteDialerSessionsExpr,
		rancherperformancedebugging.ErrorsByRemoteDialerSessionsExpr,
		rancherperformancedebugging.RemoteDialerConnectionsRemovedExpr,
		rancherperformancedebugging.RemoteDialerConnectionsAddedByClientExpr,
	}
}

func GrafanaNginxYAML(relPath string) (string, error) {
	files, err := os.ReadDir(relPath)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(relPath)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if file.Name() == "nginx.yaml" {
			return filepath.Join(path, file.Name()), nil
		}
	}
	return "", fmt.Errorf("did not find `nginx.yaml` in path (%s)", path)
}
