# reverseproxy
--
    import "vimagination.zapto.org/reverseproxy"

Package reverseproxy implements a basic HTTP/TLS connection forwarder based
either the passed Host header or SNI extension

## Usage

```go
var (
	ErrClosed = errors.New("closed")
)
```
Error

```go
var (
	ErrInvalidPort = errors.New("cannot register on port 0")
)
```
Errors

#### type HostName

```go
type HostName string
```

HostName represents an exact hostname to match on

#### func (HostName) MatchService

```go
func (h HostName) MatchService(serviceName string) bool
```
MatchService implements the MatchServiceName interface

#### type HostNameSuffix

```go
type HostNameSuffix string
```

HostNameSuffix represents a partial hostname to match the end on

#### func (HostNameSuffix) MatchService

```go
func (h HostNameSuffix) MatchService(serviceName string) bool
```
MatchService implements the MatchServiceName interface

#### type Hosts

```go
type Hosts []MatchServiceName
```

Hosts represents a list of servicenames to match against

#### func (Hosts) MatchService

```go
func (h Hosts) MatchService(serviceName string) bool
```
MatchService implements the MatchServiceName interface

#### type MatchServiceName

```go
type MatchServiceName interface {
	MatchService(string) bool
}
```

MatchServiceName allows differing ways of matching a service name to a service

#### type Port

```go
type Port struct {
}
```

Port represents a service waiting on a port

#### func  AddRedirect

```go
func AddRedirect(serviceName MatchServiceName, port uint16, to net.Addr) (*Port, error)
```
AddRedirect sets a port to be redirected to an external service

#### func (*Port) Close

```go
func (p *Port) Close() error
```
Close closes this port connection

#### func (*Port) Closed

```go
func (p *Port) Closed() bool
```
Closed returns whether the port has been closed or not

#### func (*Port) Status

```go
func (p *Port) Status() Status
```
Status retrieves the status of a Port

#### type Status

```go
type Status struct {
	Ports           []uint16
	Closing, Active bool
}
```

Status constains the status of a Port

#### type UnixCmd

```go
type UnixCmd struct {
}
```

UnixCmd holds the information required to control (close) a server and its
resources

#### func  RegisterCmd

```go
func RegisterCmd(msn MatchServiceName, cmd *exec.Cmd) (*UnixCmd, error)
```
RegisterCmd runs the given command and waits for incoming listeners from it

#### func (*UnixCmd) Close

```go
func (u *UnixCmd) Close() error
```
Close closes all ports for the server and sends a signal to the server to close

#### func (*UnixCmd) Status

```go
func (u *UnixCmd) Status() Status
```
Status retrieves the Status of the UnixCmd
