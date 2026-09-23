package facts

import (
	"bufio"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/canonical/microovn/microovn/api/types"
)

func CollectNodeSnapshot(
	ctx context.Context,
	name string,
	address string,
	serviceStates func(context.Context) (io.Reader, error),
	environment func() (io.Reader, error),
) types.InspectionNodeSnapshot {
	snapshot := types.InspectionNodeSnapshot{NodeName: name, Address: address}

	rawServices, err := serviceStates(ctx)
	if err != nil {
		snapshot.Errors = append(snapshot.Errors, types.InspectionCollectionError{
			FactGroup: "daemons",
			Message:   err.Error(),
		})
	} else {
		daemons, err := parseServiceStates(rawServices)
		if err != nil {
			snapshot.Errors = append(snapshot.Errors, types.InspectionCollectionError{
				FactGroup: "daemons",
				Message:   err.Error(),
			})
		} else {
			snapshot.Daemons = daemons
		}
	}

	rawEnvironment, err := environment()
	if err != nil {
		snapshot.Errors = append(snapshot.Errors, types.InspectionCollectionError{
			FactGroup: "environment",
			Message:   err.Error(),
		})
	} else {
		env, err := parseEnvironment(rawEnvironment)
		if err != nil {
			snapshot.Errors = append(snapshot.Errors, types.InspectionCollectionError{
				FactGroup: "environment",
				Message:   err.Error(),
			})
		} else {
			snapshot.Environment = env
		}
	}

	return snapshot
}

func parseEnvironment(reader io.Reader) ([]types.InspectionEnvironment, error) {
	scanner := bufio.NewScanner(reader)
	seen := map[string]struct{}{}
	var entries []types.InspectionEnvironment

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" || value == "" {
			continue
		}

		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}

		value, err := strconv.Unquote(value)
		if err != nil {
			continue
		}
		hash := sha256.Sum256([]byte(value))

		entries = append(entries, types.InspectionEnvironment{
			Name: name,
			Hash: hex.EncodeToString(hash[:]),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read environment: %w", err)
	}

	slices.SortFunc(entries, func(a, b types.InspectionEnvironment) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return entries, nil
}

func parseServiceStates(reader io.Reader) ([]types.InspectionDaemonState, error) {
	scanner := bufio.NewScanner(reader)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("invalid 'snapctl services' output")
	}

	if header := strings.Join(strings.Fields(scanner.Text()), " "); header != "Service Startup Current Notes" {
		return nil, fmt.Errorf("invalid header")
	}

	var entries []types.InspectionDaemonState
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		service, err := parseServiceState(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse service state: %w", err)
		}
		// Filter timer-activated service
		if service.Name == "microovn.refresh-expiring-certs" {
			continue
		}
		entries = append(entries, service)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read service states: %w", err)
	}

	slices.SortFunc(entries, func(a, b types.InspectionDaemonState) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return entries, nil
}

func parseServiceState(output string) (types.InspectionDaemonState, error) {
	fields := strings.Fields(output)
	if len(fields) < 3 {
		return types.InspectionDaemonState{}, fmt.Errorf("invalid service state")
	}

	service := types.InspectionDaemonState{Name: fields[0]}
	switch fields[1] {
	case "enabled":
		service.Enabled = true
	case "disabled":
		service.Enabled = false
	default:
		return service, fmt.Errorf("invalid startup state: %s", fields[1])
	}

	switch fields[2] {
	case "active":
		service.Active = true
	case "inactive":
		service.Active = false
	default:
		return service, fmt.Errorf("invalid current state: %s", fields[2])
	}

	return service, nil
}
