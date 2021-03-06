package meta

import (
	"encoding/json"
)

//MicroAPP 一个函数式应用.
type MicroAPP struct {
	Version string
	Host    string
	Port    int
	PID     int
}

//NewMicroAPP 一个应用.
func NewMicroAPP(version, host string, port, pid int) *MicroAPP {
	return &MicroAPP{
		Version: version,
		PID:     pid,
		Host:    host,
		Port:    port,
	}
}

func (m *MicroAPP) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}
