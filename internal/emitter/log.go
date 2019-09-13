package emitter

import (
	log "github.com/sirupsen/logrus"
)

type LogEmitterConfig struct {
	Prefix string `envconfig:"EMIT_LOG_PREFIX"`
}

type logEmitter struct {
	emitChan chan map[string]interface{}
	config   LogEmitterConfig
}

/******************************************************************************
 *
 * Confirm we are valid
 *
 ******************************************************************************/
func (e *logEmitter) Validate() error {
	return nil
}

/******************************************************************************
 *
 * Return the channel we read from
 *
 ******************************************************************************/
func (e *logEmitter) EmitChan() chan map[string]interface{} {
	return e.emitChan
}

/******************************************************************************
 *
 * Start emitting things
 *
 ******************************************************************************/
func (e *logEmitter) Start(controlChan chan ControlMessage) error {
	log.Info("emitter::Log: Starting emitter")

	if err := e.Validate(); err != nil {
		log.Warnf("emitter::Log: Validation failed: %v", err)
		return err
	}

	for {
		select {
		case msg := <-controlChan:
			switch msg {
			case ControlMessageStart:
				// We're ready!
				log.Debug("emitter::Log: Control Message: Start")
				continue
			case ControlMessageQuit:
				log.Debug("emitter::Log: Control Message: Quit")
				controlChan <- ControlMessageDone // Signal exit

				return nil
			}
		case msg := <-e.emitChan:
			log.WithFields(msg).Info("emitter message")
		}
	}
}
