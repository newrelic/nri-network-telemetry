package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"

	"github.com/newrelic/go-agent/_integrations/nrlogrus"
	"github.com/newrelic/nri-network-telemetry/internal/emitter"
	"github.com/newrelic/nri-network-telemetry/internal/flowhandler"
	"github.com/newrelic/nri-network-telemetry/internal/httpserver"
)

const (
	AppName string = "NRNT"
	Version string = "dev"
)

const (
	TimeoutShutdownUpstream = 20 * time.Second // Timeout for upstream to notice we are gone after killing /status/check
	TimeoutShutdownFlow     = 15 * time.Second // Timeout allowed for the flow processor to drain
	TimeoutShutdownHTTP     = 10 * time.Second // Timeout allowed for the http server to finish up
	TimeoutShutdownEmit     = 20 * time.Second // Timeout allowed for the emitter to drain
	TimeoutShutdownNR       = 10 * time.Second // Timeout allowed for the New Relic go-agent to drain
)

func main() {
	var (
		err    error
		nrApp  newrelic.Application
		config Config
	)

	// Capture signals we care about
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load and Parse our config
	if config, err = parseCommandLine(os.Args[1:]); err != nil {
		log.Fatalf("Failed to start with error: %v", err)
	}

	/***********************************************
	 * Create a New Relic Agent
	 **********************************************/
	nrCfg := newrelic.NewConfig(config.NrServiceName, config.NrLicenseKey)
	nrCfg.Enabled = config.NrEnabled
	nrCfg.Logger = nrlogrus.StandardLogger()

	nrApp, err = newrelic.NewApplication(nrCfg)
	if nil != err {
		log.Fatalf("Failed to create New Relic application: %v", err)
	}

	/***********************************************
	 * Result handling
	 **********************************************/
	emitterControlChan := make(chan emitter.ControlMessage, 1)
	emitterControlChan <- emitter.ControlMessageStart
	resultEmitter := emitter.New(config.EmitTarget, config.EmitConfig, nrApp)

	go func() {
		err := resultEmitter.Start(emitterControlChan)
		if err != nil {
			log.Fatal(err)
		}
	}()

	/***********************************************
	 * Simple UDP listener, drops packets into the right parser
	 **********************************************/
	flowControlChan := make(chan flowhandler.ControlMessage, 1)
	flowControlChan <- flowhandler.ControlMessageStart
	fh := flowhandler.New(config.FlowConfig, resultEmitter.EmitChan(), nrApp)

	go func() {
		err := fh.Start(flowControlChan)
		if err != nil {
			log.Fatal(err)
		}
	}()

	/***********************************************
	 * Start HTTP server last
	 **********************************************/
	httpControlChan := make(chan httpserver.ControlMessage, 1)
	httpControlChan <- httpserver.ControlMessageStart
	apiHandler := httpserver.New(Version, config.BindAddress, config.HTTPPort, nrApp)

	go func() {
		err := apiHandler.Start(httpControlChan)
		if err != nil {
			log.Fatal(err)
		}
	}()

	/***********************************************
	 * Wait forever for a signal, and kill things
	 **********************************************/
	log.Infof("initialized v%s", Version)

	sig := <-sigChan
	log.Debugf("exiting on signal %v", sig)

	/***********************************************
	 * Cleanup stuff here
	 **********************************************/
	httpControlChan <- httpserver.ControlMessageQuit
	select {
	case <-httpControlChan:
		log.Debugf("http server shutdown cleanly")
		close(httpControlChan)
	case <-time.After(TimeoutShutdownHTTP):
		log.Errorf("http server failed to shutdown cleanly after %f seconds", TimeoutShutdownHTTP.Seconds())
	}

	// Arbitrary sleep so upstream notices that /status/check is gone
	log.Debugf("waiting %f seconds for upstream to drain our traffic", TimeoutShutdownUpstream.Seconds())
	time.Sleep(TimeoutShutdownUpstream)

	// Stop collecting flows
	flowControlChan <- flowhandler.ControlMessageQuit
	select {
	case <-flowControlChan:
		log.Debugf("flow handler shutdown cleanly")
		close(flowControlChan)
	case <-time.After(TimeoutShutdownFlow):
		log.Errorf("flow handler failed to shutdown cleanly after %f seconds", TimeoutShutdownFlow.Seconds())
	}

	// Kill the emitter, wait for confirmation
	emitterControlChan <- emitter.ControlMessageQuit
	select {
	case <-emitterControlChan:
		log.Debugf("emitter shutdown cleanly")
		close(emitterControlChan)
	case <-time.After(TimeoutShutdownEmit):
		log.Errorf("emitter failed to shutdown cleanly after %f seconds", TimeoutShutdownEmit.Seconds())
	}

	nrApp.Shutdown(TimeoutShutdownNR) // Flush data to NewRelic
}
