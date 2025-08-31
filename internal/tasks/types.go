package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	TaskTypeSendEmail      = "send_email"
	TaskTypeGenerateReport = "generate_report"
	TaskTypeProcessOrder   = "process_order"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

var (
	ErrInvalidPayload     = errors.New("invalid payload")
	ErrValidationFailed   = errors.New("validation failed")
	ErrPermanentFailure   = errors.New("permanent failure")
	ErrRetryableFailure   = errors.New("retryable failure")
)

type PermanentError struct {
	Err error
}

func (e PermanentError) Error() string {
	return fmt.Sprintf("permanent error: %v", e.Err)
}

func (e PermanentError) Unwrap() error {
	return e.Err
}

type RetryableError struct {
	Err error
}

func (e RetryableError) Error() string {
	return fmt.Sprintf("retryable error: %v", e.Err)
}

func (e RetryableError) Unwrap() error {
	return e.Err
}

func NewPermanentError(err error) error {
	return PermanentError{Err: err}
}

func NewRetryableError(err error) error {
	return RetryableError{Err: err}
}

type SendEmailPayload struct {
	To      string `json:"to" validate:"required,email"`
	Subject string `json:"subject" validate:"required,max=200"`
	Body    string `json:"body" validate:"required,max=10000"`
}

func (p *SendEmailPayload) Validate() error {
	if p.To == "" {
		return NewPermanentError(fmt.Errorf("%w: to field is required", ErrValidationFailed))
	}
	if p.Subject == "" {
		return NewPermanentError(fmt.Errorf("%w: subject field is required", ErrValidationFailed))
	}
	if p.Body == "" {
		return NewPermanentError(fmt.Errorf("%w: body field is required", ErrValidationFailed))
	}
	if len(p.Subject) > 200 {
		return NewPermanentError(fmt.Errorf("%w: subject too long (max 200 chars)", ErrValidationFailed))
	}
	if len(p.Body) > 10000 {
		return NewPermanentError(fmt.Errorf("%w: body too long (max 10000 chars)", ErrValidationFailed))
	}
	
	// Basic email validation
	if !isValidEmail(p.To) {
		return NewPermanentError(fmt.Errorf("%w: invalid email format", ErrValidationFailed))
	}
	
	return nil
}

type GenerateReportPayload struct {
	ReportID string                 `json:"report_id" validate:"required"`
	Params   map[string]interface{} `json:"params"`
}

type ProcessOrderPayload struct {
	OrderNumber string `json:"order_number" validate:"required"`
	Quantity    int    `json:"quantity" validate:"required,min=1"`
}

func (p *GenerateReportPayload) Validate() error {
	if p.ReportID == "" {
		return NewPermanentError(fmt.Errorf("%w: report_id field is required", ErrValidationFailed))
	}
	return nil
}

func (p *ProcessOrderPayload) Validate() error {
	if p.OrderNumber == "" {
		return NewPermanentError(fmt.Errorf("%w: order_number field is required", ErrValidationFailed))
	}
	if p.Quantity < 1 {
		return NewPermanentError(fmt.Errorf("%w: quantity must be at least 1", ErrValidationFailed))
	}
	return nil
}

func ValidateTaskPayload(taskType string, payload []byte) error {
	switch taskType {
	case TaskTypeSendEmail:
		var p SendEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return NewPermanentError(fmt.Errorf("%w: failed to unmarshal send_email payload: %v", ErrInvalidPayload, err))
		}
		return p.Validate()
		
	case TaskTypeGenerateReport:
		var p GenerateReportPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return NewPermanentError(fmt.Errorf("%w: failed to unmarshal generate_report payload: %v", ErrInvalidPayload, err))
		}
		return p.Validate()
		
	case TaskTypeProcessOrder:
		var p ProcessOrderPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return NewPermanentError(fmt.Errorf("%w: failed to unmarshal process_order payload: %v", ErrInvalidPayload, err))
		}
		return p.Validate()
		
	default:
		return NewPermanentError(fmt.Errorf("%w: unknown task type: %s", ErrInvalidPayload, taskType))
	}
}

func isValidEmail(email string) bool {
	// Very basic email validation - in production use a proper library
	if len(email) < 5 || len(email) > 320 {
		return false
	}
	
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			if atIndex != -1 {
				return false // Multiple @
			}
			atIndex = i
		}
	}
	
	if atIndex <= 0 || atIndex >= len(email)-1 {
		return false // @ at beginning, end, or missing
	}
	
	// Check for dot in domain part
	domain := email[atIndex+1:]
	hasDot := false
	for _, char := range domain {
		if char == '.' {
			hasDot = true
			break
		}
	}
	
	return hasDot
}