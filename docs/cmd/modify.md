### Description

The Modify Command runs a [gRIBI Modify RPC](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L31) as a client, sending a stream of [ModifyRequest(s)](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L52) to a gRIBI server.
The Server returns a stream of [ModifyResponse(s)](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L213).

The ModifyRequest is used to set the current session parameters (redundancy, persistence, and ack mode) as well as issuing AFT operation to the server.

The AFT operation can be an ADD, REPLACE or DELETE and references a single AFT entry of type IPV4, IPv6, Next Hop, Next Hop Group, MPLS, MAC or Policy Forwarding.

A single modifyRequest can carry multiple AFT operations.

A Modify RPC start with the client sending a ModifyRequest indicating the [session parameters](https://github.com/openconfig/gribi/blob/master/v1/proto/service/gribi.proto#L342) it wants to apply to the current session, the parameters are:

- Redundancy: specifies the client redundancy mode, either `ALL_PRIMARY` or `SINGLE_PRIMARY`
    - `ALL_PRIMARY`: is the default and indicates that the server should accept AFT operations from all clients.

        When it comes to ADD operations, the server should add an entry when it first receives it from any client.
        While it should wait for the last delete to remove it from its RIB.

        In other words, the server should keep track of the number of clients it received a specific entry from.

    - `SINGLE_PRIMARY`: implies that the clients went through an election process and a single one came out as primary, it has the highest election ID which it sends to the server in the initial ModifyRequest as well as with each AFT Operation.

       The server accepts AFT Operations only from the client with the highest election ID.

- Persistence: Specifies desired server behavior when the client disconnects.
    - `DELETE`: is the default, it means that all AFTs received from the client shall be deleted when it disconnects.
    - `PRESERVE`: the server should keep the RIB and FIB entries set by the client when it disconnects.

- Ack Mode: Specifies the Ack type expected by the client
    - `RIB_ACK`: the server must respond with `RIB_PROGRAMMED`
    - `RIB_AND_FIB_ACK`: the server must respond with `RIB_PROGRAMMED`, if the AFT entry is also programmed in the NE FIB, the server must response with `FIB_PROGRAMMED` instead.

### Usage

`gribic [global-flags] modify [local-flags]`

Alias: `mod`, `m`

### Flags

#### single-primary

The `--single-primary` flag set the session parameters redundancy to `SINGLE_PRIMARY`

#### preserve

The `--preserve` flag set the session parameters persistence to `PRESERVE`

#### fib

The `--fib` flag set the session parameters Ack mode to `RIB_AND_FIB_ACK`

#### input-file

The `--input-file` flag points to a modify input file

See [here](https://github.com/karimra/gribic/examples) for some input file examples

### Examples

Run all operations defined in the input-file in `single-primary` redundancy mode, with persistence `preserve` and ack mode `RIB_FIB`

```bash
gribic -a router1 -u admin -p admin --skip-verify modify \
    --single-primary \
    --preserve \
    --fib \
    --election-id 1:2 \
    --input-file <path/to/modify/operations>
```
