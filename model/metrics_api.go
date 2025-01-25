package model

type MetricRecordResponseItem []struct {
	Timestamp uint64 `json:"timestamp,omitempty"`
	Value     any    `json:"value,omitempty"`
}
