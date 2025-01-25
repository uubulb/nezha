package record

import (
	"bytes"
	"errors"
	"time"

	"github.com/nezhahq/nezha/model"
	"gorm.io/gorm"
)

type Recorder struct {
	db               *gorm.DB
	lastDeletionTime time.Time
}

const (
	Record1M uint8 = iota + 1
	Record10M
	Record20M
	Record60M
	Record120M
	Record480M
)

func NewRecorder(db *gorm.DB) *Recorder {
	return &Recorder{db, time.Now()}
}

func (r *Recorder) Insert(sid uint64, host *model.Host, stat *model.HostState) error {
	return r.insert(Record1M, sid, host, stat)
}

func (r *Recorder) FindMetric(sid uint64, typ uint8, gpuModels []string) ([]*model.MetricRecord, error) {
	var m []model.Metrics
	if err := r.db.Find(&m).Where("server = ?, type = ?", sid, typ).Error; err != nil {
		return nil, err
	}

	if len(m) < 1 {
		return nil, errors.New("no records found")
	}

	needGPU := len(gpuModels) > 0
	records := make([]*model.MetricRecord, 0, len(m))
	for _, r := range m {
		nzm, err := fromBytes(r.Data)
		if err != nil {
			continue
		}

		var gpu map[string]float64
		if needGPU && len(nzm.GPU) > 0 {
			if g, err := decodeGPU(nzm.GPU, gpuModels); err == nil {
				gpu = g
			} else {
				gpu = make(map[string]float64)
			}
		} else {
			gpu = make(map[string]float64)
		}

		rec := &model.MetricRecord{
			Timestamp:    r.Timestamp,
			CPU:          nzm.CPU,
			MemUsed:      nzm.MemUsed,
			DiskUsed:     nzm.DiskUsed,
			NetInSpeed:   nzm.NetInSpeed,
			NetOutSpeed:  nzm.NetOutSpeed,
			TcpConnCount: nzm.TcpConnCount,
			UdpConnCount: nzm.UdpConnCount,
			ProcessCount: nzm.ProcessCount,
			Temperature:  nzm.Temperature,
			GPU:          gpu,
		}
		records = append(records, rec)
	}

	return records, nil
}

// borrowed from https://github.com/henrygd/beszel/blob/main/beszel/internal/records/records.go
func (r *Recorder) CreateLongerRecords() {

}

func (r *Recorder) DeleteOldRecords() {

}

func (r *Recorder) insert(typ uint8, sid uint64, host *model.Host, stat *model.HostState) error {
	ts := uint64(time.Now().Unix())
	var b bytes.Buffer
	if err := fromHostState(host, stat).Pack(&b); err != nil {
		return err
	}

	metric := model.Metrics{
		Timestamp: ts,
		Server:    sid,
		Type:      typ,
		Data:      b.Bytes(),
	}

	return r.db.Create(&metric).Error
}
