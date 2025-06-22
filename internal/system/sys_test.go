package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCollectSystemInfo(t *testing.T) {
	sysInfo, err := CollectSystemInfo()
	if err != nil {
		t.Skipf("CollectSystemInfo failed (likely non-Linux platform): %v", err)
	}

	if sysInfo == nil {
		t.Fatal("SystemInfo is nil")
	}

	if sysInfo.Memory == nil {
		t.Error("Memory info is nil")
	}

	if sysInfo.LoadAvg == nil {
		t.Error("LoadAvg info is nil")
	}

	if sysInfo.Cpus == nil {
		t.Error("CPU info is nil")
	}

	if sysInfo.NetworkDevices == nil {
		t.Error("NetworkDevices is nil")
	}

	if sysInfo.FileFd == nil {
		t.Error("FileFd info is nil")
	}
}

func TestCollectMemoryInfo(t *testing.T) {
	memory, err := collectMemoryInfo()
	if err != nil {
		t.Fatalf("collectMemoryInfo failed: %v", err)
	}

	if memory == nil {
		t.Fatal("Memory is nil")
	}

	if memory.MemTotal == 0 {
		t.Error("MemTotal should be greater than 0")
	}

	if memory.MemAvailable == nil {
		t.Error("MemAvailable should not be nil")
	}

	if memory.MemFree > memory.MemTotal {
		t.Error("MemFree should not be greater than MemTotal")
	}

	if memory.MemAvailable != nil && *memory.MemAvailable > memory.MemTotal {
		t.Error("MemAvailable should not be greater than MemTotal")
	}
}

func TestCollectNetworkDevices(t *testing.T) {
	devices, err := collectNetworkDevices()
	if err != nil {
		t.Fatalf("collectNetworkDevices failed: %v", err)
	}

	if devices == nil {
		t.Fatal("NetworkDevices is nil")
	}

	for _, device := range devices {
		if device.Name == "lo" {
			t.Error("Loopback interface 'lo' should be excluded")
		}

		if device.Name == "" {
			t.Error("Device name should not be empty")
		}

		if device.RecvBytes < 0 || device.SentBytes < 0 {
			t.Error("Network byte counts should not be negative")
		}
	}
}

func TestCollectCpuInfo(t *testing.T) {
	cpus, err := collectCpuInfo()
	if err != nil {
		t.Fatalf("collectCpuInfo failed: %v", err)
	}

	if cpus == nil {
		t.Fatal("CPU map is nil")
	}

	if len(cpus) == 0 {
		t.Error("Should have at least one CPU")
	}

	cpu0, exists := cpus[0]
	if !exists {
		t.Error("CPU 0 should exist")
	}

	if cpu0 != nil {
		if cpu0.User < 0 || cpu0.System < 0 || cpu0.Idle < 0 {
			t.Error("CPU times should be non-negative")
		}

		if cpu0.Idle == 0 {
			t.Error("CPU idle time is unexpectedly 0")
		}
	}
}

func TestCollectLoadAvg(t *testing.T) {
	loadAvg, err := collectLoadAvg()
	if err != nil {
		t.Fatalf("collectLoadAvg failed: %v", err)
	}

	if loadAvg == nil {
		t.Fatal("LoadAvg is nil")
	}

	if loadAvg.Load1 < 0 || loadAvg.Load5 < 0 || loadAvg.Load15 < 0 {
		t.Error("Load averages should be non-negative")
	}
}

func TestCollectFileFd_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file-nr")
	
	testData := "1024 512 65536\n"
	err := os.WriteFile(filePath, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	fileFd, err := collectFileFdFromPath(filePath)
	if err != nil {
		t.Fatalf("collectFileFd failed: %v", err)
	}

	if fileFd == nil {
		t.Fatal("FileFd is nil")
	}

	if fileFd.Allocated != 1024 {
		t.Errorf("Expected allocated 1024, got %d", fileFd.Allocated)
	}

	if fileFd.Maximum != 65536 {
		t.Errorf("Expected maximum 65536, got %d", fileFd.Maximum)
	}
}

func TestCollectFileFd_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file-nr")
	
	testData := "invalid data\n"
	err := os.WriteFile(filePath, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	fileFd, err := collectFileFdFromPath(filePath)
	if err == nil {
		t.Error("Expected error for invalid file content, got nil")
	}

	if fileFd != nil {
		t.Error("Expected nil FileFd for invalid content")
	}
}

func TestCollectFileFd_NonExistentFile(t *testing.T) {
	fileFd, err := collectFileFdFromPath("/nonexistent/file")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	if fileFd != nil {
		t.Error("Expected nil FileFd for non-existent file")
	}
}

func TestToFloat32Ptr(t *testing.T) {
	tests := []struct {
		input    float64
		expected float32
	}{
		{0.0, 0.0},
		{1.5, 1.5},
		{-1.5, -1.5},
		{123.456, 123.456},
	}

	for _, test := range tests {
		result := toFloat32Ptr(test.input)
		if result == nil {
			t.Errorf("Expected non-nil pointer for input %f", test.input)
			continue
		}

		if *result != test.expected {
			t.Errorf("Expected %f, got %f", test.expected, *result)
		}
	}
}

func collectFileFdFromPath(filePath string) (*FileFd, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var line string
	_, err = file.Read(make([]byte, 0))
	if err != nil {
		return nil, err
	}

	file.Seek(0, 0)
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	line = string(content)
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, fmt.Errorf("failed to parse file-nr")
	}

	allocated, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse allocated: %v", err)
	}

	maximum, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse maximum: %v", err)
	}

	return &FileFd{
		Allocated: allocated,
		Maximum:   maximum,
	}, nil
}

