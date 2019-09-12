# httpserver
--
    import "github.com/newrelic/nri-network-telemetry/internal/httpserver"


## Usage

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

#### type Server

```go
type Server struct {
}
```


#### func  New

```go
func New(version string, address string, port int, nr newrelic.Application) *Server
```
*****************************************************************************

    *
    * Create a new HTTPserver instance
    *
    *****************************************************************************

#### func (*Server) ServeHTTP

```go
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request)
```
*****************************************************************************

    *
    * Pass requests to the handler
    *
    *****************************************************************************

#### func (*Server) Start

```go
func (s *Server) Start(controlChan chan ControlMessage) error
```
*****************************************************************************

    *
    * Start the HTTP server
    *
    *****************************************************************************
