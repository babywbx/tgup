package xerrors

// Code identifies app-owned error categories.
type Code string

const (
	CodeConfig       Code = "config_error"
	CodeScan         Code = "scan_error"
	CodeUpload       Code = "upload_error"
	CodeAuth         Code = "auth_error"
	CodeState        Code = "state_error"
	CodeSecurity     Code = "security_error"
	CodeMCP          Code = "mcp_error"
	CodeRetryable    Code = "retryable_error"
	CodeNonRetryable Code = "non_retryable_error"
	CodeInterrupted  Code = "interrupted_error"
)
