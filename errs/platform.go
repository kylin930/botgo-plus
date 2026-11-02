package errs

import "fmt"

// PlatformError contains only safe response metadata, never response bodies.
// Status is HTTP status; Code is the platform's code/err_code, not HTTP status.
type PlatformError struct {
	Status  int    `json:"http_status"`
	Code    int    `json:"platform_code"`
	TraceID string `json:"trace_id,omitempty"`
}

func (e *PlatformError) Error() string {
	return fmt.Sprintf("QQ OpenAPI failed: http=%d code=%d trace=%s", e.Status, e.Code, e.TraceID)
}
