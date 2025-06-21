package system

import (
	"bufio"
	"fmt"
	cpu "github.com/shirou/gopsutil/v3/cpu"
	load "github.com/shirou/gopsutil/v3/load"
	mem "github.com/shirou/gopsutil/v3/mem"
	net "github.com/shirou/gopsutil/v3/net"
	"os"
	"strconv"
	"strings"
)

type SystemInfo struct {
	Memory         *Memory         `json:"memory,omitempty"`
	NetworkDevices []NetworkDevice `json:"net,omitempty"`
	Cpus           map[int]*Cpu    `json:"cpus,omitempty"`
	LoadAvg        *load.AvgStat   `json:"load_average,omitempty"`
	FileFd         *FileFd         `json:"filefd,omitempty"`
}

type Memory struct {
	MemTotal     uint64  `json:"mem_total"`
	MemFree      uint64  `json:"mem_free"`
	MemAvailable *uint64 `json:"mem_available,omitempty"`
	Buffers      uint64  `json:"buffers"`
	Cached       uint64  `json:"cached"`
	SwapCached   uint64  `json:"swap_cached"`
	Active       uint64  `json:"active"`
	Inactive     uint64  `json:"inactive"`
	SwapTotal    uint64  `json:"swap_total"`
	SwapFree     uint64  `json:"swap_free"`
	Dirty        uint64  `json:"dirty"`
	Writeback    uint64  `json:"writeback"`
	Slab         uint64  `json:"slab"`
	Shmem        *uint64 `json:"shmem,omitempty"`
	VmallocTotal uint64  `json:"vmalloc_total"`
	VmallocUsed  uint64  `json:"vmalloc_used"`
	VmallocChunk uint64  `json:"vmalloc_chunk"`
}

type NetworkDevice struct {
	Name           string  `json:"name"`
	RecvBytes      uint64  `json:"recv_bytes"`
	RecvPackets    uint64  `json:"recv_packets"`
	RecvErrs       uint64  `json:"recv_errs"`
	RecvDrop       uint64  `json:"recv_drop"`
	RecvFifo       uint64  `json:"recv_fifo"`
	RecvFrame      *uint64 `json:"recv_frame,omitempty"`
	RecvCompressed *uint64 `json:"recv_compressed,omitempty"`
	RecvMulticast  *uint64 `json:"recv_multicast,omitempty"`
	SentBytes      uint64  `json:"sent_bytes"`
	SentPackets    uint64  `json:"sent_packets"`
	SentErrs       uint64  `json:"sent_errs"`
	SentDrop       uint64  `json:"sent_drop"`
	SentFifo       uint64  `json:"sent_fifo"`
	SentColls      *uint64 `json:"sent_colls,omitempty"`
	SentCarrier    *uint64 `json:"sent_carrier,omitempty"`
	SentCompressed *uint64 `json:"sent_compressed,omitempty"`
}

type FileFd struct {
	Allocated uint64 `json:"allocated"`
	Maximum   uint64 `json:"maximum"`
}

type Cpu struct {
	User      float32  `json:"user"`
	Nice      float32  `json:"nice"`
	System    float32  `json:"system"`
	Idle      float32  `json:"idle"`
	Iowait    *float32 `json:"iowait,omitempty"`
	Irq       *float32 `json:"irq,omitempty"`
	Softirq   *float32 `json:"softirq,omitempty"`
	Steal     *float32 `json:"steal,omitempty"`
	Guest     *float32 `json:"guest,omitempty"`
	GuestNice *float32 `json:"guest_nice,omitempty"`
}

// CollectSystemInfo collects system information including memory, network devices, CPU stats, load average, and file descriptors.
func CollectSystemInfo() (*SystemInfo, error) {
	memory, err := collectMemoryInfo()
	if err != nil {
		return nil, err
	}

	networkDevices, err := collectNetworkDevices()
	if err != nil {
		return nil, err
	}

	cpus, err := collectCpuInfo()
	if err != nil {
		return nil, err
	}

	loadAvg, err := collectLoadAvg()
	if err != nil {
		return nil, err
	}

	fileFd, err := collectFileFd()
	if err != nil {
		return nil, err
	}

	systemInfo := &SystemInfo{
		Memory:         memory,
		NetworkDevices: networkDevices,
		Cpus:           cpus,
		LoadAvg:        loadAvg,
		FileFd:         fileFd,
	}

	return systemInfo, nil
}

// collectMemoryInfo collects memory statistics from the system using gopsutil.
func collectMemoryInfo() (*Memory, error) {
	virtualMem, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to collect memory info: %v", err)
	}
	memory := &Memory{
		MemTotal:     virtualMem.Total,
		MemFree:      virtualMem.Free,
		MemAvailable: &virtualMem.Available,
		Buffers:      virtualMem.Buffers,
		Cached:       virtualMem.Cached,
		SwapCached:   virtualMem.SwapCached,
		Active:       virtualMem.Active,
		Inactive:     virtualMem.Inactive,
		SwapTotal:    virtualMem.SwapTotal,
		SwapFree:     virtualMem.SwapFree,
		Dirty:        virtualMem.Dirty,
		Writeback:    virtualMem.WriteBack,
		Slab:         virtualMem.Slab,
		VmallocTotal: virtualMem.VmallocTotal,
		VmallocUsed:  virtualMem.VmallocUsed,
		VmallocChunk: virtualMem.VmallocChunk,
	}
	return memory, nil
}

// collectNetworkDevices collects network device statistics from the system.
func collectNetworkDevices() ([]NetworkDevice, error) {
	netDevices, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("failed to collect network device info: %v", err)
	}

	var devices []NetworkDevice
	for _, dev := range netDevices {
		// Skip loopback interface like Rust version
		if dev.Name == "lo" {
			continue
		}

		device := NetworkDevice{
			Name:        dev.Name,
			RecvBytes:   dev.BytesRecv,
			RecvPackets: dev.PacketsRecv,
			RecvErrs:    dev.Errin,
			RecvDrop:    dev.Dropin,
			RecvFifo:    dev.Fifoin,
			// RecvFrame, RecvCompressed, RecvMulticast not available in gopsutil - will be omitted
			SentBytes:   dev.BytesSent,
			SentPackets: dev.PacketsSent,
			SentErrs:    dev.Errout,
			SentDrop:    dev.Dropout,
			SentFifo:    dev.Fifoout,
			// SentColls, SentCarrier, SentCompressed not available in gopsutil - will be omitted
		}
		devices = append(devices, device)
	}
	return devices, nil
}

func collectCpuInfo() (map[int]*Cpu, error) {
	cpuStats, err := cpu.Times(true)
	if err != nil {
		return nil, fmt.Errorf("failed to collect CPU info: %v", err)
	}

	cpus := make(map[int]*Cpu)
	for i, cpuStat := range cpuStats {
		cpu := &Cpu{
			User:      float32(cpuStat.User),
			Nice:      float32(cpuStat.Nice),
			System:    float32(cpuStat.System),
			Idle:      float32(cpuStat.Idle),
			Iowait:    toFloat32Ptr(cpuStat.Iowait),
			Irq:       toFloat32Ptr(cpuStat.Irq),
			Softirq:   toFloat32Ptr(cpuStat.Softirq),
			Steal:     toFloat32Ptr(cpuStat.Steal),
			Guest:     toFloat32Ptr(cpuStat.Guest),
			GuestNice: toFloat32Ptr(cpuStat.GuestNice),
		}
		cpus[i] = cpu
	}
	return cpus, nil
}

func collectLoadAvg() (*load.AvgStat, error) {
	loadAvg, err := load.Avg()
	if err != nil {
		return nil, fmt.Errorf("failed to collect load average: %v", err)
	}
	return loadAvg, nil
}

// collectFileFd collects file descriptor statistics from the system.
// It reads from /proc/sys/fs/file-nr to get the number of allocated and maximum file descriptors.
// the file-nr file contains three fields: [allocated, free, maximum]
func collectFileFd() (*FileFd, error) {
	file, err := os.Open("/proc/sys/fs/file-nr")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 {
			allocated, _ := strconv.ParseUint(fields[0], 10, 64)
			maximum, _ := strconv.ParseUint(fields[2], 10, 64)

			return &FileFd{
				Allocated: allocated,
				Maximum:   maximum,
			}, nil
		}
	}
	return nil, fmt.Errorf("failed to parse /proc/sys/fs/file-nr")
}

// Helper function to convert float64 to *float32
func toFloat32Ptr(val float64) *float32 {
	f := float32(val)
	return &f
}
