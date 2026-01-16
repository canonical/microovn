package dpu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v3/state"

	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
)

// minimal json structs for devlink
type DevlinkPort struct {
	Flavour    string `json:"flavour"`
	Controller int    `json:"controller"`
}

type DevlinkJSON struct {
	Port map[string]DevlinkPort `json:"port"`
}

// detectDPU uses devlink to find if microovn is running on a DPU and if so
// returns the PCI address
func detectDPU(ctx context.Context) (string, error) {
	out, err := shared.RunCommandContext(ctx, "devlink", "port", "show", "-jp")
	if err != nil {
		return "", fmt.Errorf("failed to run devlink: %w", err)
	}

	var devlink DevlinkJSON
	if err := json.Unmarshal([]byte(out), &devlink); err != nil {
		return "", fmt.Errorf("failed to parse devlink JSON: %w", err)
	}

	pci := findDPUDevlink(devlink)
	if pci != "" {
		logger.Infof("DPU detected at PCI %s", pci)
		return pci, nil
	}
	return "", nil
}

// parseDPUFromDevlink inspects the devlink json from devlink port show and
// returns the PCI address of a port we expect to be a DPU.
func findDPUDevlink(devlinkOutput DevlinkJSON) string {
	for key, port := range devlinkOutput.Port {
		// local controller ports (dpu side) will have controller to be 0, this
		// however does not guarentee we are dealing with a local controller
		// port on the dpu side, which is why we check the port flavour.
		if port.Controller == 0 && port.Flavour == "pcipf" {
			pci := strings.Split(key, "/")[1]
			return pci
		}
	}
	return ""
}

// parseDPUSerial parses lspci output and extracts the DPU serial number.
func parseLspciSerial(lspciOutput string) string {
	for _, line := range strings.Split(lspciOutput, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[SN] Serial number:") {
			serial := strings.SplitN(line, ":", 2)[1]
			serial = strings.TrimSpace(serial)
			return serial
		}
	}

	return ""
}

// getSerialNumber parses lspci and gets the serial number for the DPU, given a
// specific PCI address.
func getDPUSerial(ctx context.Context, pci string) (string, error) {
	out, err := shared.RunCommandContext(ctx, "lspci", "-vvv", "-s", pci)
	if err != nil {
		return "", fmt.Errorf("failed to run lspci -vvv -s %s: %w", pci, err)
	}

	serial := parseLspciSerial(out)
	if serial != "" {
		logger.Infof("DPU serial number detected: %s", serial)
		return serial, nil
	}

	return "", fmt.Errorf("serial number not found for DPU at PCI %s", pci)
}

// setDPUSerial writes the external-id the ovn-cms-options.
func setOVNDPUSerial(ctx context.Context, s state.State, serial string) error {
	out, err := ovnCmd.VSCtl(ctx, s, "get", "Open_vSwitch", ".", "external-ids:ovn-cms-options")
	if err != nil && !strings.Contains(fmt.Sprintf("%v", err), "ovs-vsctl: no key") {
		return err
	}
	externalIDs := strings.Join([]string{
		out,
		fmt.Sprintf("ovn-cms-options=card-serial-number=%s", serial),
	}, ",")
	_, err = ovnCmd.VSCtl(
		ctx,
		s,
		"set", "Open_vSwitch", ".",
		fmt.Sprintf("external-ids:%s", externalIDs),
	)

	if err != nil {
		return fmt.Errorf("ovs-vsctl failed: %w", err)
	}
	return nil
}

// DPUSetup checks if microovn is running on a DPU and if so it extracts the
// serial number and puts it into the ovn-cms-option for card-serial-numbers.
func DPUSetup(ctx context.Context, s state.State) error {
	pci, err := detectDPU(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			logger.Warn("lspci not found; skipping DPU setup")
			return nil
		}

		// real failure
		return err
	}

	if pci == "" {
		return nil
	}

	SN, err := getDPUSerial(ctx, pci)
	if err != nil {
		return err
	}

	return setOVNDPUSerial(ctx, s, SN)
}
