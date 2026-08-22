package domain

import "errors"

type RuleError struct {
	Code    string
	Message string
	Details any
}

func (e *RuleError) Error() string { return e.Message }

func NewRuleError(code, message string) error {
	return &RuleError{Code: code, Message: message}
}

func NewDetailedRuleError(code, message string, details any) error {
	return &RuleError{Code: code, Message: message, Details: details}
}

func ErrorCode(err error) string {
	var rule *RuleError
	if errors.As(err, &rule) {
		return rule.Code
	}
	return "internal_error"
}

var ErrNotFound = NewRuleError("not_found", "请求的资源不存在")
var ErrVersionConflict = NewRuleError("version_conflict", "术语包已被其他操作更新，请刷新后重试")
