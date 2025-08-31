package tasks

import (
	"testing"
)

func TestSendEmailPayload_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payload SendEmailPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: SendEmailPayload{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "Test body content",
			},
			wantErr: false,
		},
		{
			name: "empty to field",
			payload: SendEmailPayload{
				To:      "",
				Subject: "Test Subject",
				Body:    "Test body content",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			payload: SendEmailPayload{
				To:      "invalid-email",
				Subject: "Test Subject",
				Body:    "Test body content",
			},
			wantErr: true,
		},
		{
			name: "empty subject",
			payload: SendEmailPayload{
				To:      "test@example.com",
				Subject: "",
				Body:    "Test body content",
			},
			wantErr: true,
		},
		{
			name: "empty body",
			payload: SendEmailPayload{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SendEmailPayload.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateReportPayload_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payload GenerateReportPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: GenerateReportPayload{
				ReportID: "test-report-123",
				Params:   map[string]interface{}{"type": "sales"},
			},
			wantErr: false,
		},
		{
			name: "empty report ID",
			payload: GenerateReportPayload{
				ReportID: "",
				Params:   map[string]interface{}{"type": "sales"},
			},
			wantErr: true,
		},
		{
			name: "nil params",
			payload: GenerateReportPayload{
				ReportID: "test-report-123",
				Params:   nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateReportPayload.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTaskPayload(t *testing.T) {
	tests := []struct {
		name     string
		taskType string
		payload  string
		wantErr  bool
	}{
		{
			name:     "valid send_email",
			taskType: TaskTypeSendEmail,
			payload:  `{"to":"test@example.com","subject":"Test","body":"Hello"}`,
			wantErr:  false,
		},
		{
			name:     "valid generate_report",
			taskType: TaskTypeGenerateReport,
			payload:  `{"report_id":"test-123","params":{"type":"sales"}}`,
			wantErr:  false,
		},
		{
			name:     "invalid task type",
			taskType: "unknown_task",
			payload:  `{"test":"data"}`,
			wantErr:  true,
		},
		{
			name:     "invalid JSON",
			taskType: TaskTypeSendEmail,
			payload:  `invalid json`,
			wantErr:  true,
		},
		{
			name:     "valid process_order",
			taskType: TaskTypeProcessOrder,
			payload:  `{"order_number":"ORD-2024-001","quantity":5}`,
			wantErr:  false,
		},
		{
			name:     "invalid process_order - missing order_number",
			taskType: TaskTypeProcessOrder,
			payload:  `{"quantity":5}`,
			wantErr:  true,
		},
		{
			name:     "invalid process_order - zero quantity",
			taskType: TaskTypeProcessOrder,
			payload:  `{"order_number":"ORD-2024-001","quantity":0}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskPayload(tt.taskType, []byte(tt.payload))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessOrderPayload_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payload ProcessOrderPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: ProcessOrderPayload{
				OrderNumber: "ORD-2024-001",
				Quantity:    5,
			},
			wantErr: false,
		},
		{
			name: "empty order number",
			payload: ProcessOrderPayload{
				OrderNumber: "",
				Quantity:    5,
			},
			wantErr: true,
		},
		{
			name: "zero quantity",
			payload: ProcessOrderPayload{
				OrderNumber: "ORD-2024-001",
				Quantity:    0,
			},
			wantErr: true,
		},
		{
			name: "negative quantity",
			payload: ProcessOrderPayload{
				OrderNumber: "ORD-2024-001",
				Quantity:    -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessOrderPayload.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}