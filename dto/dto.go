package dto

import "github.com/google/uuid"

// FlowDTO represents the data transfer object for a flow.
type FlowDTO struct {
	ID                uuid.UUID             `json:"id"`
	Name              string                `json:"name"`
	Description       string                `json:"description"`
	Active            bool                  `json:"active"`
	Processors        []ProcessorDTO        `json:"processors"`
	TriggerProcessors []TriggerProcessorDTO `json:"triggerProcessors"`
	Flow              string                `json:"flow"`
}

// ProcessorDTO represents the data transfer object for a simple processor.
type ProcessorDTO struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Config         map[string]interface{} `json:"config"`
	MaxRetries     int                    `json:"maxRetries"`
	BackoffSeconds int                    `json:"backoffSeconds"`
	LogLevel       string                 `json:"logLevel"`
	Enabled        bool                   `json:"enabled"`
}

// TriggerProcessorDTO represents the data transfer object for a trigger processor.
type TriggerProcessorDTO struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Config     map[string]interface{} `json:"config"`
	Enabled    bool                   `json:"enabled"`
	SingleNode bool                   `json:"singleNode"`
	CronExpr   string                 `json:"cronExpression"`
}
