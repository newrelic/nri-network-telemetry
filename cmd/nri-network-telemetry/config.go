package main

import (
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/newrelic/nri-network-telemetry/internal/emitter"
	"github.com/newrelic/nri-network-telemetry/internal/flowhandler"
	"github.com/newrelic/nri-network-telemetry/internal/netinfo"

	log "github.com/sirupsen/logrus"
)

const DefaultEmitTarget = "INSIGHTS"

// Config is the necessary parameters to function correctly, options overridden from the environment.
type Config struct {
	FlowConfig    flowhandler.Config
	EmitConfig    emitter.EmitConfig
	NetInfo       netinfo.NetInfo
	NrServiceName string `envconfig:"SERVICE_NAME"`
	NrLicenseKey  string `envconfig:"NEW_RELIC_LICENSE_KEY"`
	BindAddress   string `envconfig:"BIND_ADDRESS"`
	NetsFile      string `envconfig:"NETWORKS_FILE"`
	HostsFile     string `envconfig:"HOSTS_FILE"`
	EmitTarget    string `envconfig:"EMIT_TARGET"`
	HTTPPort      int    `envconfig:"HTTP_PORT"`
	Debug         bool   `default:"false"`
	NrEnabled     bool   `envconfig:"NEW_RELIC_ENABLED"`
}

// Load sets some default values which are then overridden by the environment to finally return a populated Config object.
func (c *Config) Load() (err error) {
	// Set defaults for those that have them
	c.Debug = false
	c.NrEnabled = true
	c.NrServiceName = AppName
	c.NetsFile = "asndb.csv"
	c.HostsFile = ""
	c.BindAddress = "0.0.0.0"
	c.HTTPPort = 8080

	// Set defaults for FlowHandler
	c.FlowConfig = flowhandler.Config{
		BindAddress:    c.BindAddress,
		Port:           6343,
		SflowEventType: "sflow",
		IpfixEventType: "ipfix",
	}

	// Set defaults for the Emitters
	c.EmitTarget = DefaultEmitTarget
	c.EmitConfig.Insights = emitter.InsightsEmitterConfig{}
	c.EmitConfig.Log = emitter.LogEmitterConfig{
		Prefix: AppName,
	}
	// Load config from Environment
	if err = envconfig.Process(AppName, c); err != nil {
		return err
	}

	// Load FlowHandler config from the Environment
	if err = envconfig.Process(AppName, &c.FlowConfig); err != nil {
		return err
	}

	// Load Emitter config from Environment
	if err = envconfig.Process(AppName, &c.EmitConfig.Log); err != nil {
		return err
	}
	if err = envconfig.Process(AppName, &c.EmitConfig.Insights); err != nil {
		return err
	}

	return err
}

/******************************************************************************
 *
 ******************************************************************************/
func parseCommandLine(args []string) (conf Config, err error) {
	log.Debugf("%s: Parsing configuration", AppName)

	cli := kingpin.New(AppName, "Flow collector for New Relic Insights")
	cli.Version(Version)

	debug := cli.Flag("debug", "Enable debugging").Default("false").Short('d').Bool()
	nrAgent := cli.Flag("nragent", "Disable New Relic Go Agent").Default("false").Short('n').Bool()
	emitTarget := cli.Flag("emit", "Target to emit to (LOG / INSIGHTS)").Short('t').String()

	netsFile := cli.Flag("nets", "ASN to Name CSV File").Short('a').String()
	hostsFile := cli.Flag("hosts", "IP to Hostname CSV File").Short('h').String()

	_, err = cli.Parse(args)
	if err != nil {
		log.Fatalf("failed to parse arguments: %v", err.Error())
	}

	if err = conf.Load(); err != nil {
		log.Fatalf("failed to load configuration with: %v", err.Error())
	}

	// Allow overriding
	conf.Debug = conf.Debug || *debug
	if conf.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	conf.NrEnabled = conf.NrEnabled && !*nrAgent

	if *emitTarget != "" {
		conf.EmitTarget = *emitTarget
	}

	log.Debugf("%s: Config before loading NetInfo: %+v", AppName, conf)

	if *netsFile != "" {
		conf.NetsFile = *netsFile
	}

	if *hostsFile != "" {
		conf.HostsFile = *hostsFile
	}

	conf.NetInfo = netinfo.NewNetInfo(conf.NetsFile, conf.HostsFile)

	conf.FlowConfig.AsnPeerMap = conf.NetInfo.AsnPeerMap()

	log.Infof("%s: Finished parsing config", AppName)

	return conf, err
}
