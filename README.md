# Go-Streamline

Go-Streamline is a scalable modular flow management system made to be as simplified as
can be. For example, it can be used to automate data processing pipelines, 
where incoming data triggers a sequence of processing steps such as transformation, 
validation, and storage in a database. 
It offers a recovery mechanism to never lose data in case of a forced shutdown 
and flexibility to make any small change as easily as possible. 
Its modularity offers the capability to change as much as your use case requires 
as easily as can be.


---

## Table of Contents
<!-- TOC -->
* [Go-Streamline](#go-streamline)
  * [Table of Contents](#table-of-contents)
  * [Features](#features)
    * [Core Features:](#core-features)
    * [Database Integration:](#database-integration)
    * [Extensibility:](#extensibility)
    * [Performance:](#performance)
  * [Installation](#installation)
    * [Prerequisites](#prerequisites)
    * [Quick Build](#quick-build)
  * [Usage](#usage)
    * [Flow](#flow)
    * [Trigger Processor](#trigger-processor)
      * [Types](#types)
      * [Configuration](#configuration)
      * [In cluster](#in-cluster)
    * [Processors](#processors)
    * [Flow metadata](#flow-metadata)
    * [Expr Processor Configuration](#expr-processor-configuration)
    * [Routes](#routes)
      * [GET /api/flows](#get-apiflows)
      * [POST /api/flows](#post-apiflows)
      * [GET /api/flows/{id}](#get-apiflowsid)
      * [GET /api/flows/:id/processors](#get-apiflowsidprocessors)
      * [GET /api/flows/:id/trigger-processors](#get-apiflowsidtrigger-processors)
      * [POST /api/flows/:id/activate](#post-apiflowsidactivate)
  * [Developer Guide](#developer-guide)
    * [Trigger Processor](#trigger-processor-1)
    * [Processor](#processor)
    * [Processor Factory](#processor-factory)
      * [Change DB Dialect](#change-db-dialect)
      * [Error Handling](#error-handling)
  * [Configuration](#configuration-1)
  * [Contributing](#contributing)
  * [License](#license)
  * [Acknowledgments](#acknowledgments)
<!-- TOC -->


## Features

### Core Features:
- **Processors and Flows**: Manage workflows with decoupled Processor and Flow models, allowing for dynamic execution of tasks.
- **Trigger Processors**: Automatically scheduled via CRON or event-driven to initiate workflows.
- **Write-Ahead Logging (WAL)**: Ensures data integrity by logging ahead of operations, supporting robust recovery mechanisms.
- **Copy-On-Write (COW)**: Enables efficient state management for processors and flows.

### Database Integration:
- **Flow Storage**: Flows and their states are stored using Gorm with database migrations managed via Goose.
- **Optimized Queries**: Indexed database queries for efficient lookup, avoiding SQL keyword conflicts.

### Extensibility:
- **Interface-Driven Design**: Processors can be customized for diverse tasks with a unified interface for seamless integration.
- **Trigger Processor Integration**: Registers trigger processors automatically when flows are registered, simplifying configuration.

### Performance:
- **Parallel Execution**: Supports conditional parallel execution of processors without blocking worker threads.
- **Cron Scheduling**: Scheduled execution for recurring tasks using CRON expressions.
- **Minimal Overhead**: Optimized for performance with low-latency task handling.

---

## Installation
### Prerequisites

Ensure you have the following installed before proceeding:
* Go (version 1.22 or later)
* Git (for cloning the repository)
* A supported database (e.g., PostgreSQL, SQLite, or MySQL).

### Build
1. **Clone the Repository:**
   ```bash
   git clone https://github.com/go-streamline/go-streamline.git
   cd go-streamline
   ```

2. **Install Dependencies:**
   ```bash
   go mod tidy
   ```
   
3. **Build the Project:**
   ```bash
   go build -o go-streamline
   ```

4. **Run the Project:**
   ```bash
   CONFIG_PATH=./config.yaml ./go-streamline
   ```   

## Usage
Download the go-streamline package:
```shell
go get github.com/go-streamline/go-streamline
```
Create a `main.go` file:
```go
package main

import (
    "github.com/go-streamline/go-streamline/standard"
)

func main() {
  standard.Run(standard.GetDefaultAppPackage())
}

```
This will start up go-streamline with the default configuration(port 8080, sqlite database).
You can modify the startup setup by editing the `AppPackage` you get from `standard.GetDefaultAppPackage()`. 
After starting up go-streamline, a Rest API will start up which you can access at `{server}:8080`.


### Flow
A flow describes the processors, their configuration and in which order they will run. 
It consists the description of processors, trigger processors and the order of processors.
A flow consists of the following:
* **name** - Name of the flow.
* **description** - Description of the flow.
* **active** - If the flow is active or not. If it is not active, the flow will not run.
* **processors** - List of processors in the flow.
* **triggerProcessors** - List of trigger processors in the flow. A flow can have multiple trigger processors. Each will start its own new session.
* **flow** - The order of processors in the flow. The order of processors in the flow is determined by this field. The flow of processors is defined by the processor names separated by `->`. For example, if you want to run Processor1 first and then Processor2, the flow will be `Processor1->Processor2`.
  You can also define multiple processors to run in parallel by using `[]` brackets. For example, if you want to run Processor1 and Processor2 in parallel and then Processor3, the flow will be `[Processor1, Processor2]->Processor3`. You can also do nested parallel processing. For example, if you want to run Processor1 and Processor2 in parallel and then Processor3 and Processor4 in parallel, the flow will be `[Processor1, Processor2]->[Processor3, Processor4]`.
  Additionally, you can use inner orders. For example, `Processor1->[Processor2->Processor3, Processor4]` will run Processor1 first, then Processor2 and Processor3 in parallel and then Processor4(immediately after Processor2 finishes).


### Trigger Processor
A trigger processor is the starting point of the flow.
A flow can have several trigger processors, each will start its own new session and all results of each 
trigger processor will go through the same order of processors.
#### Types
A trigger processor can have 2 types:
* **Event Based** - Based on events. For example, a message is arriving from message queue.
* **Cron based** -- Scheduled to run every X time.  

This choice is up to the developer of the trigger processor.

Each trigger processor has the following basic configuration:
* **name** - Name of the trigger processor(can be anything).
* **type** - Type of the trigger processor(needs to be supported in the processor factory).
* **config** - Configuration of the trigger processor.
* **enabled** - If the trigger processor is enabled or not. If it is disabled, the flow will skip this trigger processor as if it does not exist.
* **singleNode** - Relevant for cluster - whether the trigger processor should run on only one node in the cluster.
* **cronExpression** - Cron expression for the trigger processor(if cron-based). If the trigger processor is cron based, this is the expression that will determine when the trigger processor will run.


#### Configuration
Each trigger processor has its own configuration. 
You can configure it according to your needs to determine the behavior of the trigger processor.

#### In cluster
When running in a cluster, you might want your trigger processor to run in only one node.
For example: running a query in your database which you would only want to process once.
For that, `singleNode` comes into action. If you configure it to `true`, go-streamline will make it run on only one node.
Moreover, go-streamline distributes all singleNode trigger processors as evenly as it across the cluster can so no one node in the cluster will be responsible for all singleNode trigger processors.


### Processors
Processors can transform, save or reload data. Each processor has its own custom configuration
and is loaded using a `ProcessorFactory`.
Each processor has the following basic configuration:
* **name** - Name of the processor(can be anything).
* **type** - Type of the processor(needs to be supported in the processor factory).
* **config** - Configuration of the processor.
* **maxRetries** - Maximum number of retries for the processor in case it fails.
* **backoffSeconds** - Number of seconds to wait before retrying the processor.
* **logLevel** - Log level of the processor.
* **enabled** - If the processor is enabled or not. If it is disabled, the flow will skip this processor as if it does not exist.

### Flow metadata
Each processor can create metadata that will be passed to the next processor in the flow.
This metadata can be used to pass information between processors.
For example, if you have a processor that reads a file and another processor that writes a file, you can pass the file path from the first processor to the second processor using metadata.
This takes us the to next point which is the Expr Processor Configuration.

### Expr Processor Configuration
Some processors support expressions in their configuration(in some of their fields).
In case of an expression-supported field, you can use `${expresion}` to use [expr language](https://github.com/expr-lang/expr) to make the configuration even more dynamic and flow session dependent.
The `expr` is extended with some functions that can be used in the configuration:
- `uuid()` - Generates a new UUID.
- `random(max)` - Generates a random number between 0 and max.
- `getEnv(key)` - Gets the environment variable with the given key.
Any processor can extend the expr language even further with its own additional functions.
All flow session metadata is available in the expression context.
For example, if you use `WriteFile`, it puts out `WriteFile.OutputPath`.
You can then use it later in the flow using `${$env["WriteFile.OutputPath"]"}` and you can even append strings to make it a nested path like this:
`${$env["WriteFile.OutputPath"]}/newFile.txt`. Inside `${}`, you can use any expression that is supported by the expr language so you can manipulate it as you wish.

### Routes
#### GET /api/flows
Get all flows.
Response:
<Status 200 OK>
```json
{
	"id": "dea256eb-c223-4284-bc4a-7371e660e9a4",
	"name": "my flow name",
	"description": "description for this flow",
	"active": true,
	"processors": [
		{
			"id": "069a777f-2502-4133-9649-b8c1dc97a4e0",
			"name": "ProcessorName1",
			"type": "ProcessorType1",
			"config": {
				"key": "value"
			},
			"maxRetries": 3,
			"backoffSeconds": 5,
			"logLevel": "info",
			"enabled": true
		},
		{
			"id": "94d04a46-d29a-4384-a9c0-7d7149712faa",
			"name": "ProcessorName2",
			"type": "ProcessorType2",
			"config": {
				"key": "value"
			},
			"maxRetries": 3,
			"backoffSeconds": 5,
			"logLevel": "info",
			"enabled": true
		}
	],
	"triggerProcessors": [
		{
			"name": "TriggerProcessorName1",
			"type": "TriggerProcessorType1",
			"config": {
				"key": "value"
			},
			"enabled": true,
			"singleNode": false,
			"cronExpression": "10 * * * * *"
		}
	],
	"flow": "ProcessorName1->ProcessorName2"
}
```

#### POST /api/flows
Create/Update a flow. If an ID is provided, it will update the flow with that ID.
Request:
```json
{
    "name": "my flow name",
    "description": "description for this flow",
    "active": true,
    "processors": [
        {
            "name": "ProcessorName1",
            "type": "ProcessorType1",
            "config": {
                "key": "value"
            },
            "maxRetries": 3,
            "backoffSeconds": 5,
            "logLevel": "info",
            "enabled": true
        },
        {
            "name": "ProcessorName2",
            "type": "ProcessorType2",
            "config": {
                "key": "value"
            },
            "maxRetries": 3,
            "backoffSeconds": 5,
            "logLevel": "info",
            "enabled": true
        }
    ],
    "triggerProcessors": [
        {
            "name": "TriggerProcessorName1",
            "type": "TriggerProcessorType1",
            "config": {
                "key": "value"
            },
            "enabled": true,
            "singleNode": false,
            "cronExpression": "10 * * * * *"
        }
    ],
    "flow": "ProcessorName1->ProcessorName2"
}
```

Response:
<Status 201 Created>
```json
{
    "id": "dea256eb-c223-4284-bc4a-7371e660e9a4",
    "name": "my flow name",
    "description": "description for this flow",
    "active": true,
    "processors": [
        {
            "id": "069a777f-2502-4133-9649-b8c1dc97a4e0",
            "name": "ProcessorName1",
            "type": "ProcessorType1",
            "config": {
                "key": "value"
            },
            "maxRetries": 3,
            "backoffSeconds": 5,
            "logLevel": "info",
            "enabled": true
        },
        {
            "id": "94d04a46-d29a-4384-a9c0-7d7149712faa",
            "name": "ProcessorName2",
            "type": "ProcessorType2",
            "config": {
                "key": "value"
            },
            "maxRetries": 3,
            "backoffSeconds": 5,
            "logLevel": "info",
            "enabled": true
        }
    ],
    "triggerProcessors": [
        {
            "name": "TriggerProcessorName1",
            "type": "TriggerProcessorType1",
            "config": {
                "key": "value"
            },
            "enabled": true,
            "singleNode": false,
            "cronExpression": "10 * * * * *"
        }
    ],
    "flow": "ProcessorName1->ProcessorName2"
}
```

#### GET /api/flows/{id}
Get a flow by ID.
Response:
<Status 200 OK>
```json
{
    "id": "dea256eb-c223-4284-bc4a-7371e660e9a4",
    "name": "my flow name",
    "description": "description for this flow",
    "active": true,
    "processors": [
        {
            "id": "069a777f-2502-4133-9649-b8c1dc97a4e0",
            "name": "ProcessorName1",
            "type": "ProcessorType1",
            "config": {
                "key": "value"
            },
            "maxRetries": 3,
            "backoffSeconds": 5,
            "logLevel": "info",
            "enabled": true
        },
        {
            "id": "94d04a46-d29a-4384-a9c0-7d7149712faa",
            "name": "ProcessorName2",
            "type": "ProcessorType2",
            "config": {
                "key": "value"
            },
            "maxRetries": 3,
            "backoffSeconds": 5,
            "logLevel": "info",
            "enabled": true
        }
    ],
    "triggerProcessors": [
        {
            "name": "TriggerProcessorName1",
            "type": "TriggerProcessorType1",
            "config": {
                "key": "value"
            },
            "enabled": true,
            "singleNode": false,
            "cronExpression": "10 * * * * *"
        }
    ],
    "flow": "ProcessorName1->ProcessorName2"
}
```

#### GET /api/flows/:id/processors
Get all processors of a flow by ID.
Response:
<Status 200 OK>
```json
[
    {
        "id": "069a777f-2502-4133-9649-b8c1dc97a4e0",
        "name": "ProcessorName1",
        "type": "ProcessorType1",
        "config": {
            "key": "value"
        },
        "maxRetries": 3,
        "backoffSeconds": 5,
        "logLevel": "info",
        "enabled": true
    },
    {
        "id": "94d04a46-d29a-4384-a9c0-7d7149712faa",
        "name": "ProcessorName2",
        "type": "ProcessorType2",
        "config": {
            "key": "value"
        },
        "maxRetries": 3,
        "backoffSeconds": 5,
        "logLevel": "info",
        "enabled": true
    }
]
```

#### GET /api/flows/:id/trigger-processors
Get all trigger processors of a flow by ID.
Response:
<Status 200 OK>
```json
[
   {
      "id": "069a777f-2502-4133-9649-b8c1dc97a4e0",
      "name": "TriggerProcessorName1",
      "type": "TriggerProcessorType1",
      "config": {
         "key": "value"
      },
      "enabled": true,
      "singleNode": false,
      "cronExpression": "10 * * * * *"
   }
]
```

#### POST /api/flows/:id/activate
Activate a flow by ID.
Response:
<Status 204 No Content>



## Developer Guide
After cloning the repository, you can edit `main.go` to fit your purposes:


### Trigger Processor
A trigger processor is a unit of work that can be executed based on an event or a schedule. It can be anything from a message arriving in a message queue to a scheduled task.

The only requirement for a trigger processor is to implement the `TriggerProcessor` interface.
However, you can also use `definitions.BaseProcessor` to implement the `TriggerProcessor` interface to get free utils.
To implement a trigger processor, you need to implement the following methods:
* `SetConfig(conf map[string]interface{}) error` - Set the configuration of the trigger processor. This method is called when the flow is loaded and can be used to validate the configuration and return an error if the configuration is invalid.
* `Name() string` - Return the type of the trigger processor.
* `Close() error` - Close the trigger processor. This method is called when the flow is stopped and can be used to clean up resources.
* `GetScheduleType() ScheduleType` - Return the schedule type of the processor. The schedule type can be either `CronDriven` or `EventDriven`. If the processor is event-based, return `EventDriven`. If the processor is cron-based, return `CronDriven`.
* `HandleSessionUpdate(update SessionUpdate)` - Handle a session update. This method is called when a session update is sent to the processor. The session update contains the metadata of the session and can be used to update the processor's state.
* `Execute(info *EngineFlowObject, produceFileHandler func() ProcessorFileHandler, log *logrus.Logger) ([]*TriggerProcessorResponse, error)` - execute the trigger processor. This method is either called infinitely in case of an event-based trigger processor or each time the cron expression is met in case of a cron-based trigger processor. It can be used to perform the work of the trigger processor. The method receives:
  * `*definitions.EngineFlowObject` - The flow object that contains the metadata of the flow.
  * `func() ProcessorFileHandler` - A function that returns a file handler that can be used to create files and thus create new sessions.
  * `*logrus.Logger` - The trigger processor-specific logger that can be used to log messages.
Each file(or session) that is created needs to be returned in the list of `TriggerProcessorResponse` so the engine can create new sessions for them along with its file handler and its own metadata.
You can also use the metadata of the flow object to mark files for later use. For example, if you use pub/sub and you want to ack the message only after the flow is done successfully, you can mark the file's metadata using the `TPMark` field, then save it and wait to get it back in the session update. This way, you can ack the message only after the flow is done successfully. If you see the session ended with an error, you can nack the message and the flow will be retried(or go to dlq). 

  It returns a list of `TriggerProcessorResponse` and an error if the trigger processor fails. `TriggerProcessorResponse` consists of:
  * `*definitions.EngineFlowObject` - The updated flow object.
  * `[]*definitions.EngineFlowObject` - The updated flow objects. This is used to create new sessions in case the trigger processor creates new sessions.
  * `error` - The error that occurred(if any).

You can also accept `definitions.StateManager` in the constructor of the trigger processor to make use of the state manager to save and load states of the trigger processor.
For example, if you read from a database, you can use the `StateTypeCluster` and save the last `last_update` timestamp in the state manager. This way, you can load the last update timestamp from the state manager and use it in your query to get only the new data since the last update.

Currently, `definitions.BaseProcessor` provides `DecodeMap(input interface{}, output interface{}) error` which helps in decoding input configuration to a struct.
Just annotate the fields of the struct with `mapstructure:"key"` to map the key in the configuration to the field in the struct.
For example:

```go
package main

import (
    "errors"
    "github.com/go-streamline/interfaces/definitions"
    "github.com/sirupsen/logrus"
)

type customTriggerProcessor struct {
    definitions.BaseProcessor
    config *myConfig
}

type myConfig struct {
    Value string `mapstructure:"value"`
}

func (c *customTriggerProcessor) SetConfig(conf map[string]interface{}) error {
    c.config = &myConfig{}
    err := c.DecodeMap(conf, c.config)
    if err != nil {
        return err
    }
    if c.config.Value == "" {
        return errors.New("value is required")
    }
    return nil
}

func (c *customTriggerProcessor) Name() string {
    return "CustomTriggerProcessor"
}

func (c *customTriggerProcessor) Close() error {
    return nil
}

func (c *customTriggerProcessor) GetScheduleType() definitions.ScheduleType {
    return definitions.CronDriven
}

func (c *customTriggerProcessor) HandleSessionUpdate(update definitions.SessionUpdate) {
    // Your session update handling logic
}

func (c *customTriggerProcessor) Execute(
    info *definitions.EngineFlowObject,
    produceFileHandler func() definitions.ProcessorFileHandler,
    log *logrus.Logger,
) ([]*definitions.TriggerProcessorResponse, error) {
    log.Infof("evaluating expression: %s", c.config.Value)
    evaluatedValue, err := info.EvaluateExpression(c.config.Value)
    if err != nil {
        return nil, err
    }
    log.Infof("evaluated value: %s", evaluatedValue)
    return []*definitions.TriggerProcessorResponse{
        {
			EngineFlowObject: &definitions.EngineFlowObject{
				Metadata: map[string]interface{}{},
				TPMark:  "",
			},
			FileHandler:    produceFileHandler(),
        },
    }, nil
}
```

### Processor
A processor is a unit of work that can be executed. It can be anything from a simple file copy to a complex data transformation.
The only requirement for a processor is to implement the `Processor` interface.
However, you can also use `definitions.BaseProcessor` to implement the `Processor` interface to get free utils.
To implement a processor, you need to implement the following methods:
* `SetConfig(conf map[string]interface{}) error` - Set the configuration of the processor. This method is called when the flow is loaded and can be used to validate the configuration and return an error if the configuration is invalid.
* `Name() string` - Return the type of the processor.
* `Close() error` - Close the processor. This method is called when the flow is stopped and can be used to clean up resources.
* `Execute(info *definitions.EngineFlowObject, fileHandler definitions.ProcessorFileHandler, log *logrus.Logger) (*definitions.EngineFlowObject, error)` - execute the processor. This method is called when the processor is executed and can be used to perform the work of the processor. The method should return the updated flow object and an error if the processor fails. It receives:
  * `*definitions.EngineFlowObject` - The flow object that contains the metadata of the flow.
  * `definitions.ProcessorFileHandler` - The file handler that can be used to read and write files.
  * `*logrus.Logger` - The processor-specific logger that can be used to log messages.

  It returns the updated flow object and an error if the processor fails. You can also use `*definitions.EngineFlowObject` to pass metadata between processors or even evaluate expressions in the configuration of the processor. For example, `outputPath, err := info.EvaluateExpression(w.config.Output)` will evaluate the expression in the `Output` field of the configuration of the processor.

You can also accept `definitions.StateManager` in the constructor of the trigger processor to make use of the state manager to save and load states of the trigger processor.
For example, if you read from a database, you can use the `StateTypeCluster` and save the last `last_update` timestamp in the state manager. This way, you can load the last update timestamp from the state manager and use it in your query to get only the new data since the last update.

Currently, `definitions.BaseProcessor` provides `DecodeMap(input interface{}, output interface{}) error` which helps in decoding input configuration to a struct.
Just annotate the fields of the struct with `mapstructure:"key"` to map the key in the configuration to the field in the struct.
For example:

```go
package main

import (
	"errors"
	"github.com/go-streamline/interfaces/definitions"
	"github.com/sirupsen/logrus"
)

type customProcessor struct {
	definitions.BaseProcessor
	config *myConfig
}

type myConfig struct {
	Value string `mapstructure:"value"`
}

func (c *customProcessor) SetConfig(conf map[string]interface{}) error {
	c.config = &myConfig{}
	err := c.DecodeMap(conf, c.config)
	if err != nil {
		return err
	}
	if c.config.Value == "" {
		return errors.New("value is required")
	}
	return nil
}

func (c *customProcessor) Name() string {
	return "CustomProcessor"
}

func (c *customProcessor) Close() error {
	return nil
}

func (c *customProcessor) Execute(
	info *definitions.EngineFlowObject,
	fileHandler definitions.ProcessorFileHandler,
	log *logrus.Logger,
) (*definitions.EngineFlowObject, error) {
	log.Infof("evaluating expression: %s", c.config.Value)
	evaluatedValue, err := info.EvaluateExpression(c.config.Value)
	if err != nil {
		return nil, err
	}
	log.Infof("evaluated value: %s", evaluatedValue)
	info.Metadata["CustomProcessor.evaluatedValue"] = evaluatedValue
	return info, nil
}

```

### Processor Factory
You can expand the default processor factory so you can use both the default processor factory 
and your own(to add additional processors of your own).
You can also override the existing processors in that way:

```go
package main

import (
	"github.com/go-streamline/interfaces/definitions"
	"github.com/google/uuid"
)

type customProcessorFactory struct {
	factory             definitions.ProcessorFactory
	stateManagerFactory definitions.StateManagerFactory
}

func (c *customProcessorFactory) GetProcessor(id uuid.UUID, typeName string) (definitions.Processor, error) {
	switch typeName {
	case "PublishPubSub":
		return NewCustomPubSubProcessor(), nil
    default:
       return c.factory.GetProcessor(id, typeName)
	}
}

func (c *customProcessorFactory) GetTriggerProcessor(id uuid.UUID, typeName string) (definitions.TriggerProcessor, error) {
   switch typeName {
   default:
      return c.factory.GetTriggerProcessor(id, typeName)
   }
}

func CreateCustomProcessorFactory(stateManagerFactory definitions.StateManagerFactory) definitions.ProcessorFactory {
	return &customProcessorFactory{
		stateManagerFactory: stateManagerFactory,
	}
}
```

Now, you can modify `main.go` to use your new processor factory:
```go
package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"github.com/go-streamline/go-streamline/config"
	"streamline/standard"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	app := config.Initialize(
		ctx,
		os.Args[1],
        CreateCustomProcessorFactory,
		standard.CreateDB,
		standard.CreateLoggerFactory,
		standard.CreateWriteAheadLogger,
		standard.CreateEngineErrorHandler,
	)
	go func() {
		err := app.Listen(":8080")
		if err != nil {
			logrus.WithError(err).Fatalf("could not init rest api")
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	<-ch
	cancel()
}
```

#### Change DB Dialect
Default dialect is sqlite. go-streamline uses [gorm](https://gorm.io) as its ORM library hence any dialect gorm supports is supported by go-streamline.
For example, to change dialect to Postgres, install the Postgres dialect:
```shell
go get -u gorm.io/driver.postgres
```
then modify `main.go` to use postgres driver:
```go
package main

import (
   "context"
   "github.com/sirupsen/logrus"
   "os"
   "os/signal"
   "github.com/go-streamline/go-streamline/config"
   "github.com/go-streamline/go-streamline/standard"
   "gorm.io/gorm"
   "gorm.io/driver/postgres"
   "syscall"
)

func createDB(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.Engine.DB.DSN), &gorm.Config{})
}

func main() {
   ctx, cancel := context.WithCancel(context.Background())
   app := config.Initialize(
      ctx,
      os.Args[1],
      standard.CreateProcessorFactory,
      standard.CreateDB,
      standard.CreateLoggerFactory,
      standard.CreateWriteAheadLogger,
      standard.CreateEngineErrorHandler,
   )
   go func() {
      err := app.Listen(":8080")
      if err != nil {
         logrus.WithError(err).Fatalf("could not init rest api")
      }
   }()

   ch := make(chan os.Signal, 1)
   signal.Notify(ch, syscall.SIGTERM)
   <-ch
   cancel()
}

```


#### Error Handling
When a processor fails, the error is logged and the processor is retried according to the configuration.
For each fail, a session update is sent to the EngineErrorHandler. The default one copies the failing file to a new path in case of a total failure(meaning the session ended with a failure).
You can modify it and create your own error handler:
```go
package main

import (
    "github.com/go-streamline/interfaces/definitions"
    "github.com/google/uuid"
)

type customErrorHandler struct {
}

func (c *customErrorHandler) HandleError(sessionID uuid.UUID, processorID uuid.UUID, err error) {
    // Your error handling logic
}
```

A session updates consists:
* **SessionID** - The ID of the session.
* **Finished** - Whether the session has finished or not.
* **Error** - The error that occurred(if any).
* **TPMark** - Used by the trigger processor to mark the file for later use.
* **SourcePath** - The path of the file that was processed.

Since go-streamline uses Copy on Write, the original file is never touched and thus can be copied for auditing purposes.

---

## Configuration

Configuration is done via config.yaml that is provided by environment variable `CONFIG_PATH`.
Example configuration:
```yaml
engine:
  # Path to store the streamline engine data
  workdir: /tmp/streamline
  # Number of workers to run the flow
  maxWorkers: 2
  # Interval in seconds to fetch the flows from the database
  flowCheckInterval: 60
  # Number of flows to fetch in a batch
  flowBatchSize: 100
  db:
    # Connection string for the database
    dsn: /tmp/streamline/flow.db
writeAheadLog:
  # Enable WAL for flow execution
  enabled: false
logging:
  # Log level
  level: info
  # Log file path (./ is the working directory)
  filename: ./logs/streamline.log
  # Max size of the log file in MB
  maxSizeMB: 5
  # Max age of the log file in days
  maxAgeDays: 7
  # Max number of log files to keep
  maxBackups: 3
  # Compress the log files
  compress: true
  # Log to console
  logToConsole: true
  # Custom log level per logger name
  customLogLevel:
    branch_tracker: warn
```

---

## Contributing

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/new-feature`).
3. Commit your changes (`git commit -m 'Add new feature'`).
4. Push to the branch (`git push origin feature/new-feature`).
5. Open a pull request.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- Contributors to `Go-Streamline`.
- Libraries and tools used in this project, including `Gorm`, `Goose`, and others.
