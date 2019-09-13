package netinfo

import (
	"github.com/yl2chen/cidranger"

	log "github.com/sirupsen/logrus"
)

type NetInfo struct {
	networks cidranger.Ranger
	asns     map[uint32]string
	hosts    map[string]string
}

func NewNetInfo(networkFile string, hostFile string) NetInfo {
	n := NetInfo{}

	if networkFile != "" {
		log.Infof("loading network data from '%s'", networkFile)
		count, err := n.loadNetworks(networkFile)
		if err != nil {
			log.Errorf("failed with error: %v", err.Error())
		}
		log.Debugf("loaded %d network entries", count)
	}

	if hostFile != "" {
		log.Infof("loading host data from '%s'", hostFile)
		count, err := n.loadHosts(hostFile)
		if err != nil {
			log.Errorf("failed with error: %v", err.Error())
		}
		log.Debugf("loaded %d host entries", count)
	}

	return n
}

// AsnPeerMap returns a map of all ASNs to Organization name
func (n *NetInfo) AsnPeerMap() map[uint32]string {
	if n.asns == nil {
		n.asns = make(map[uint32]string)
	}

	return n.asns
}
