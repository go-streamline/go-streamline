package dto

import (
	"github.com/go-streamline/interfaces/definitions"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strings"
)

// ProcessorDTOToEntity converts a ProcessorDTO to a SimpleProcessor entity.
func ProcessorDTOToEntity(processorDTO *ProcessorDTO) *definitions.SimpleProcessor {
	id := processorDTO.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	return &definitions.SimpleProcessor{
		ID:             id,
		Name:           processorDTO.Name,
		Type:           processorDTO.Type,
		Config:         processorDTO.Config,
		MaxRetries:     processorDTO.MaxRetries,
		BackoffSeconds: processorDTO.BackoffSeconds,
		LogLevel:       parseLogLevel(processorDTO.LogLevel),
		Enabled:        processorDTO.Enabled,
	}
}

// ProcessorEntityToDTO converts a SimpleProcessor entity to a ProcessorDTO.
func ProcessorEntityToDTO(processor *definitions.SimpleProcessor) *ProcessorDTO {
	return &ProcessorDTO{
		ID:             processor.ID,
		Name:           processor.Name,
		Type:           processor.Type,
		Config:         processor.Config,
		MaxRetries:     processor.MaxRetries,
		BackoffSeconds: processor.BackoffSeconds,
		LogLevel:       processor.LogLevel.String(),
		Enabled:        processor.Enabled,
	}
}

// TriggerProcessorDTOToEntity converts a TriggerProcessorDTO to a SimpleTriggerProcessor entity.
func TriggerProcessorDTOToEntity(triggerProcessorDTO *TriggerProcessorDTO) *definitions.SimpleTriggerProcessor {
	return &definitions.SimpleTriggerProcessor{
		ID:         uuid.New(),
		Name:       triggerProcessorDTO.Name,
		Type:       triggerProcessorDTO.Type,
		Config:     triggerProcessorDTO.Config,
		Enabled:    triggerProcessorDTO.Enabled,
		SingleNode: triggerProcessorDTO.SingleNode,
		CronExpr:   triggerProcessorDTO.CronExpr,
	}
}

// TriggerProcessorEntityToDTO converts a SimpleTriggerProcessor entity to a TriggerProcessorDTO.
func TriggerProcessorEntityToDTO(triggerProcessor *definitions.SimpleTriggerProcessor) *TriggerProcessorDTO {
	return &TriggerProcessorDTO{
		Name:       triggerProcessor.Name,
		Type:       triggerProcessor.Type,
		Config:     triggerProcessor.Config,
		Enabled:    triggerProcessor.Enabled,
		SingleNode: triggerProcessor.SingleNode,
		CronExpr:   triggerProcessor.CronExpr,
	}
}

// parseLogLevel converts a log level string to a logrus.Level.
func parseLogLevel(logLevel string) logrus.Level {
	switch strings.ToLower(logLevel) {
	case "panic":
		return logrus.PanicLevel
	case "fatal":
		return logrus.FatalLevel
	case "error":
		return logrus.ErrorLevel
	case "warn":
		return logrus.WarnLevel
	case "info":
		return logrus.InfoLevel
	case "debug":
		return logrus.DebugLevel
	case "trace":
		return logrus.TraceLevel
	default:
		return logrus.InfoLevel
	}
}

// FlowEntityToDTO converts a Flow entity to a FlowDTO.
// We reconstruct the flow string from the FirstProcessors and their NextProcessors.
func FlowEntityToDTO(flow *definitions.Flow) (*FlowDTO, error) {
	flowDTO := &FlowDTO{
		ID:          flow.ID,
		Name:        flow.Name,
		Description: flow.Description,
		Active:      flow.Active,
	}

	// Convert all processors to DTOs
	for _, processor := range flow.Processors {
		dto := ProcessorEntityToDTO(processor)
		flowDTO.Processors = append(flowDTO.Processors, *dto)
	}

	// Convert TriggerProcessors
	for _, trigger := range flow.TriggerProcessors {
		dto := TriggerProcessorEntityToDTO(trigger)
		flowDTO.TriggerProcessors = append(flowDTO.TriggerProcessors, *dto)
	}

	// If FirstProcessors is not empty, use them as entry points
	// Otherwise, derive entry processors again (in case they're not provided)
	firstProcessors := flow.FirstProcessors
	if len(firstProcessors) == 0 {
		// Derive if needed
		referenced := make(map[uuid.UUID]bool)
		for _, p := range flow.Processors {
			for _, np := range p.NextProcessors {
				referenced[np.ID] = true
			}
		}
		for _, p := range flow.Processors {
			if !referenced[p.ID] {
				firstProcessors = append(firstProcessors, p)
			}
		}
	}

	// Build the flow string from the first processors
	var sb strings.Builder
	visited := make(map[uuid.UUID]bool)
	for i, fp := range firstProcessors {
		if i > 0 {
			sb.WriteString(", ")
		}
		subFlow, err := serializeProcessor(fp, visited)
		if err != nil {
			return nil, err
		}
		sb.WriteString(subFlow)
	}
	flowDTO.Flow = sb.String()

	return flowDTO, nil
}

// serializeProcessor traverses the graph of processors using NextProcessors to rebuild the flow string.
func serializeProcessor(processor *definitions.SimpleProcessor, visited map[uuid.UUID]bool) (string, error) {
	if visited[processor.ID] {
		return processor.Name, nil // Avoid cycles
	}
	visited[processor.ID] = true

	var sb strings.Builder
	sb.WriteString(processor.Name)

	if len(processor.NextProcessors) > 0 {
		if len(processor.NextProcessors) > 1 {
			sb.WriteString("->[")
			for i, np := range processor.NextProcessors {
				if i > 0 {
					sb.WriteString(", ")
				}
				subFlow, err := serializeProcessor(np, visited)
				if err != nil {
					return "", err
				}
				sb.WriteString(subFlow)
			}
			sb.WriteString("]")
		} else {
			sb.WriteString("->")
			subFlow, err := serializeProcessor(processor.NextProcessors[0], visited)
			if err != nil {
				return "", err
			}
			sb.WriteString(subFlow)
		}
	}

	return sb.String(), nil
}
