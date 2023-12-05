package grafanautils

import (
	"time"

	"github.com/git-ival/dartboard/test/utils/grafanautils"
	rancher_cluster_nodes "github.com/git-ival/dartboard/test/utils/rancher_monitoring/dashboards/rancher_cluster_nodes"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const DefaultRancherClusterNodesQueryRateInterval = "4m0s"

func RancherClusterCPUUtilizationQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultCPUUtilizationExpr(), start, end)
}

func RancherClusterLoadAverage1QueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.LoadAverage1Expr, start, end)
}

func RancherClusterLoadAverage5QueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.LoadAverage5Expr, start, end)
}

func RancherClusterLoadAverage15QueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.LoadAverage15Expr, start, end)
}

func RancherClusterDiskReadBytesTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultDiskReadBytesTotalExpr(), start, end)
}

func RancherClusterDiskWriteBytesTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultDiskWriteBytesTotalExpr(), start, end)
}

func RancherClusterNetworkReceiveErrorsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultNetworkReceiveErrorsTotalExpr(), start, end)
}

func RancherClusterNetworkReceivePacketsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultNetworkReceivePacketsTotalExpr(), start, end)
}

func RancherClusterNetworkTransmitErrorsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultNetworkTransmitErrorsTotalExpr(), start, end)
}

func RancherClusterNetworkReceiveDropsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultNetworkReceiveDropsTotalExpr(), start, end)
}

func RancherClusterNetworkTransmitDropsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultNetworkTransmitDropsTotalExpr(), start, end)
}

func RancherClusterNetworkTransmitPacketsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultNetworkTransmitPacketsTotalExpr(), start, end)
}

func RancherClusterNetworkTransmitBytesTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultNetworkTransmitBytesTotalExpr(), start, end)
}

func RancherClusterNetworkReceiveBytesTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancher_cluster_nodes.DefaultNetworkReceiveBytesTotalExpr(), start, end)
}
