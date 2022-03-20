# Welcome to gRIBIc

`gRIBIc` is a gRIBI CLI client that implements the Openconfig gRIBI RPCs.
It is intended to be used for educational and testing purposes.

## Features

* **Full Support for Get And Flush RPCs**

* **Modify RPC is supported with IPv4, IPv6, Next Hop Group and Next Hop AFTs**

* **Template based modify RPC operations configuration**

* **Concurrent multi target RPC execution**

## Quick start guide

### Installation

```
bash -c "$(curl -sL https://get-gribic.kmrd.dev)"
```

### Get Request

Query all AFTs in all network instances

```bash
gribic -a router1 -u admin -p admin --skip-verify get
```

Query all AFTs in network instance `default`

```bash
gribic -a router1 -u admin -p admin --skip-verify get --ns default
```

Query AFT type `ipv4` in network instance `default`

```bash
gribic -a router1 -u admin -p admin --skip-verify get --ns default --aft ipv4
```

Query AFT type `nhg` (next hop group) in all network instances

```bash
gribic -a router1 -u admin -p admin --skip-verify get --aft nhg
```

### Flush Request

Flush all AFTs in network instance `default`

```bash
gribic -a router1 -u admin -p admin --skip-verify flush --ns default 
```

Flush all AFTs in all network instances

```bash
gribic -a router1 -u admin -p admin --skip-verify flush --ns-all
```

### Modify Request

Run all operations defined in the input-file in `single-primary` redundancy mode, with persistence `preserve` and ack mode `RIB_FIB`

```bash
gribic -a router1 -u admin -p admin --skip-verify modify \
    --single-primary \
    --preserve \
    --fib \
    --election-id 1:2 \
    --input-file <path/to/modify/operations>
```
