package netinfo

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/yl2chen/cidranger"

	log "github.com/sirupsen/logrus"
)

/******************************************************************************
 *
 * Read the ASN Database into the config
 *
 * Expected Format:
 *   network,autonomous_system_number,autonomous_system_organization
 ******************************************************************************/

// Network is an interface for insertable entry into a Ranger.
type Network interface {
	Network() net.IPNet
	Asn() uint32
}

type network struct {
	ipNet net.IPNet
	asn   uint32
}

// NewNetwork creates a new struct to hold information about this specific entry
func NewNetwork(ipNet net.IPNet, asn uint32) Network {
	return &network{
		ipNet: ipNet,
		asn:   asn,
	}
}

// Minimum required for RangerEntry
func (n *network) Network() net.IPNet {
	return n.ipNet
}

// Return the autonomous_system_number
func (n *network) Asn() uint32 {
	return n.asn
}

// For debugging
func (n *network) String() string {
	return fmt.Sprintf("%s, %d", n.ipNet.String(), n.asn)
}

/******************************************************************************
 *
 * Read the ASN Database into the config
 *
 * Expected Format:
 *   network,autonomous_system_number,autonomous_system_organization
 ******************************************************************************/
func (n *NetInfo) loadNetworks(filename string) (count uint32, err error) {
	n.asns = make(map[uint32]string)
	n.networks = cidranger.NewPCTrieRanger()

	if filename == "" {
		return 0, nil
	}

	fileh, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	reader := csv.NewReader(fileh)

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}

		// Grab the IP
		_, network, err := net.ParseCIDR(line[0])
		if err != nil {
			log.Errorf("failed to parse network with error: %v", err.Error())
			continue
		}

		// Grab the ASN
		asn, err := strconv.ParseUint(line[1], 10, 32)
		if err != nil {
			log.Errorf("converting ASN failed with error: %v", err.Error())
			continue
		}

		if asn > 0 {
			n.asns[uint32(asn)] = line[2]
			err = n.networks.Insert(NewNetwork(*network, uint32(asn)))
			if err != nil {
				return count, err
			}

			count++
		}
	}

	return count, nil
}
