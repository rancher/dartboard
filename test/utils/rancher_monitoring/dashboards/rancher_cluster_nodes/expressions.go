package rancher_cluster_nodes

import (
	"fmt"
)

const DefaultRancherClusterNodesQueryRateInterval = "4m0s"

func DefaultCPUUtilizationExpr() string {
	return fmt.Sprintf("1 - avg(irate({__name__=~\"node_cpu_seconds_total|windows_cpu_time_total\",mode=\"idle\"}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultDiskReadBytesTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_disk_read_bytes_total[%s]) OR rate(windows_logical_disk_read_bytes_total[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultDiskWriteBytesTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_disk_written_bytes_total[%s]) OR rate(windows_logical_disk_write_bytes_total[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultNetworkReceiveErrorsTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_network_receive_errs_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[%s])) by (instance) OR sum(rate(windows_net_packets_received_errors_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultNetworkReceivePacketsTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_network_receive_packets_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[%s])) by (instance) OR sum(rate(windows_net_packets_received_total_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultNetworkTransmitErrorsTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_network_transmit_errs_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[%s])) by (instance) OR sum(rate(windows_net_packets_outbound_errors_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultNetworkReceiveDropsTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_network_receive_drop_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[%s])) by (instance) OR sum(rate(windows_net_packets_received_discarded_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultNetworkTransmitDropsTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_network_transmit_drop_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[%s])) by (instance) OR sum(rate(windows_net_packets_outbound_discarded{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultNetworkTransmitPacketsTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_network_transmit_packets_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[%s])) by (instance) OR sum(rate(windows_net_packets_sent_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultNetworkTransmitBytesTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_network_transmit_bytes_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[%s]) OR rate(windows_net_packets_sent_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}

func DefaultNetworkReceiveBytesTotalExpr() string {
	return fmt.Sprintf("sum(rate(node_network_receive_bytes_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[%s]) OR rate(windows_net_packets_received_total_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[%s])) by (instance)", DefaultRancherClusterNodesQueryRateInterval, DefaultRancherClusterNodesQueryRateInterval)
}
