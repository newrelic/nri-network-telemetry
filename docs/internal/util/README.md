# util
--
    import "github.com/newrelic/nri-network-telemetry/internal/util"


## Usage

#### func  CombinedHash

```go
func CombinedHash(packet gopacket.Packet) uint64
```

#### func  FnvHash

```go
func FnvHash(s []byte) (h uint64)
```

#### func  LogIfErr

```go
func LogIfErr(err error)
```

#### func  Uint64ToS

```go
func Uint64ToS(longNum uint64) string
```
