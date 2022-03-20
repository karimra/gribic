### Description

The Flush Command runs a [gRIBI Flush RPC](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L47) as a client, sending a [FlushRequest](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L469) to a gRIBI server.
The Server returns a single [FlushResponse](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L518).

### Usage

`gribic [global-flags] flush [local-flags]`

Alias: `f`

### Flags

#### ns

The `--ns` flag sets the network instance name the client wants to flush.

#### ns-all

The `--ns-all` flag indicates to the server that the client wants to flush all instances.

#### override

The `--override` flag indicates to the server that the client wants the server to not compare the election ID with already known `single-primary` clients.

### Examples

Flush all AFTs in network instance `default`

```bash
gribic -a router1 -u admin -p admin --skip-verify flush --ns default 
```

Flush all AFTs in all network instances

```bash
gribic -a router1 -u admin -p admin --skip-verify flush --ns-all
```
