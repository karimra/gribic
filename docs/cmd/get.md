### Description

The Get Command runs a [gRIBI Get RPC](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L42) as a client, sending a [GetRequest](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L422) to a gRIBI server.
The Server returns a stream of [GetResponse(s)](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L462) with the installed [AFTs](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L444).

The client can specify the type of AFTs as well as the network instance it is interested on. Or simply request ALL AFT types from ALL network instances.

### Usage

`gribic [global-flags] get [local-flags]`

Alias: `g`

### Flags

#### ns

The `--ns` flag sets the network instance name the client is interested on.

#### aft

The `--aft` flag sets the AFT type the client is interested on. It defaults to `ALL` which means all AFT types.

Accepted values:

- `all`
- `nexthop` (or `nh`)
- `nexthop-group` (or `nhg`)
- `ipv4`
- `ipv6`
- `mac`
- `mpls`
- `policy-forwarding` (or `pf`)

### Examples

Query all AFTs in network instance `default`

```bash
gribic -a router1 -u admin -p admin --skip-verify get --ns default
```

Query all AFTs in all network instances

```bash
gribic -a router1 -u admin -p admin --skip-verify get --ns-all
```

Query AFT type `ipv4` in network instance `default`

```bash
gribic -a router1 -u admin -p admin --skip-verify get --ns default --aft ipv4
```

Query AFT type `nhg` (next hop group) in all network instances

```bash
gribic -a router1 -u admin -p admin --skip-verify get --ns-all --aft nhg
```
