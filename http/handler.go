package http

import (
	"fmt"
	"github.com/go-streamline/go-streamline/dto"
	"github.com/go-streamline/interfaces/definitions"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"sync"
	"time"
)

type flowManagerAPI struct {
	flowManager      definitions.FlowManager
	processorFactory definitions.ProcessorFactory
}

var paginationPool = sync.Pool{
	New: func() any {
		return new(definitions.PaginationRequest)
	},
}

var flowDTOPool = sync.Pool{
	New: func() any {
		return new(dto.FlowDTO)
	},
}

func NewFlowManagerAPI(flowManager definitions.FlowManager, processorFactory definitions.ProcessorFactory) *fiber.App {
	f := &flowManagerAPI{
		flowManager:      flowManager,
		processorFactory: processorFactory,
	}
	app := fiber.New()
	f.setupRoutes(app)
	return app
}

func (f *flowManagerAPI) setupRoutes(app *fiber.App) {
	app.Get("/api/flows", f.listFlows)
	app.Get("/api/flows/:id", f.getFlowByID)
	app.Get("/api/flows/:id/processors", f.getFlowProcessors)
	app.Get("/api/flows/:id/trigger-processors", f.getFlowTriggerProcessors)
	app.Post("/api/flows/:id/activate", f.activateFlow)
	app.Post("/api/flows", f.saveFlow)
}

func (f *flowManagerAPI) listFlows(c *fiber.Ctx) error {
	page, err := c.ParamsInt("page", 1)
	if err != nil {
		page = 1
	}
	size, err := c.ParamsInt("size", 10)
	if err != nil {
		size = 10
	}
	pagination := paginationPool.Get().(*definitions.PaginationRequest)
	defer paginationPool.Put(pagination)
	pagination.Page = page
	pagination.PerPage = size
	if err := c.QueryParser(pagination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid pagination parameters"})
	}
	since := time.Now().AddDate(-100, 0, 0)
	flows, err := f.flowManager.ListFlows(pagination, since)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	var flowDTOs []*dto.FlowDTO
	for _, flow := range flows.Data {
		flowDTO := flowDTOPool.Get().(*dto.FlowDTO)
		resetFlowDTO(flowDTO)
		err = dto.FlowEntityToDTO(flow, flowDTO)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		flowDTOs = append(flowDTOs, flowDTO)
	}
	defer func() {
		for _, flowDTO := range flowDTOs {
			flowDTOPool.Put(flowDTO)
		}
	}()
	return c.JSON(flowDTOs)
}

func resetFlowDTO(dto *dto.FlowDTO) {
	dto.ID = uuid.Nil
	dto.Name = ""
	dto.Description = ""
	dto.Active = false
	dto.Flow = ""
	dto.Processors = dto.Processors[:0]
	dto.TriggerProcessors = dto.TriggerProcessors[:0]
}

func (f *flowManagerAPI) getFlowByID(c *fiber.Ctx) error {
	flowID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid flow ID"})
	}
	flow, err := f.flowManager.GetFlowByID(flowID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	flowDTO := flowDTOPool.Get().(*dto.FlowDTO)
	resetFlowDTO(flowDTO)
	err = dto.FlowEntityToDTO(flow, flowDTO)
	defer flowDTOPool.Put(flowDTO)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(flowDTO)
}

func (f *flowManagerAPI) getFlowProcessors(c *fiber.Ctx) error {
	flowID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid flow ID"})
	}
	processors, err := f.flowManager.GetFlowProcessors(flowID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	processorsDTO := make([]*dto.ProcessorDTO, len(processors))
	for i, processor := range processors {
		processorsDTO[i] = dto.ProcessorEntityToDTO(processor)
	}
	return c.JSON(processorsDTO)
}

func (f *flowManagerAPI) getFlowTriggerProcessors(c *fiber.Ctx) error {
	flowID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid flow ID"})
	}
	processors, err := f.flowManager.GetTriggerProcessorsForFlow(flowID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	processorsDTO := make([]*dto.TriggerProcessorDTO, len(processors))
	for i, processor := range processors {
		processorsDTO[i] = dto.TriggerProcessorEntityToDTO(processor)
	}
	return c.JSON(processorsDTO)
}

func (f *flowManagerAPI) activateFlow(c *fiber.Ctx) error {
	flowID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid flow ID"})
	}
	var request struct {
		Active bool `json:"active"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := f.flowManager.SetFlowActive(flowID, request.Active); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func getProcessorsIDs(processors []*definitions.SimpleProcessor) []uuid.UUID {
	var ids []uuid.UUID
	for _, p := range processors {
		ids = append(ids, p.ID)
	}
	return ids
}

func (f *flowManagerAPI) saveFlow(c *fiber.Ctx) error {
	var flowDTO dto.FlowDTO
	if err := c.BodyParser(&flowDTO); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	flow, err := dto.FlowDTOToEntity(&flowDTO)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if flow.ID == uuid.Nil {
		flow.ID = uuid.New()
	}
	errs := f.validateProcessors(flow)
	if errs != nil && len(errs) > 0 {
		var errorMessages []string
		for _, e := range errs {
			errorMessages = append(errorMessages, e.Error())
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errorMessages})
	}
	if err := f.flowManager.SaveFlow(flow); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	newFlowDTO := flowDTOPool.Get().(*dto.FlowDTO)
	resetFlowDTO(newFlowDTO)
	err = dto.FlowEntityToDTO(flow, newFlowDTO)
	defer flowDTOPool.Put(newFlowDTO)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(newFlowDTO)
}

func (f *flowManagerAPI) validateProcessors(flow *definitions.Flow) []error {
	var allErrors []error
	var err error
	for _, processor := range flow.Processors {
		_, err = f.processorFactory.GetProcessor(processor.ID, processor.Type)
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("invalid processor %s: %w", processor.Name, err))
		}
	}
	for _, triggerProcessor := range flow.TriggerProcessors {
		_, err = f.processorFactory.GetTriggerProcessor(triggerProcessor.ID, triggerProcessor.Type)
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("invalid trigger processor %s: %w", triggerProcessor.Name, err))
		}
	}
	return allErrors
}
