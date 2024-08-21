package ranchermonitoring

import (
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/rancher/dartboard/test/utils/grafanautils"
	"github.com/rancher/dartboard/test/utils/ranchermonitoring/dashboards/rancherclusternodes"
)

const DefaultRancherClusterNodesQueryRateInterval = "4m0s"

func RancherClusterCPUUtilizationQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultCPUUtilizationExpr(), start, end)
}

func RancherClusterLoadAverage1QueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.LoadAverage1Expr, start, end)
}

func RancherClusterLoadAverage5QueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.LoadAverage5Expr, start, end)
}

func RancherClusterLoadAverage15QueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.LoadAverage15Expr, start, end)
}

func RancherClusterDiskReadBytesTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultDiskReadBytesTotalExpr(), start, end)
}

func RancherClusterDiskWriteBytesTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultDiskWriteBytesTotalExpr(), start, end)
}

func RancherClusterNetworkReceiveErrorsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultNetworkReceiveErrorsTotalExpr(), start, end)
}

func RancherClusterNetworkReceivePacketsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultNetworkReceivePacketsTotalExpr(), start, end)
}

func RancherClusterNetworkTransmitErrorsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultNetworkTransmitErrorsTotalExpr(), start, end)
}

func RancherClusterNetworkReceiveDropsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultNetworkReceiveDropsTotalExpr(), start, end)
}

func RancherClusterNetworkTransmitDropsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultNetworkTransmitDropsTotalExpr(), start, end)
}

func RancherClusterNetworkTransmitPacketsTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultNetworkTransmitPacketsTotalExpr(), start, end)
}

func RancherClusterNetworkTransmitBytesTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultNetworkTransmitBytesTotalExpr(), start, end)
}

func RancherClusterNetworkReceiveBytesTotalQueryValue(p promv1.API, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	return grafanautils.GetQueryValue(p, rancherclusternodes.DefaultNetworkReceiveBytesTotalExpr(), start, end)
}
