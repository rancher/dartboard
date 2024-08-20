package rancherclusternodes

const (
	UID                             = "rancher-cluster-nodes-1"
	CPUUtilizationExpr              = "1 - avg(irate({__name__=~\"node_cpu_seconds_total|windows_cpu_time_total\",mode=\"idle\"}[$__rate_interval])) by (instance)"
	LoadAverage1Expr                = "sum(node_load1 OR avg_over_time(windows_system_processor_queue_length[1m])) by (instance)"
	LoadAverage5Expr                = "sum(node_load5 OR avg_over_time(windows_system_processor_queue_length[5m])) by (instance)"
	LoadAverage15Expr               = "sum(node_load15 OR avg_over_time(windows_system_processor_queue_length[15m])) by (instance)"
	MemUtilizationExpr              = "1 - sum(node_memory_MemAvailable_bytes OR windows_os_physical_memory_free_bytes) by (instance) / sum(node_memory_MemTotal_bytes OR windows_cs_physical_memory_bytes) by (instance)"
	DiskUtilizationExpr             = "1 - (sum(node_filesystem_free_bytes{device!~\"rootfs|HarddiskVolume.+\"} OR windows_logical_disk_free_bytes{volume!~\"(HarddiskVolume.+|[A-Z]:.+)\"}) by (instance) / sum(node_filesystem_size_bytes{device!~\"rootfs|HarddiskVolume.+\"} OR windows_logical_disk_size_bytes{volume!~\"(HarddiskVolume.+|[A-Z]:.+)\"}) by (instance))"
	DiskReadBytesTotalExpr          = "sum(rate(node_disk_read_bytes_total[$__rate_interval]) OR rate(windows_logical_disk_read_bytes_total[$__rate_interval])) by (instance)"
	DiskWriteBytesTotalExpr         = "sum(rate(node_disk_written_bytes_total[$__rate_interval]) OR rate(windows_logical_disk_write_bytes_total[$__rate_interval])) by (instance)"
	NetworkReceiveErrorsTotalExpr   = "sum(rate(node_network_receive_errs_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[$__rate_interval])) by (instance) OR sum(rate(windows_net_packets_received_errors_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[$__rate_interval])) by (instance)"
	NetworkReceivePacketsTotalExpr  = "sum(rate(node_network_receive_packets_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[$__rate_interval])) by (instance) OR sum(rate(windows_net_packets_received_total_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[$__rate_interval])) by (instance)"
	NetworkTransmitErrorsTotalExpr  = "sum(rate(node_network_transmit_errs_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[$__rate_interval])) by (instance) OR sum(rate(windows_net_packets_outbound_errors_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[$__rate_interval])) by (instance)"
	NetworkReceiveDropsTotalExpr    = "sum(rate(node_network_receive_drop_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[$__rate_interval])) by (instance) OR sum(rate(windows_net_packets_received_discarded_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[$__rate_interval])) by (instance)"
	NetworkTransmitDropsTotalExpr   = "sum(rate(node_network_transmit_drop_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[$__rate_interval])) by (instance) OR sum(rate(windows_net_packets_outbound_discarded{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[$__rate_interval])) by (instance)"
	NetworkTransmitPacketsTotalExpr = "sum(rate(node_network_transmit_packets_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[$__rate_interval])) by (instance) OR sum(rate(windows_net_packets_sent_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[$__rate_interval])) by (instance)"
	NetworkTransmitBytesTotalExpr   = "sum(rate(node_network_transmit_bytes_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[$__rate_interval]) OR rate(windows_net_packets_sent_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[$__rate_interval])) by (instance)"
	NetworkReceiveBytesTotalExpr    = "sum(rate(node_network_receive_bytes_total{device!~\"lo|veth.*|docker.*|flannel.*|cali.*|cbr.*\"}[$__rate_interval]) OR rate(windows_net_packets_received_total_total{nic!~'.*isatap.*|.*VPN.*|.*Pseudo.*|.*tunneling.*'}[$__rate_interval])) by (instance)"
)
