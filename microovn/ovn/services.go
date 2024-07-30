package ovn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/canonical/microcluster/state"
	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/node"
)

type SrvName string

const (
	SrvChassis SrvName = "chassis"
	SrvCentral SrvName = "central"
	SrvSwitch  SrvName = "switch"
)

var ServiceNames = []SrvName{SrvChassis, SrvCentral, SrvSwitch}

func CheckValidService(service string) bool {
	return slices.Contains(ServiceNames, SrvName(service))
}

func DisableService(s *state.State, service string) error {
	exists, err := node.HasServiceActive(s, service)

	if err != nil {
		return err
	}
	if !exists {
		return errors.New("This service is not enabled")
	}

	if SrvName(service) == SrvCentral {
		err = snapStop("ovn-ovsdb-server-nb", true)
		if err != nil {
			return err
		}
		err = snapStop("ovn-ovsdb-server-sb", true)
		if err != nil {
			return err
		}
		err = snapStop("ovn-northd", true)
	} else {
		err = snapStop(service, true)
	}

	if err != nil {
		return fmt.Errorf("Snapctl error, likely due to service not existing:\n %w", err)
	}

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		err := database.DeleteService(ctx, tx, s.Name(), service)
		return err
	})

	return err

}

func EnableService(s *state.State, service string) error {
	exists, err := node.HasServiceActive(s, service)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("This Service is already enabled")
	}

	if !CheckValidService(service) {
		return errors.New("Service does not exist")
	}
	err = snapStart(service, true)
	if err != nil {
		return fmt.Errorf("Snapctl error, likely due to service not existing:\n%w", err)
	}

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		_, err := database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: service})
		return err
	})

	return err

}
