package record

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"slices"
	"strings"

	"github.com/lunixbochs/struc"

	"github.com/nezhahq/nezha/model"
)

type nezhaMetric struct {
	CPU          float64
	MemUsed      uint64
	DiskUsed     uint64
	NetInSpeed   uint64
	NetOutSpeed  uint64
	TcpConnCount uint64
	UdpConnCount uint64
	ProcessCount uint64
	Temperature  float64
	GPULen       uint64 `struc:"sizeof=GPU"`
	GPU          [][12]byte
}

func fromHostState(host *model.Host, stat *model.HostState) *nezhaMetric {
	gpu := encodeGPU(host.GPU, stat.GPU)

	return &nezhaMetric{
		CPU:          stat.CPU,
		MemUsed:      stat.MemUsed,
		DiskUsed:     stat.DiskUsed,
		NetInSpeed:   stat.NetInSpeed,
		NetOutSpeed:  stat.NetOutSpeed,
		TcpConnCount: stat.TcpConnCount,
		UdpConnCount: stat.UdpConnCount,
		ProcessCount: stat.ProcessCount,
		Temperature: slices.MaxFunc(stat.Temperatures, func(a, b model.SensorTemperature) int {
			return cmp.Compare(a.Temperature, b.Temperature)
		}).Temperature,
		GPULen: uint64(len(gpu)),
		GPU:    gpu,
	}
}

func fromBytes(b []byte) (*nezhaMetric, error) {
	reader := bytes.NewReader(b)
	nzm := &nezhaMetric{}
	err := struc.Unpack(reader, nzm)
	if err != nil {
		return nil, err
	}

	return nzm, nil
}

func (m *nezhaMetric) Pack(w io.Writer) error {
	return struc.Pack(w, m)
}

func encodeGPU(k []string, v []float64) [][12]byte {
	if len(v) < 1 || len(k) < len(v) {
		return nil
	}

	buf := make([][12]byte, 0, len(v))
	w := bytes.NewBuffer(make([]byte, 0, 12))
	for i, metric := range v {
		modelID := k[i]

		parts := strings.SplitN(modelID, ",", 2)
		if len(parts) < 2 {
			continue
		}

		w.Reset()
		sum := crc32.ChecksumIEEE([]byte(parts[1]))

		var w bytes.Buffer
		binary.Write(&w, binary.BigEndian, sum)
		binary.Write(&w, binary.BigEndian, metric)

		b := w.Bytes()
		if len(b) != 12 {
			continue
		}

		var data [12]byte
		copy(data[:], b)
		buf = append(buf, data)
	}

	return buf
}

func decodeGPU(data [][12]byte, gpuIDs []string) (map[string]float64, error) {
	if len(data) == 0 {
		return nil, nil
	}

	hashToID := make(map[uint32]string, len(gpuIDs))
	for _, fullID := range gpuIDs {
		parts := strings.SplitN(fullID, ",", 2)
		if len(parts) < 2 {
			continue
		}
		id := parts[1]
		hash := crc32.ChecksumIEEE([]byte(id))
		hashToID[hash] = id
	}

	result := make(map[string]float64)
	for i, item := range data {
		if len(item) != 12 {
			return nil, fmt.Errorf("invalid data length at index %d: got %d, want 12", i, len(item))
		}

		hash := binary.BigEndian.Uint32(item[:4])
		metric := math.Float64frombits(binary.BigEndian.Uint64(item[4:]))

		if originalID, ok := hashToID[hash]; ok {
			result[originalID] = metric
		}
	}

	return result, nil
}
