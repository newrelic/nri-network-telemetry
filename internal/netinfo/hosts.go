package netinfo

import (
	"encoding/csv"
	"io"
	"os"
)

/******************************************************************************
 *
 * Read the ASN Database into the config
 *
 * Expected Format:
 *   ip_address,hostname
 ******************************************************************************/
func (n *NetInfo) loadHosts(filename string) (count uint32, err error) {
	n.hosts = make(map[string]string)

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

		if line[0] != "" && line[1] != "" {
			n.hosts[line[0]] = line[1]
			count++
		}
	}

	return count, nil
}
