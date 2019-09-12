package flowhandler

import (
	"net"
	"time"

	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"

	"github.com/calmh/ipfix"
	"github.com/google/gopacket/layers"
	"github.com/newrelic/nri-network-telemetry/internal/util"
)

// Can't make constant slices...
var (
	ipAddressFields = []string{
		"sourceIPv4Address", "destinationIPv4Address", "ipNextHopIPv4Address", "bgpNextHopIPv4Address",
		"sourceIPv6Address", "destinationIPv6Address", "ipv4RouterSc", "sourceIPv4Prefix",
		"destinationIPv4Prefix", "mplsTopLabelIPv4Address", "ipNextHopIPv6Address", "bgpNextHopIPv6Address",
		"exporterIPv4Address", "exporterIPv6Address", "mplsTopLabelIPv6Address", "destinationIPv6Prefix",
		"sourceIPv6Prefix", "collectorIPv4Address", "collectorIPv6Address", "postNATSourceIPv4Address",
		"postNATDestinationIPv4Address", "postNATSourceIPv6Address", "postNATDestinationIPv6Address",
		"originalExporterIPv4Address", "originalExporterIPv6Address", "pseudoWireDestinationIPv4Address",
	}
	macAddressFields = []string{
		"sourceMacAddress", "postDestinationMacAddress", "destinationMacAddress", "postSourceMacAddress",
		"staMacAddress", "wtpMacAddress", "dot1qCustomerSourceMacAddress", "dot1qCustomerDestinationMacAddress",
	}
)

const (
	ICMPTypeDestinationUnreachable = "Destination Unreachable"
	ICMPTypeParameterProblem       = "Parameter Problem"
	ICMPTypeRedirectMessage        = "Redirect Message"
)

/******************************************************************************
 *
 * Create a new IPFIXhandler instance
 *
 ******************************************************************************/
func NewIpfixHandler(packetChan chan IpfixPacket, resultChan chan map[string]interface{}, eventType string, peerMap map[uint32]string, nr newrelic.Application) *IpfixHandler {
	return (&IpfixHandler{
		packetChan: packetChan,
		resultChan: resultChan,
		eventType:  eventType,
		peerMap:    peerMap,
		nr:         nr,
	})
}

/******************************************************************************
 *
 * IPFIXhandler object
 *
 ******************************************************************************/
type IpfixHandler struct {
	resultChan chan map[string]interface{}
	packetChan chan IpfixPacket
	eventType  string
	peerMap    map[uint32]string
	nr         newrelic.Application
}

type IpfixPacket struct {
	AgentIP   string
	BytesRead int
	Data      []byte
}

/******************************************************************************
 *
 * Start the IPFIX processor
 *
 ******************************************************************************/
func (h *IpfixHandler) Start() {
	// Keep track of all the sessions
	sessions := make(map[string]chan []byte)

	for packet := range h.packetChan {
		if packetChan, ok := sessions[packet.AgentIP]; ok {
			packetChan <- packet.Data
		} else {
			sessions[packet.AgentIP] = make(chan []byte, flowBufferSizePacket)
			go h.handlePacketForAgent(packet.AgentIP, sessions[packet.AgentIP])
			sessions[packet.AgentIP] <- packet.Data
		}
	}
}

/******************************************************************************
 *
 * Per Agent, handle the packets coming in
 *
 ******************************************************************************/
func (h *IpfixHandler) handlePacketForAgent(agent string, packetChan chan []byte) {
	session := ipfix.NewSession()
	interpreter := ipfix.NewInterpreter(session)

	for packet := range packetChan {
		txn := h.nr.StartTransaction("IpfixPacket", nil, nil)
		util.LogIfErr(txn.AddAttribute("agent", agent))

		parseSegment := newrelic.StartSegment(txn, "ParseBuffer")
		msg, err := session.ParseBuffer(packet)
		util.LogIfErr(parseSegment.End())
		if err != nil {
			log.Warnf("IPFIXhandler: Error reading packet from %s: %v", agent, err)
			util.LogIfErr(txn.NoticeError(err))
			util.LogIfErr(txn.End())
			continue
		}

		recordsSeg := newrelic.StartSegment(txn, "ParseDataRecords")
		for _, record := range msg.DataRecords {
			rec := make(map[string]interface{})

			// Initial data
			rec["eventType"] = h.eventType
			rec["timestamp"] = time.Now()
			rec["agent"] = agent
			rec["templateId"] = record.TemplateID

			interpSeg := newrelic.StartSegment(txn, "InterpretRecord")
			ifs := interpreter.Interpret(record)
			util.LogIfErr(interpSeg.End())

			copySeg := newrelic.StartSegment(txn, "CopyRecord")
			for _, iif := range ifs {
				rec[iif.Name] = iif.Value
			}
			util.LogIfErr(copySeg.End())

			translateSeg := newrelic.StartSegment(txn, "TranslateRecord")
			if _, ok := rec["tcpControlBits"]; ok {
				rec["tcpFlagNS"] = (rec["tcpControlBits"].(uint16)&0x0100 == 0x0100)
				rec["tcpFlagCWR"] = (rec["tcpControlBits"].(uint16)&0x0080 == 0x0080)
				rec["tcpFlagECE"] = (rec["tcpControlBits"].(uint16)&0x0040 == 0x0040)
				rec["tcpFlagURG"] = (rec["tcpControlBits"].(uint16)&0x0020 == 0x0020)
				rec["tcpFlagACK"] = (rec["tcpControlBits"].(uint16)&0x0010 == 0x0010)
				rec["tcpFlagPSH"] = (rec["tcpControlBits"].(uint16)&0x0008 == 0x0008)
				rec["tcpFlagRST"] = (rec["tcpControlBits"].(uint16)&0x0004 == 0x0004)
				rec["tcpFlagSYN"] = (rec["tcpControlBits"].(uint16)&0x0002 == 0x0002)
				rec["tcpFlagFIN"] = (rec["tcpControlBits"].(uint16)&0x0001 == 0x0001)

				delete(rec, "tcpControlBits")
			}

			if _, ok := rec["icmpTypeCodeIPv4"]; ok {
				rec["icmpTypeCodeIPv4"] = rec["icmpTypeCodeIPv4"].(layers.ICMPv4TypeCode).String()
			}

			if _, ok := rec["icmpTypeCodeIPv6"]; ok {
				rec["icmpTypeCodeIPv6"] = rec["icmpTypeCodeIPv6"].(layers.ICMPv6TypeCode).String()
			}

			// *net.IP to String conversions
			for _, name := range ipAddressFields {
				if _, ok := rec[name]; ok {
					rec[name] = rec[name].(*net.IP).String()
				}
			}

			// *net.IP to String conversions
			for _, name := range macAddressFields {
				if _, ok := rec[name]; ok {
					rec[name] = rec[name].(*net.HardwareAddr).String()
				}
			}

			if _, ok := rec["bgpSourceAsNumber"]; ok {
				rec["peerName"] = h.peerMap[rec["bgpSourceAsNumber"].(uint32)]
			}

			if _, ok := rec["flowStartMilliseconds"]; ok {
				rec["duration"] = rec["flowEndMilliseconds"].(time.Time).Sub(rec["flowStartMilliseconds"].(time.Time)).Nanoseconds()

				delete(rec, "flowStartMilliseconds")
				delete(rec, "flowEndMilliseconds")
			}
			util.LogIfErr(translateSeg.End())

			// Send Event
			queueSegment := newrelic.StartSegment(txn, "QueueForEmit")
			h.resultChan <- rec
			util.LogIfErr(queueSegment.End())
		}
		util.LogIfErr(recordsSeg.End())

		util.LogIfErr(txn.End())
	}
}
