// Package securitylog provides structured OWASP security event logging.
//
// Each entry carries "security", "category" (AUTHN/AUTHZ/SYS) and "event"
// fields.  Logging is enabled by default and can be turned off once at
// daemon start-up via SetEnabled(false).
//
// References:
//   - https://cheatsheetseries.owasp.org/cheatsheets/Logging_Vocabulary_Cheat_Sheet.html
//   - https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
package securitylog

import (
	"fmt"

	"github.com/canonical/lxd/shared/logger"
)

// Category is an OWASP security event category
type Category string

const (
	CatAuthn Category = "AUTHN" // authentication
	CatAuthz Category = "AUTHZ" // authorization
	CatSys   Category = "SYS"   // system
)

// Event is an OWASP logging vocabulary event identifier
type Event string

const (
	EventPasswordChanged Event = "authn_password_changed" // certificate (re)issuance
	EventAdminActivity   Event = "authz_admin"            // privileged admin operation
	EventSysStartup      Event = "sys_startup"            // daemon start
	EventSysShutdown     Event = "sys_shutdown"           // daemon shutdown
)

// Log emits a structured security event at INFO level,
// it is a no-op when security logging has been disabled via SetEnabled
func Log(cat Category, evt Event, extra logger.Ctx, format string, args ...any) {
	ctx := logger.Ctx{
		"security": true,
		"category": string(cat),
		"event":    string(evt),
	}

	for k, v := range extra {
		ctx[k] = v
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	logger.Info(msg, ctx)
}
