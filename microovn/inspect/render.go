package inspect

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/canonical/microovn/microovn/api/types"
)

func RenderJSON(w io.Writer, report types.InspectionReport) error {
	return json.NewEncoder(w).Encode(report)
}

func RenderText(w io.Writer, report types.InspectionReport, verbose bool) error {
	buf := bufio.NewWriter(w)

	_, err := fmt.Fprintf(buf, "Execution: node=%s role=%s scope=%s authoritative=%t\n",
		report.ExecutionContext.LocalNode,
		report.ExecutionContext.MemberRole,
		report.ExecutionContext.Scope,
		report.ExecutionContext.Authoritative)
	if err != nil {
		return err
	}

	if err := renderDatabaseSummary(buf, report.DatabaseSummary); err != nil {
		return err
	}

	for _, result := range report.Results {
		if !verbose && result.Status == types.InspectionStatusPass {
			continue
		}

		if _, err := fmt.Fprintf(buf, "[%s] %s/%s: %s\n", result.Status, result.Category, result.ID, result.Summary); err != nil {
			return err
		}

		if !verbose {
			continue
		}

		for _, detail := range result.Details {
			label := detail.ID
			if detail.Node != "" {
				label = detail.Node + "/" + detail.ID
			}
			if _, err := fmt.Fprintf(buf, "  [%s] %s: %s\n", detail.Status, label, detail.Summary); err != nil {
				return err
			}

			keys := make([]string, 0, len(detail.Data))
			for key := range detail.Data {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if _, err := fmt.Fprintf(buf, "    %s=%s\n", key, detail.Data[key]); err != nil {
					return err
				}
			}
		}

		if result.CollectionError != "" {
			if _, err := fmt.Fprintf(buf, "  Collection error: %s\n", result.CollectionError); err != nil {
				return err
			}
		}
		if result.Remediation != "" {
			if _, err := fmt.Fprintf(buf, "  Remediation: %s\n", result.Remediation); err != nil {
				return err
			}
		}
	}

	_, err = fmt.Fprintf(buf, "Summary: %s (pass=%d warning=%d fail=%d unknown=%d)\n",
		report.Summary.Status,
		report.Summary.Counts.Pass,
		report.Summary.Counts.Warning,
		report.Summary.Counts.Fail,
		report.Summary.Counts.Unknown)
	if err != nil {
		return err
	}

	return buf.Flush()
}

func renderDatabaseSummary(w io.Writer, summary types.InspectionDatabaseSummary) error {
	if _, err := fmt.Fprintln(w, "Database:"); err != nil {
		return err
	}

	if err := renderDatabaseStatus(w, "Northbound", summary.Northbound); err != nil {
		return err
	}
	if err := renderDatabaseStatus(w, "Southbound", summary.Southbound); err != nil {
		return err
	}

	communication := summary.Communication
	if _, err := fmt.Fprintf(w, "  Communication: %s", communication.Status); err != nil {
		return err
	}
	if communication.NBCfg != nil {
		if _, err := fmt.Fprintf(w, " nb_cfg=%d", *communication.NBCfg); err != nil {
			return err
		}
	}
	if communication.SBCfg != nil {
		if _, err := fmt.Fprintf(w, " sb_cfg=%d", *communication.SBCfg); err != nil {
			return err
		}
	}
	if communication.Message != "" {
		if _, err := fmt.Fprintf(w, " message=%q", communication.Message); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}

func renderDatabaseStatus(w io.Writer, name string, status types.InspectionDatabaseStatus) error {
	if _, err := fmt.Fprintf(w, "  %s: %s", name, status.Status); err != nil {
		return err
	}
	if status.ActiveSchemaVersion != "" {
		if _, err := fmt.Fprintf(w, " active_schema=%s", status.ActiveSchemaVersion); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, " responding=%d/%d", status.RespondingMembers, status.ExpectedMembers); err != nil {
		return err
	}
	if status.Message != "" {
		if _, err := fmt.Fprintf(w, " message=%q", status.Message); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}

func ExitCode(report types.InspectionReport) int {
	if report.Summary.Status == types.InspectionStatusPass {
		return 0
	}

	return 1
}
