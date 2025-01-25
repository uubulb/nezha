package model

type Metrics struct {
	ID        uint64 `gorm:"primaryKey"`
	Timestamp uint64
	Server    uint64 `gorm:"index:idx_server_type"`
	Type      uint8  `gorm:"index:idx_server_type"`
	Data      []byte `gorm:"type:blob"`
}

type MetricRecord struct {
	Timestamp    uint64             `json:"timestamp,omitempty"`
	CPU          float64            `json:"cpu,omitempty"`
	MemUsed      uint64             `json:"mem_used,omitempty"`
	DiskUsed     uint64             `json:"disk_used,omitempty"`
	NetInSpeed   uint64             `json:"net_in_speed,omitempty"`
	NetOutSpeed  uint64             `json:"net_out_speed,omitempty"`
	TcpConnCount uint64             `json:"tcp_conn_count,omitempty"`
	UdpConnCount uint64             `json:"udp_conn_count,omitempty"`
	ProcessCount uint64             `json:"process_count,omitempty"`
	Temperature  float64            `json:"temperature,omitempty"`
	GPU          map[string]float64 `json:"gpu,omitempty"`
}
