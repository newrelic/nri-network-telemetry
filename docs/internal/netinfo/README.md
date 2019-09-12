# netinfo
--
    import "github.com/newrelic/nri-network-telemetry/internal/netinfo"


## Usage

#### type NetInfo

```go
type NetInfo struct {
}
```


#### func  NewNetInfo

```go
func NewNetInfo(networkFile string, hostFile string) NetInfo
```

#### func (*NetInfo) AsnPeerMap

```go
func (n *NetInfo) AsnPeerMap() map[uint32]string
```
AsnPeerMap returns a map of all ASNs to Organization name

#### type Network

```go
type Network interface {
	Network() net.IPNet
	Asn() uint32
}
```

Network is an interface for insertable entry into a Ranger.

#### func  NewNetwork

```go
func NewNetwork(ipNet net.IPNet, asn uint32) Network
```
NewNetwork creates a new struct to hold information about this specific entry
