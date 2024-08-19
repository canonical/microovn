package bgp

import (
	"errors"

	"github.com/canonical/microovn/microovn/snap"
	"github.com/zitadel/logging"
)

const FrrBgpService = "frr-bgp"
const FrrZebraService = "frr-zebra"

func EnableService() error {
	err := snap.Start(FrrZebraService, true)
	if err != nil {
		logging.Errorf("Failed to start %s service: %s", FrrZebraService, err)
		return errors.New("failed to start zebra service")
	}

	err = snap.Start(FrrBgpService, true)
	if err != nil {
		logging.Errorf("Failed to start %s service: %s", FrrBgpService, err)
		return errors.New("failed to start BGP service")
	}
	return nil
}

func DisableService() error {
	var allErrors error

	err := snap.Stop(FrrZebraService, true)
	if err != nil {
		logging.Warnf("Failed to stop %s service: %s", FrrZebraService, err)
		allErrors = errors.Join(allErrors, errors.New("failed to stop zebra service"))
	}

	err = snap.Stop(FrrBgpService, true)
	if err != nil {
		logging.Warnf("Failed to stop %s service: %s", FrrBgpService, err)
		allErrors = errors.Join(allErrors, errors.New("failed to stop BGP service"))
	}

	return allErrors
}
