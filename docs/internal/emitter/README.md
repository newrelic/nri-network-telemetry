# emitter
--
    import "github.com/newrelic/nri-network-telemetry/internal/emitter"


## Usage

```go
const EmitChannelBufferSize = 15000
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

#### type EmitConfig

```go
type EmitConfig struct {
	Log      LogEmitterConfig
	Insights InsightsEmitterConfig
}
```


#### type EmitInterface

```go
type EmitInterface interface {
	Start(controlChan chan ControlMessage) error
	Validate() error
	EmitChan() chan map[string]interface{}
}
```


#### func  New

```go
func New(target string, config EmitConfig, nr newrelic.Application) EmitInterface
```
*****************************************************************************

    *
    * Create a new instance of an Emitter
    *
    *****************************************************************************

#### type InsightsEmitterConfig

```go
type InsightsEmitterConfig struct {
	NrAccountID    string `envconfig:"NEW_RELIC_ACCOUNT_ID"`
	NrInsertKey    string `envconfig:"NEW_RELIC_INSERT_KEY"`
	NrInsightsHost string `envconfig:"NEW_RELIC_INSIGHTS_HOST"`
}
```


#### type LogEmitterConfig

```go
type LogEmitterConfig struct {
	Prefix string `envconfig:"EMIT_LOG_PREFIX"`
}
```
