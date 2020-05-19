package flowhandler

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/newrelic/nri-network-telemetry/internal/util"

	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
)

const SflowPacketBufferSize = 2048

/******************************************************************************
 *
 * Create a new SflowHandler instance
 *
 ******************************************************************************/
func NewSflowHandler(packetChan chan SflowPacket, resultChan chan map[string]interface{}, eventType string, nr newrelic.Application) *SflowHandler {
	return (&SflowHandler{
		packetChan: packetChan,
		resultChan: resultChan,
		eventType:  eventType,
		nr:         nr,
	})
}

/******************************************************************************
 *
 * SflowHandler object
 *
 ******************************************************************************/
type SflowHandler struct {
	resultChan chan map[string]interface{}
	packetChan chan SflowPacket
	eventType  string
	nr         newrelic.Application
}

type SflowPacket []byte

/******************************************************************************
 *
 * Start the Sflow processor
 *
 ******************************************************************************/
func (h *SflowHandler) Start() {
	for packet := range h.packetChan {
		var sflow layers.SFlowDatagram

		txn := h.nr.StartTransaction("SflowPacket", nil, nil)
		parseSegment := newrelic.StartSegment(txn, "CreateParser")
		parser := gopacket.NewDecodingLayerParser(layers.LayerTypeSFlow, &sflow)
		decoded := make([]gopacket.LayerType, 0, 10)

		util.LogIfErr(parseSegment.End())

		decodeSegment := newrelic.StartSegment(txn, "DecodeLayers")
		err := parser.DecodeLayers(packet, &decoded)

		util.LogIfErr(decodeSegment.End())

		if err != nil {
			log.Warnf("SflowHandler: Unable to create decoder: %v", err)
			log.Debugf("%s", hex.Dump(packet))

			util.LogIfErr(txn.NoticeError(err))
			util.LogIfErr(txn.End())

			continue
		}

		util.LogIfErr(txn.AddAttribute("agent", sflow.AgentAddress.String()))

		err = h.makeEvents(sflow, txn)
		if err != nil {
			log.Errorf("SflowHandler: Failed to make events with error: %v", err)
			util.LogIfErr(txn.NoticeError(err))
			util.LogIfErr(txn.End())

			continue
		}

		util.LogIfErr(txn.End())
	}
}

/******************************************************************************
 *
 * Process Sflow packet
 *
 ******************************************************************************/
func (h *SflowHandler) makeEvents(sflow layers.SFlowDatagram, txn newrelic.Transaction) error {
	eventsSegment := newrelic.StartSegment(txn, "MakeEvents")

	for _, sample := range sflow.FlowSamples {
		rec := make(map[string]interface{})
		rec["eventType"] = h.eventType
		rec["timestamp"] = time.Now()
		rec["agent"] = sflow.AgentAddress.String()
		rec["agentAddress"] = sflow.AgentAddress.String() // TODO: REMOVE THIS!
		rec["samplingRate"] = int32(sample.SamplingRate)

		for _, record := range sample.GetRecords() {
			//nolint:gocritic
			switch record := record.(type) {
			case layers.SFlowRawPacketFlowRecord:
				util.LogIfErr(txn.AddAttribute("SFlowRawPacketFlowRecord", true))

				packet := record.Header
				for _, layer := range packet.Layers() {
					switch layer.LayerType() {
					case layers.LayerTypeDot1Q:
						rec["dot1qVLAN"] = int32(layer.(*layers.Dot1Q).VLANIdentifier)
						rec["dot1qNextLayer"] = layer.(*layers.Dot1Q).NextLayerType().String()
					case layers.LayerTypeIPv4:
						rec["length"] = int64(packet.NetworkLayer().(*layers.IPv4).Length)
						rec["networkDestinationAddress"] = packet.NetworkLayer().NetworkFlow().Dst().String()
						rec["networkFlowHash"] = util.Uint64ToS(packet.NetworkLayer().NetworkFlow().FastHash())
						rec["networkNextLayer"] = layer.(*layers.IPv4).NextLayerType().String()
						rec["networkSourceAddress"] = packet.NetworkLayer().NetworkFlow().Src().String()
						rec["networkType"] = packet.NetworkLayer().LayerType().String()
						rec["scaledByteCount"] = rec["length"].(int64) * int64(sample.SamplingRate)

					case layers.LayerTypeEthernet:
						rec["linkSourceAddress"] = packet.LinkLayer().LinkFlow().Src().String()
						rec["linkDestinationAddress"] = packet.LinkLayer().LinkFlow().Dst().String()
						rec["linkFlowHash"] = util.Uint64ToS(packet.LinkLayer().LinkFlow().FastHash())
						rec["linkType"] = packet.LinkLayer().LayerType().String()
						rec["linkNextLayer"] = layer.(*layers.Ethernet).NextLayerType().String()
					case layers.LayerTypeTCP:
						rec["transportSourcePort"] = packet.TransportLayer().TransportFlow().Src().String()
						rec["transportDestinationPort"] = packet.TransportLayer().TransportFlow().Dst().String()
						rec["transportFlowHash"] = util.Uint64ToS(packet.TransportLayer().TransportFlow().FastHash())
						rec["transportType"] = packet.TransportLayer().LayerType().String()
						rec["combinedHash"] = util.Uint64ToS(util.CombinedHash(packet))
						rec["transportWindowSize"] = int32(layer.(*layers.TCP).Window)
					case layers.LayerTypeUDP:
						rec["transportSourcePort"] = packet.TransportLayer().TransportFlow().Src().String()
						rec["transportDestinationPort"] = packet.TransportLayer().TransportFlow().Dst().String()
						rec["transportFlowHash"] = util.Uint64ToS(packet.TransportLayer().TransportFlow().FastHash())
						rec["transportType"] = packet.TransportLayer().LayerType().String()
						rec["combinedHash"] = util.Uint64ToS(util.CombinedHash(packet))
					}
				}

			case layers.SFlowExtendedGatewayFlowRecord:
				util.LogIfErr(txn.AddAttribute("SFlowExtendedSwitchFlowRecord", true))

				rec["nextHop"] = record.NextHop.String()
				rec["AS"] = record.AS
				rec["sourceAS"] = record.SourceAS
				rec["peerAS"] = record.PeerAS
				rec["ASPathCount"] = record.ASPathCount
				//rec["ASPath"] = record.ASPath
				//rec["communities"]
				rec["localPref"] = record.LocalPref

			case layers.SFlowExtendedSwitchFlowRecord:
				util.LogIfErr(txn.AddAttribute("SFlowExtendedSwitchFlowRecord", true))

			case layers.SFlowExtendedRouterFlowRecord:
				util.LogIfErr(txn.AddAttribute("SFlowExtendedRouterFlowRecord", true))

			case layers.SFlowExtendedURLRecord:
				util.LogIfErr(txn.AddAttribute("SFlowExtendedURLRecord", true))

			case layers.SFlowExtendedUserFlow:
				util.LogIfErr(txn.AddAttribute("SFlowExtendedUserFlow", true))

			default:
				err := errors.New("sflowHandler: unknown record type")
				log.Warn(err)
				util.LogIfErr(txn.NoticeError(err))
			}
		}

		// Send off the event
		queueSegment := newrelic.StartSegment(txn, "QueueForEmit")
		h.resultChan <- rec

		util.LogIfErr(queueSegment.End())
	}

	return eventsSegment.End()
}
