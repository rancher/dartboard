package rancher_monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/cookiejar"
	"net/url"

	"github.com/git-ival/dartboard/test/utils/grafanautils"
	"github.com/git-ival/dartboard/test/utils/rancher_monitoring/dashboards/kubernetes_api_server"
	"github.com/git-ival/dartboard/test/utils/rancher_monitoring/dashboards/rancher_cluster_nodes"
	"github.com/git-ival/dartboard/test/utils/rancher_monitoring/dashboards/rancher_performance_debugging"
	gapi "github.com/grafana/grafana-api-golang-client"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	PrometheusRoute      = "/api/v1/namespaces/cattle-monitoring-system/services/http:rancher-monitoring-prometheus:9090/proxy"
	GrafanaRoute         = "/api/v1/namespaces/cattle-monitoring-system/services/http:rancher-monitoring-grafana:80/proxy"
	GrafanaSnapshotRoute = GrafanaRoute + "/" + "dashboard/snapshot/"
	PanelContentSelector = "div.panel-content"
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
	return []string{kubernetes_api_server.UID1, kubernetes_api_server.UID2}
}

func CustomRancherPerformanceDebuggingUIDs() []string {
	return []string{rancher_performance_debugging.UIDFormatted1, rancher_performance_debugging.UIDFormatted2, rancher_performance_debugging.UIDFormatted3}
}

func RancherClusterNodesExprs() []string {
	return []string{
		rancher_cluster_nodes.CPUUtilizationExpr,
		rancher_cluster_nodes.LoadAverage1Expr,
		rancher_cluster_nodes.LoadAverage5Expr,
		rancher_cluster_nodes.LoadAverage15Expr,
		rancher_cluster_nodes.MemUtilizationExpr,
		rancher_cluster_nodes.DiskUtilizationExpr,
		rancher_cluster_nodes.DiskReadBytesTotalExpr,
		rancher_cluster_nodes.DiskWriteBytesTotalExpr,
		rancher_cluster_nodes.NetworkReceiveErrorsTotalExpr,
		rancher_cluster_nodes.NetworkReceivePacketsTotalExpr,
		rancher_cluster_nodes.NetworkTransmitErrorsTotalExpr,
		rancher_cluster_nodes.NetworkReceiveDropsTotalExpr,
		rancher_cluster_nodes.NetworkTransmitDropsTotalExpr,
		rancher_cluster_nodes.NetworkTransmitPacketsTotalExpr,
		rancher_cluster_nodes.NetworkTransmitBytesTotalExpr,
		rancher_cluster_nodes.NetworkReceiveBytesTotalExpr,
	}
}

func RancherPerformanceDebuggingExprs() []string {
	return []string{
		rancher_performance_debugging.HandlerAverageExecutionTimesExpr,
		rancher_performance_debugging.RancherAPIAverageRequestTimesExpr,
		rancher_performance_debugging.SubscribeAverageRequestTimesExpr,
		rancher_performance_debugging.LassoControllerWorkQueueDepthExpr,
		rancher_performance_debugging.NumberOfRancherRequestsExpr,
		rancher_performance_debugging.NumberOfFailedRancherRequestsExpr,
		rancher_performance_debugging.K8sProxyStoreAverageRequestTimesExpr,
		rancher_performance_debugging.K8sProxyClientAverageRequestTimesExpr,
		rancher_performance_debugging.CachedObjectsByGroupVersionKindExpr,
		rancher_performance_debugging.LassoHandlerExecutionsExpr,
		rancher_performance_debugging.LassoHandlerExecutionsByClusterNameExpr,
		rancher_performance_debugging.LassoHandlerExecutionsWithErrorExpr,
		rancher_performance_debugging.LassoHandlerExecutionsWithErrorByClusterNameExpr,
		rancher_performance_debugging.DataTransmittedByRemoteDialerSessionsExpr,
		rancher_performance_debugging.ErrorsByRemoteDialerSessionsExpr,
		rancher_performance_debugging.RemoteDialerConnectionsRemovedExpr,
		rancher_performance_debugging.RemoteDialerConnectionsAddedByClientExpr,
	}
}
