package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

type RunConfig struct {
	ImageConfig  *ImageConfig      `json:"imageConfig,omitempty"`
	ExecOverride []string          `json:"execOverride,omitempty"`
	ExtraEnv     map[string]string `json:"extraEnv,omitempty"`
	UserOverride string            `json:"userOverride,omitempty"`
	CmdOverride  string            `json:"cmdOverride,omitempty"`
	IPConfigs    []IPConfig        `json:"ipConfigs,omitempty"`
	TTY          bool              `json:"tty"`
	Hostname     string            `json:"hostname,omitempty"`
	Mounts       []Mount           `json:"mounts,omitempty"`
	RootDevice   string            `json:"rootDevice,omitempty"`
	EtcResolv    *EtcResolv        `json:"etcResolv,omitempty"`
	EtcHosts     []EtcHost         `json:"etcHosts,omitempty"`
}

type ImageConfig struct {
	Entrypoint []string `json:"entrypoint,omitempty"`
	Cmd        []string `json:"cmd,omitempty"`
	Env        []string `json:"env,omitempty"`
	WorkingDir string   `json:"workingDir,omitempty"`
	User       string   `json:"user,omitempty"`
}

type IPConfig struct {
	Gateway *net.IPNet `json:"gateway,omitempty"`
	IP      *net.IPNet `json:"ip,omitempty"`
}

type Mount struct {
	MountPath  string `json:"mountPath"`
	DevicePath string `json:"devicePath"`
	FSType     string `json:"fsType,omitempty"`
	Options    string `json:"options,omitempty"`
}

type EtcHost struct {
	Host        string `json:"host"`
	IP          string `json:"ip"`
	Description string `json:"description,omitempty"`
}

type EtcResolv struct {
	Nameservers []string `json:"nameservers,omitempty"`
	Search      []string `json:"search,omitempty"`
	Options     []string `json:"options,omitempty"`
}

func (ip *IPConfig) MarshalJSON() ([]byte, error) {
	type Alias IPConfig
	aux := &struct {
		Gateway string `json:"gateway,omitempty"`
		IP      string `json:"ip,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(ip),
	}
	
	if ip.Gateway != nil {
		aux.Gateway = ip.Gateway.String()
	}
	if ip.IP != nil {
		aux.IP = ip.IP.String()
	}
	
	return json.Marshal(aux)
}

func (ip *IPConfig) UnmarshalJSON(data []byte) error {
	type Alias IPConfig
	aux := &struct {
		Gateway string `json:"gateway,omitempty"`
		IP      string `json:"ip,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(ip),
	}
	
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	
	if aux.Gateway != "" {
		_, gateway, err := net.ParseCIDR(aux.Gateway)
		if err != nil {
			return fmt.Errorf("invalid gateway CIDR %s: %v", aux.Gateway, err)
		}
		ip.Gateway = gateway
	}
	
	if aux.IP != "" {
		_, ipNet, err := net.ParseCIDR(aux.IP)
		if err != nil {
			return fmt.Errorf("invalid IP CIDR %s: %v", aux.IP, err)
		}
		ip.IP = ipNet
	}
	
	return nil
}

func LoadConfig(path string) (*RunConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", path, err)
	}
	
	var config RunConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %v", err)
	}
	
	return &config, nil
}

func LoadConfigOrDefault(path string) *RunConfig {
	config, err := LoadConfig(path)
	if err != nil {
		return &RunConfig{
			TTY:      false,
			Hostname: "localhost",
		}
	}
	return config
}

func (c *RunConfig) GetCommand() []string {
	if len(c.ExecOverride) > 0 {
		return c.ExecOverride
	}
	
	if c.CmdOverride != "" {
		return strings.Fields(c.CmdOverride)
	}
	
	if c.ImageConfig != nil {
		if len(c.ImageConfig.Cmd) > 0 {
			if len(c.ImageConfig.Entrypoint) > 0 {
				result := make([]string, 0, len(c.ImageConfig.Entrypoint)+len(c.ImageConfig.Cmd))
				result = append(result, c.ImageConfig.Entrypoint...)
				result = append(result, c.ImageConfig.Cmd...)
				return result
			}
			return c.ImageConfig.Cmd
		}
		
		if len(c.ImageConfig.Entrypoint) > 0 {
			return c.ImageConfig.Entrypoint
		}
	}
	
	return nil
}

func (c *RunConfig) GetEnvironment() []string {
	var env []string
	
	if c.ImageConfig != nil && len(c.ImageConfig.Env) > 0 {
		env = append(env, c.ImageConfig.Env...)
	}
	
	for key, value := range c.ExtraEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	return env
}

func (c *RunConfig) GetUser() string {
	if c.UserOverride != "" {
		return c.UserOverride
	}
	
	if c.ImageConfig != nil && c.ImageConfig.User != "" {
		return c.ImageConfig.User
	}
	
	return "root"
}

func (c *RunConfig) GetWorkingDir() string {
	if c.ImageConfig != nil && c.ImageConfig.WorkingDir != "" {
		return c.ImageConfig.WorkingDir
	}
	
	return "/"
}

func (c *RunConfig) GetRootDevice() string {
	if c.RootDevice != "" {
		return c.RootDevice
	}
	
	return "/dev/vdb"
}

func (c *RunConfig) GetNameservers() []net.IP {
	if c.EtcResolv == nil {
		return nil
	}
	
	nameservers := make([]net.IP, 0, len(c.EtcResolv.Nameservers))
	for _, ns := range c.EtcResolv.Nameservers {
		if ip := net.ParseIP(ns); ip != nil {
			nameservers = append(nameservers, ip)
		}
	}
	return nameservers
}

func (c *RunConfig) GetHosts() map[string]net.IP {
	hosts := make(map[string]net.IP)
	for _, host := range c.EtcHosts {
		if ip := net.ParseIP(host.IP); ip != nil {
			hosts[host.Host] = ip
		}
	}
	return hosts
}

func (c *RunConfig) Validate() error {
	if cmd := c.GetCommand(); len(cmd) == 0 {
		return fmt.Errorf("no command specified to run")
	}
	
	for i, ipConfig := range c.IPConfigs {
		if ipConfig.IP == nil {
			return fmt.Errorf("IP config %d: IP address is required", i)
		}
	}
	
	for i, mount := range c.Mounts {
		if mount.MountPath == "" {
			return fmt.Errorf("mount %d: mountPath is required", i)
		}
		if mount.DevicePath == "" {
			return fmt.Errorf("mount %d: devicePath is required", i)
		}
	}
	
	for i, host := range c.EtcHosts {
		if host.Host == "" {
			return fmt.Errorf("etcHosts %d: host is required", i)
		}
		if net.ParseIP(host.IP) == nil {
			return fmt.Errorf("etcHosts %d: invalid IP address %s", i, host.IP)
		}
	}
	
	if c.EtcResolv != nil {
		for i, ns := range c.EtcResolv.Nameservers {
			if net.ParseIP(ns) == nil {
				return fmt.Errorf("etcResolv nameserver %d: invalid IP address %s", i, ns)
			}
		}
	}
	
	return nil
}

func (c *RunConfig) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}