package flowhandler

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/newrelic/nri-network-telemetry/internal/util"

	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
)

const (
	flowBufferSizeUDP    = 65535
	flowBufferSizePacket = 4096
	flowTimeoutRead      = 5
)

type ControlMessage int

const (
	ControlMessageQuit  ControlMessage = iota
	ControlMessageStart ControlMessage = iota
	ControlMessageReady ControlMessage = iota
	ControlMessageDone  ControlMessage = iota
)

type Config struct {
	BindAddress    string `envconfig:"FLOW_BIND_ADDRESS"`
	Port           int    `envconfig:"FLOW_PORT"`
	SflowEventType string `envconfig:"SFLOW_EVENT_TYPE"`
	IpfixEventType string `envconfig:"IPFIX_EVENT_TYPE"`
	AsnPeerMap     map[uint32]string
}

/******************************************************************************
 *
 * Create a new FlowHandler instance
 *
 ******************************************************************************/
func New(config Config, resultChan chan map[string]interface{}, nr newrelic.Application) *FlowHandler {
	return (&FlowHandler{
		config:     config,
		nr:         nr,
		resultChan: resultChan,
		sflowChan:  make(chan SflowPacket, flowBufferSizePacket),
		ipfixChan:  make(chan IpfixPacket, flowBufferSizePacket),
	})
}

/******************************************************************************
 *
 * FlowHandler struct
 *
 ******************************************************************************/
type FlowHandler struct {
	config     Config
	resultChan chan map[string]interface{}
	sflowChan  chan SflowPacket
	ipfixChan  chan IpfixPacket
	nr         newrelic.Application
}

func (s *FlowHandler) listenAddr() string {
	return fmt.Sprintf("%s:%d", s.config.BindAddress, s.config.Port)
}

/******************************************************************************
 *
 * Start the UDP server
 *
 ******************************************************************************/
func (s *FlowHandler) Start(controlChan chan ControlMessage) error {
	// Start the goroutines here
	ipfix := NewIpfixHandler(s.ipfixChan, s.resultChan, s.config.IpfixEventType, s.config.AsnPeerMap, s.nr)
	go ipfix.Start()

	sflow := NewSflowHandler(s.sflowChan, s.resultChan, s.config.SflowEventType, s.nr)
	go sflow.Start()

	/*
	 * Start the UDP server
	 */
	localAddr, err := net.ResolveUDPAddr("udp", s.listenAddr())
	if err != nil {
		log.Errorf("flowHandler: Unable to resolve '%s' with error: %v", s.listenAddr(), err)
		return err
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Errorf("flowHandler: Unable to bind with error: %v", err)
		return err
	}

	log.Infof("flowHandler: Listening on '%s'", localAddr)

	/*
	 * Infinitely process things
	 */
	for {
		select {
		case msg := <-controlChan:
			// Read from the control channel if it's available
			switch msg {
			case ControlMessageStart:
				// We're ready!
				log.Debug("flowHandler: Control Message: Start")
				continue
			case ControlMessageQuit:
				log.Debug("flowHandler: Control Message: Quit")
				// kill off the channels (Children will exit)
				close(s.sflowChan)
				log.Debug("flowHandler: Sflow channel closed")
				close(s.ipfixChan)
				log.Debug("flowHandler: Ipfix channel closed")

				controlChan <- ControlMessageDone // Signal exit

				log.Debug("flowHandler: Control Message Done sent")

				return nil
			}
		default:
			buf := make([]byte, flowBufferSizeUDP)
			/*
			 * Require a timeout so we can exit cleanly
			 * Timeout must be constantly updated
			 */
			util.LogIfErr(conn.SetReadBuffer(flowBufferSizeUDP))
			util.LogIfErr(conn.SetReadDeadline(time.Now().Add(time.Second * flowTimeoutRead)))

			bytesRead, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				// Do not log read timeouts
				if netErr, ok := err.(net.Error); ok && !netErr.Timeout() {
					log.Errorf("flowHandler: Error reading from UDP socket: %+v", err)
				}

				continue
			}

			agentIP := addr.IP.String()

			flowVersion := uint32(binary.BigEndian.Uint16(buf[0:])) // IPFIX version is 16 bits
			if flowVersion == 0 {
				flowVersion = binary.BigEndian.Uint32(buf[0:]) // sflow is 32 bits
			}

			log.Debugf("flowHandler: %s send %d bytes of version: %x ", agentIP, bytesRead, flowVersion)

			/*
			 * Route to the correct channel
			 */
			switch flowVersion {
			case 5: // Sflow
				util.LogIfErr(s.nr.RecordCustomMetric("sflowChanLength", float64(len(s.sflowChan))))
				s.sflowChan <- buf[:bytesRead]
			case 10: // IPFIX
				util.LogIfErr(s.nr.RecordCustomMetric("ipfixChanLength", float64(len(s.ipfixChan))))
				s.ipfixChan <- (IpfixPacket{
					AgentIP:   agentIP,
					BytesRead: bytesRead,
					Data:      buf[:bytesRead],
				})
			default: // Unsupported
				err := fmt.Errorf("unknown flow version %#x from %s, discarding %d bytes", flowVersion, agentIP, bytesRead)
				log.Warnf("flowHandler: %v", err)
			}
		}
	}
}
