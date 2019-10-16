package emitter

import (
	"errors"

	newrelic "github.com/newrelic/go-agent"
	insights "github.com/newrelic/go-insights/client"
	util "github.com/newrelic/nri-network-telemetry/internal/util"
	log "github.com/sirupsen/logrus"
)

type InsightsEmitterConfig struct {
	NrAccountID    string `envconfig:"NEW_RELIC_ACCOUNT_ID"`
	NrInsertKey    string `envconfig:"NEW_RELIC_INSERT_KEY"`
	NrInsightsHost string `envconfig:"NEW_RELIC_INSIGHTS_HOST"`
}

type insightsEmitter struct {
	emitChan chan map[string]interface{}
	config   InsightsEmitterConfig
	nr       newrelic.Application
}

/******************************************************************************
 *
 * Confirm we are valid
 *
 ******************************************************************************/
func (e *insightsEmitter) Validate() error {
	if e.config.NrAccountID == "" {
		return errors.New("missing Account ID")
	}

	if e.config.NrInsertKey == "" {
		return errors.New("missing Insert Key")
	}

	return nil
}

/******************************************************************************
 *
 * Return the channel we read from
 *
 ******************************************************************************/
func (e *insightsEmitter) EmitChan() chan map[string]interface{} {
	return e.emitChan
}

/******************************************************************************
 *
 * Start emitting things
 *
 ******************************************************************************/
func (e *insightsEmitter) Start(controlChan chan ControlMessage) error {
	log.Info("emitter::Insights: Starting emitter")

	if err := e.Validate(); err != nil {
		log.Errorf("emitter::Insights: Validation failed: %v", err)
		return err
	}

	// Do some stuff to prep
	client := insights.NewInsertClient(e.config.NrInsertKey, e.config.NrAccountID)

	if e.config.NrInsightsHost != "" {
		client.UseCustomURL(e.config.NrInsightsHost)
	}

	if err := client.Validate(); err != nil {
		log.Errorf("emitter::Insights: Client validation Error!")
		return err
	}

	if err := client.Start(); err != nil {
		log.Errorf("emitter::Insights: Failed to start batch client!")
		return err
	}

	for {
		select {
		case msg := <-controlChan:
			switch msg {
			case ControlMessageStart:
				// We're ready!
				log.Debug("emitter::Insights: Control Message: Start")
				continue
			case ControlMessageQuit:
				log.Debug("emitter::Insights: Control Message: Quit")
				client.Flush()
				controlChan <- ControlMessageDone // Signal exit

				return nil
			}
		case msg := <-e.emitChan:
			util.LogIfErr(client.EnqueueEvent(msg))
		}
	}
}
