# flowhandler
--
    import "github.com/newrelic/nri-network-telemetry/internal/flowhandler"


## Usage

```go
const (
	ICMPTypeDestinationUnreachable = "Destination Unreachable"
	ICMPTypeParameterProblem       = "Parameter Problem"
	ICMPTypeRedirectMessage        = "Redirect Message"
)
```

```go
const SflowPacketBufferSize = 2048
```

#### type Config

```go
type Config struct {
	BindAddress    string `envconfig:"FLOW_BIND_ADDRESS"`
	Port           int    `envconfig:"FLOW_PORT"`
	SflowEventType string `envconfig:"SFLOW_EVENT_TYPE"`
	IpfixEventType string `envconfig:"IPFIX_EVENT_TYPE"`
	AsnPeerMap     map[uint32]string
}
```


#### type ControlMessage

```go
type ControlMessage int
```


```go
const (
	ControlMessageQuit  ControlMessage = iota
	ControlMessageStart ControlMessage = iota
	ControlMessageReady ControlMessage = iota
	ControlMessageDone  ControlMessage = iota
)
```

#### type FlowHandler

```go
type FlowHandler struct {
}
```

*****************************************************************************

    *
    * FlowHandler struct
    *
    *****************************************************************************

#### func  New

```go
func New(config Config, resultChan chan map[string]interface{}, nr newrelic.Application) *FlowHandler
```
*****************************************************************************

    *
    * Create a new FlowHandler instance
    *
    *****************************************************************************

#### func (*FlowHandler) Start

```go
func (s *FlowHandler) Start(controlChan chan ControlMessage) error
```
*****************************************************************************

    *
    * Start the UDP server
    *
    *****************************************************************************

#### type IpfixHandler

```go
type IpfixHandler struct {
}
```

*****************************************************************************

    *
    * IPFIXhandler object
    *
    *****************************************************************************

#### func  NewIpfixHandler

```go
func NewIpfixHandler(packetChan chan IpfixPacket, resultChan chan map[string]interface{}, eventType string, peerMap map[uint32]string, nr newrelic.Application) *IpfixHandler
```
*****************************************************************************

    *
    * Create a new IPFIXhandler instance
    *
    *****************************************************************************

#### func (*IpfixHandler) Start

```go
func (h *IpfixHandler) Start()
```
*****************************************************************************

    *
    * Start the IPFIX processor
    *
    *****************************************************************************

#### type IpfixPacket

```go
type IpfixPacket struct {
	AgentIP   string
	BytesRead int
	Data      []byte
}
```


#### type SflowHandler

```go
type SflowHandler struct {
}
```

*****************************************************************************

    *
    * SflowHandler object
    *
    *****************************************************************************

#### func  NewSflowHandler

```go
func NewSflowHandler(packetChan chan SflowPacket, resultChan chan map[string]interface{}, eventType string, nr newrelic.Application) *SflowHandler
```
*****************************************************************************

    *
    * Create a new SflowHandler instance
    *
    *****************************************************************************

#### func (*SflowHandler) Start

```go
func (h *SflowHandler) Start()
```
*****************************************************************************

    *
    * Start the Sflow processor
    *
    *****************************************************************************

#### type SflowPacket

```go
type SflowPacket []byte
```
