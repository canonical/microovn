package netplan

import "testing"

func TestNewConfig_Defaults(t *testing.T) {
	cfg := NewConfig()
	if cfg.Network.Version != supportedVersion {
		t.Errorf("expected version %d, got %d", supportedVersion, cfg.Network.Version)
	}
	if len(cfg.Network.VirtualEthernets) != 0 ||
		len(cfg.Network.Vrfs) != 0 ||
		len(cfg.Network.Bridges) != 0 ||
		len(cfg.Network.OpenvSwitch.ExternalIDs) != 0 {

		t.Errorf("expected empty maps in new config")
	}
	cfg.Network.OpenvSwitch.ExternalIDs["some-string"] = "some-other-string"
}

func TestAddMethods(t *testing.T) {
	cfg := NewConfig()

	if _, ok := cfg.Network.VirtualEthernets["veth413"]; ok {
		t.Errorf("unexpected veth1 in config")
	}
	cfg.AddVeth("veth413", "veth612", "a1:b2:c3:d4:e5:f6", false)
	if _, ok := cfg.Network.VirtualEthernets["veth413"]; !ok {
		t.Errorf("expected veth413 in config")
	}
	if cfg.Network.VirtualEthernets["veth413"].MacAddress != "a1:b2:c3:d4:e5:f6" {
		t.Errorf("mac address does not match expected")
	}
	if cfg.Network.VirtualEthernets["veth413"].Peer != "veth612" {
		t.Errorf("peer does not match expected")
	}

	if cfg.Network.VirtualEthernets["veth413"].AcceptRa {
		t.Errorf("acceptRa is true when false expected")
	}

	cfg.AddVeth("veth612", "veth413", "", true)
	if _, ok := cfg.Network.VirtualEthernets["veth612"]; !ok {
		t.Errorf("expected veth612 in config")
	}

	if _, ok := cfg.Network.VirtualEthernets["veth413"]; !ok {
		t.Errorf("expected veth413 still in config")
	}

	if cfg.Network.VirtualEthernets["veth612"].MacAddress != "" {
		t.Errorf("veth612 mac address is non empty")
	}

	if !cfg.Network.VirtualEthernets["veth612"].AcceptRa {
		t.Errorf("acceptRa is false when true expected")
	}

	if _, ok := cfg.Network.Vrfs["vrf1"]; ok {
		t.Errorf("unexpected vrf1 in config")
	}

	cfg.AddVRF("vrf1", "10", []string{"veth413"})
	if _, ok := cfg.Network.Vrfs["vrf1"]; !ok {
		t.Errorf("expected vrf1 in config")
	}

	if len(cfg.Network.Vrfs["vrf1"].Interfaces) != 1 {
		t.Errorf("expected 1 item in vrf1 Interfaces")
	}

	cfg.AddVRF("vrf1", "10", []string{})
	if len(cfg.Network.Vrfs["vrf1"].Interfaces) != 0 {
		t.Errorf("expected 0 items in vrf1 Interfaces")
	}

	cfg.AddBridge("br-int", []string{"veth2"})

	if _, ok := cfg.Network.Bridges["br-int"]; !ok {
		t.Errorf("expected br-int in config")
	}

	if cfg.Network.Bridges["br-int"].OpenvSwitch.FailMode != "secure" {
		t.Errorf("expected br-int to have failmode secure")
	}

	if len(cfg.Network.Bridges["br-int"].Interfaces) != 1 {
		t.Errorf("expected 1 item in br-int Interfaces")
	}

}
