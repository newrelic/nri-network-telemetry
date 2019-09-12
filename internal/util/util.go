package util

import (
	"fmt"

	"github.com/google/gopacket"

	log "github.com/sirupsen/logrus"
)

const (
	fnvBasis = 14695981039346656037
	fnvPrime = 1099511628211
)

func FnvHash(s []byte) (h uint64) {
	h = fnvBasis
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime
	}
	return
}

func Uint64ToS(longNum uint64) string {
	return fmt.Sprintf("%d", longNum)
}

func CombinedHash(packet gopacket.Packet) uint64 {
	networkSource := packet.NetworkLayer().NetworkFlow().Src().Raw()
	transportSource := packet.TransportLayer().TransportFlow().Src().Raw()

	networkDestination := packet.NetworkLayer().NetworkFlow().Dst().Raw()
	transportDestination := packet.TransportLayer().TransportFlow().Dst().Raw()

	h := FnvHash(append(networkSource, transportSource...)) + FnvHash(append(networkDestination, transportDestination...))

	return h
}

func LogIfErr(err error) {
	if err != nil {
		log.Error(err)
	}
}
