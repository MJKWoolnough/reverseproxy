# unixconn
--
    import "vimagination.zapto.org/reverseproxy/unixconn"

Package unixconn facilitates creating reverse proxy connections

## Usage

```go
var (
	ErrInvalidAddress   = errors.New("port must be 0 < port < 2^16")
	ErrAlreadyListening = errors.New("port already being listened on")
)
```
Errors

#### func  Listen

```go
func Listen(network, address string) (net.Listener, error)
```
Listen creates a reverse proxy connection, falling back to the net package if
the reverse proxy is not available
