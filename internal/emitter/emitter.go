package emitter

import (
	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
)

type ControlMessage int

const EmitChannelBufferSize = 15000

const (
	ControlMessageQuit  ControlMessage = iota
	ControlMessageStart ControlMessage = iota
	ControlMessageReady ControlMessage = iota
	ControlMessageDone  ControlMessage = iota
)

type EmitInterface interface {
	Start(controlChan chan ControlMessage) error
	Validate() error
	EmitChan() chan map[string]interface{}
}

type EmitConfig struct {
	Log      LogEmitterConfig
	Insights InsightsEmitterConfig
}

/******************************************************************************
 *
 * Create a new instance of an Emitter
 *
 ******************************************************************************/
func New(target string, config EmitConfig, nr newrelic.Application) EmitInterface {

	// Return the correct type based on the requested Target
	switch target {
	case "LOG":
		return &logEmitter{
			emitChan: make(chan map[string]interface{}, EmitChannelBufferSize),
			config:   config.Log,
		}
	case "INSIGHTS":
		return &insightsEmitter{
			emitChan: make(chan map[string]interface{}, EmitChannelBufferSize),
			config:   config.Insights,
			nr:       nr,
		}
	default:
		log.Fatalf("Emitter: Unknown emit target '%s'", target)
	}

	return nil
}
