// Package netplan implements netplan specific functions
package netplan

import (
	"context"
	"fmt"
	"os"

	"github.com/canonical/lxd/shared"
	"gopkg.in/yaml.v3"
)

// SupportedVersion is a const for the version of netplan this supports
const supportedVersion = 2

// Config represents the top-level netplan structure of a yaml file
type Config struct {
	Network network `yaml:"network"`
}

// Network represents the networks the config is defining
type network struct {
	Version          int                        `yaml:"version"`
	VirtualEthernets map[string]virtualEthernet `yaml:"virtual-ethernets,omitempty"`
	Vrfs             map[string]vrf             `yaml:"vrfs,omitempty"`
	Bridges          map[string]bridge          `yaml:"bridges,omitempty"`
}

// VirtualEthernet is a struct for defining virtual ethernets
type virtualEthernet struct {
	Peer       string `yaml:"peer"`
	MacAddress string `yaml:"macaddress,omitempty"`
}

// VRF defines vrf entires in the netplan config
type vrf struct {
	Table      string   `yaml:"table"`
	Interfaces []string `yaml:"interfaces"`
}

// Bridge represents a network bridge
type bridge struct {
	OpenvSwitch *openvSwitch `yaml:"openvswitch,omitempty"`
	Interfaces  []string     `yaml:"interfaces,omitempty"`
}

// OpenvSwitch options for a bridge
type openvSwitch struct {
	FailMode string `yaml:"fail-mode,omitempty"`
}

// NewConfig returns a new Netplan config with default version set.
func NewConfig() *Config {
	return &Config{
		Network: network{
			Version:          supportedVersion,
			VirtualEthernets: make(map[string]virtualEthernet),
			Vrfs:             make(map[string]vrf),
			Bridges:          make(map[string]bridge),
		},
	}
}

// AddVeth adds a veth pair to the config.
func (c *Config) AddVeth(iface string, peer string, mac string) {
	c.Network.VirtualEthernets[iface] = virtualEthernet{
		Peer:       peer,
		MacAddress: mac,
	}
}

// AddVRF adds a VRF with interfaces.
func (c *Config) AddVRF(name string, table string, ifaces []string) {
	c.Network.Vrfs[name] = vrf{
		Table:      table,
		Interfaces: ifaces,
	}
}

// AddBridge adds a bridge with interfaces.
func (c *Config) AddBridge(name string, ifaces []string) {
	c.Network.Bridges[name] = bridge{
		OpenvSwitch: &openvSwitch{FailMode: "secure"},
		Interfaces:  ifaces,
	}
}

// CleaupVirtualEthernets cleansup virtual ethernets represented by this config
func (c *Config) CleanupVirtualEthernets(ctx context.Context) error {
	deletedPeers := map[string]bool{}
	for iface, ifaceData := range c.Network.VirtualEthernets {
		if deletedPeers[iface] {
			continue
		}
		_, err := shared.RunCommandContext(ctx, "ip", "link", "delete", "dev", iface)
		if err != nil {
			return fmt.Errorf("failed remove interface '%s': %v", iface, err)
		} else {
			deletedPeers[ifaceData.Peer] = true
		}
	}
	return nil
}

// CleanupVRFs cleans up the vrfs represented in the config, and will delete them
// depending on the "delete" argument
func (c *Config) CleanupVRFs(ctx context.Context, delete bool) error {
	for vrf, vrfData := range c.Network.Vrfs {
		_, err := shared.RunCommandContext(ctx, "ip", "route", "flush", "table", vrfData.Table)
		if err != nil {
			return fmt.Errorf("failed to flush vrf '%s': %v", vrf, err)
		}
		if delete {
			_, err = shared.RunCommandContext(ctx, "ip", "link ", "delete", "dev", vrf)
			if err != nil {
				return fmt.Errorf("failed to delete vrf '%s': %v", vrf, err)
			}
		}
	}
	return nil
}

// WriteToNetplan writes a file and then moves it to the netplan directory, due
// to permissions issues writing to /etc/netplan directly fails
func WriteToNetplan(ctx context.Context, filename string, config Config) error {
	data, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	filepath := fmt.Sprintf("/tmp/%s", filename)
	err = os.WriteFile(filepath, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	_, err = shared.RunCommandContext(ctx, "mv", filepath, "/etc/netplan")
	return err
}

// Apply uses dbus to trigger the netplan apply command
func Apply(ctx context.Context) error {
	_, err := shared.RunCommandContext(ctx, "dbus-send", "--system", "--type=method_call", "--print-reply", "--dest=io.netplan.Netplan", "/io/netplan/Netplan", "io.netplan.Netplan.Apply")
	return err
}

// LoadConfig reads a netplan file and returns a Config object representing the file
func LoadConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", filename, err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return &cfg, nil
}

func Cleanup(ctx context.Context, filename string) error {
	filepath := fmt.Sprintf("/etc/netplan/%s", filename)
	tmpfilepath := fmt.Sprintf("/tmp/%s", filename)
	_, err := shared.RunCommandContext(ctx, "mv", filepath, tmpfilepath)
	if err != nil {
		return fmt.Errorf("failed to move netplan config: %v", err)
	}

	netplanFile, err := LoadConfig(tmpfilepath)
	if err != nil {
		return fmt.Errorf("cannot read netplan config: %v", err)
	} else {
		err = netplanFile.CleanupVirtualEthernets(ctx)
		if err != nil {
			return fmt.Errorf("virtual ethernets cleanup failed: %v", err)
		}
		err = netplanFile.CleanupVRFs(ctx, false)
		if err != nil {
			return fmt.Errorf("VRF cleanup failed: %v", err)
		}
	}

	_, err = shared.RunCommandContext(ctx, "rm", tmpfilepath)
	if err != nil {
		return fmt.Errorf("failed to delete netplan config: %v", err)
	}

	return Apply(ctx)
}
