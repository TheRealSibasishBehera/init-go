package config

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestIPConfig_MarshalJSON(t *testing.T) {
	jsonData := `{"gateway":"192.168.1.1/24","ip":"192.168.1.10/24"}`
	
	var ipConfig IPConfig
	err := json.Unmarshal([]byte(jsonData), &ipConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal IPConfig: %v", err)
	}

	data, err := json.Marshal(ipConfig)
	if err != nil {
		t.Fatalf("Failed to marshal IPConfig: %v", err)
	}

	expected := `{"gateway":"192.168.1.0/24","ip":"192.168.1.0/24"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestIPConfig_UnmarshalJSON(t *testing.T) {
	jsonData := `{"gateway":"192.168.1.1/24","ip":"192.168.1.10/24"}`
	
	var ipConfig IPConfig
	err := json.Unmarshal([]byte(jsonData), &ipConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal IPConfig: %v", err)
	}

	if ipConfig.IP.String() != "192.168.1.0/24" {
		t.Errorf("Expected IP 192.168.1.0/24, got %s", ipConfig.IP.String())
	}

	if ipConfig.Gateway.String() != "192.168.1.0/24" {
		t.Errorf("Expected Gateway 192.168.1.0/24, got %s", ipConfig.Gateway.String())
	}
}

func TestIPConfig_UnmarshalJSON_InvalidCIDR(t *testing.T) {
	jsonData := `{"ip":"invalid-cidr"}`
	
	var ipConfig IPConfig
	err := json.Unmarshal([]byte(jsonData), &ipConfig)
	if err == nil {
		t.Error("Expected error for invalid CIDR, got nil")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")
	
	configData := `{
		"hostname": "test-host",
		"tty": true,
		"imageConfig": {
			"cmd": ["echo", "hello"],
			"env": ["PATH=/bin:/usr/bin"]
		},
		"ipConfigs": [{
			"ip": "192.168.1.10/24",
			"gateway": "192.168.1.1/24"
		}]
	}`
	
	err := os.WriteFile(configPath, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", config.Hostname)
	}

	if !config.TTY {
		t.Error("Expected TTY to be true")
	}

	if len(config.IPConfigs) != 1 {
		t.Errorf("Expected 1 IP config, got %d", len(config.IPConfigs))
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")
	
	err := os.WriteFile(configPath, []byte("{invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestLoadConfigOrDefault(t *testing.T) {
	config := LoadConfigOrDefault("/nonexistent/path/config.json")
	
	if config.Hostname != "localhost" {
		t.Errorf("Expected default hostname 'localhost', got '%s'", config.Hostname)
	}

	if config.TTY {
		t.Error("Expected default TTY to be false")
	}
}

func TestRunConfig_GetCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   RunConfig
		expected []string
	}{
		{
			name: "ExecOverride takes precedence",
			config: RunConfig{
				ExecOverride: []string{"override", "command"},
				CmdOverride:  "cmd override",
				ImageConfig: &ImageConfig{
					Entrypoint: []string{"entrypoint"},
					Cmd:        []string{"cmd"},
				},
			},
			expected: []string{"override", "command"},
		},
		{
			name: "CmdOverride when no ExecOverride",
			config: RunConfig{
				CmdOverride: "cmd override with args",
				ImageConfig: &ImageConfig{
					Entrypoint: []string{"entrypoint"},
					Cmd:        []string{"cmd"},
				},
			},
			expected: []string{"cmd", "override", "with", "args"},
		},
		{
			name: "Entrypoint + Cmd from ImageConfig",
			config: RunConfig{
				ImageConfig: &ImageConfig{
					Entrypoint: []string{"entrypoint"},
					Cmd:        []string{"cmd", "arg"},
				},
			},
			expected: []string{"entrypoint", "cmd", "arg"},
		},
		{
			name: "Only Cmd from ImageConfig",
			config: RunConfig{
				ImageConfig: &ImageConfig{
					Cmd: []string{"cmd", "arg"},
				},
			},
			expected: []string{"cmd", "arg"},
		},
		{
			name: "Only Entrypoint from ImageConfig",
			config: RunConfig{
				ImageConfig: &ImageConfig{
					Entrypoint: []string{"entrypoint", "arg"},
				},
			},
			expected: []string{"entrypoint", "arg"},
		},
		{
			name:     "No command specified",
			config:   RunConfig{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetCommand()
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Expected %v, got %v", tt.expected, result)
					break
				}
			}
		})
	}
}

func TestRunConfig_GetEnvironment(t *testing.T) {
	config := RunConfig{
		ImageConfig: &ImageConfig{
			Env: []string{"PATH=/bin", "HOME=/root"},
		},
		ExtraEnv: map[string]string{
			"CUSTOM": "value",
			"DEBUG":  "true",
		},
	}

	env := config.GetEnvironment()
	
	if len(env) != 4 {
		t.Errorf("Expected 4 environment variables, got %d", len(env))
	}

	envMap := make(map[string]bool)
	for _, e := range env {
		envMap[e] = true
	}

	expected := []string{"PATH=/bin", "HOME=/root", "CUSTOM=value", "DEBUG=true"}
	for _, exp := range expected {
		if !envMap[exp] {
			t.Errorf("Expected environment variable %s not found", exp)
		}
	}
}

func TestRunConfig_GetUser(t *testing.T) {
	tests := []struct {
		name     string
		config   RunConfig
		expected string
	}{
		{
			name: "UserOverride takes precedence",
			config: RunConfig{
				UserOverride: "override-user",
				ImageConfig: &ImageConfig{
					User: "image-user",
				},
			},
			expected: "override-user",
		},
		{
			name: "ImageConfig user when no override",
			config: RunConfig{
				ImageConfig: &ImageConfig{
					User: "image-user",
				},
			},
			expected: "image-user",
		},
		{
			name:     "Default to root",
			config:   RunConfig{},
			expected: "root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetUser()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRunConfig_GetWorkingDir(t *testing.T) {
	tests := []struct {
		name     string
		config   RunConfig
		expected string
	}{
		{
			name: "ImageConfig working directory",
			config: RunConfig{
				ImageConfig: &ImageConfig{
					WorkingDir: "/app",
				},
			},
			expected: "/app",
		},
		{
			name:     "Default to root",
			config:   RunConfig{},
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetWorkingDir()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRunConfig_GetRootDevice(t *testing.T) {
	tests := []struct {
		name     string
		config   RunConfig
		expected string
	}{
		{
			name: "Custom root device",
			config: RunConfig{
				RootDevice: "/dev/custom",
			},
			expected: "/dev/custom",
		},
		{
			name:     "Default root device",
			config:   RunConfig{},
			expected: "/dev/vdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetRootDevice()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRunConfig_GetNameservers(t *testing.T) {
	config := RunConfig{
		EtcResolv: &EtcResolv{
			Nameservers: []string{"8.8.8.8", "8.8.4.4", "invalid-ip"},
		},
	}

	nameservers := config.GetNameservers()
	
	if len(nameservers) != 2 {
		t.Errorf("Expected 2 valid nameservers, got %d", len(nameservers))
	}

	expected := []string{"8.8.8.8", "8.8.4.4"}
	for i, ns := range nameservers {
		if ns.String() != expected[i] {
			t.Errorf("Expected nameserver %s, got %s", expected[i], ns.String())
		}
	}
}

func TestRunConfig_GetHosts(t *testing.T) {
	config := RunConfig{
		EtcHosts: []EtcHost{
			{Host: "localhost", IP: "127.0.0.1"},
			{Host: "example.com", IP: "192.168.1.1"},
			{Host: "invalid", IP: "not-an-ip"},
		},
	}

	hosts := config.GetHosts()
	
	if len(hosts) != 2 {
		t.Errorf("Expected 2 valid hosts, got %d", len(hosts))
	}

	if hosts["localhost"].String() != "127.0.0.1" {
		t.Errorf("Expected localhost to be 127.0.0.1, got %s", hosts["localhost"].String())
	}

	if hosts["example.com"].String() != "192.168.1.1" {
		t.Errorf("Expected example.com to be 192.168.1.1, got %s", hosts["example.com"].String())
	}
}

func TestRunConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      RunConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config",
			config: RunConfig{
				ImageConfig: &ImageConfig{
					Cmd: []string{"echo", "hello"},
				},
				IPConfigs: []IPConfig{
					{IP: mustParseCIDR("192.168.1.10/24")},
				},
				Mounts: []Mount{
					{MountPath: "/data", DevicePath: "/dev/sdb"},
				},
				EtcHosts: []EtcHost{
					{Host: "localhost", IP: "127.0.0.1"},
				},
				EtcResolv: &EtcResolv{
					Nameservers: []string{"8.8.8.8"},
				},
			},
			expectError: false,
		},
		{
			name:        "No command specified",
			config:      RunConfig{},
			expectError: true,
			errorMsg:    "no command specified to run",
		},
		{
			name: "IP config missing IP",
			config: RunConfig{
				ImageConfig: &ImageConfig{Cmd: []string{"echo"}},
				IPConfigs:   []IPConfig{{}},
			},
			expectError: true,
			errorMsg:    "IP config 0: IP address is required",
		},
		{
			name: "Mount missing mountPath",
			config: RunConfig{
				ImageConfig: &ImageConfig{Cmd: []string{"echo"}},
				Mounts:      []Mount{{DevicePath: "/dev/sdb"}},
			},
			expectError: true,
			errorMsg:    "mount 0: mountPath is required",
		},
		{
			name: "Mount missing devicePath",
			config: RunConfig{
				ImageConfig: &ImageConfig{Cmd: []string{"echo"}},
				Mounts:      []Mount{{MountPath: "/data"}},
			},
			expectError: true,
			errorMsg:    "mount 0: devicePath is required",
		},
		{
			name: "EtcHost missing host",
			config: RunConfig{
				ImageConfig: &ImageConfig{Cmd: []string{"echo"}},
				EtcHosts:    []EtcHost{{IP: "127.0.0.1"}},
			},
			expectError: true,
			errorMsg:    "etcHosts 0: host is required",
		},
		{
			name: "EtcHost invalid IP",
			config: RunConfig{
				ImageConfig: &ImageConfig{Cmd: []string{"echo"}},
				EtcHosts:    []EtcHost{{Host: "localhost", IP: "invalid"}},
			},
			expectError: true,
			errorMsg:    "etcHosts 0: invalid IP address invalid",
		},
		{
			name: "EtcResolv invalid nameserver",
			config: RunConfig{
				ImageConfig: &ImageConfig{Cmd: []string{"echo"}},
				EtcResolv: &EtcResolv{
					Nameservers: []string{"invalid-ip"},
				},
			},
			expectError: true,
			errorMsg:    "etcResolv nameserver 0: invalid IP address invalid-ip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestRunConfig_ToJSON(t *testing.T) {
	config := RunConfig{
		Hostname: "test-host",
		TTY:      true,
	}

	jsonStr, err := config.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert config to JSON: %v", err)
	}

	var parsed RunConfig
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse generated JSON: %v", err)
	}

	if parsed.Hostname != config.Hostname {
		t.Errorf("Expected hostname %s, got %s", config.Hostname, parsed.Hostname)
	}

	if parsed.TTY != config.TTY {
		t.Errorf("Expected TTY %v, got %v", config.TTY, parsed.TTY)
	}
}

func mustParseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return ipNet
}