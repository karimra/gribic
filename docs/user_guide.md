# Flags


## Global Flags

### config

The `--config` flag specifies the location of a configuration file that `gribic` will read. 

If not specified, gribic searches for a file named `.gribic` with extensions `yaml, yml, toml or json` in the following locations:

* `$PWD`
* `$HOME`
* `$XDG_CONFIG_HOME`
* `$XDG_CONFIG_HOME/gribic`

### address

The address flag `[-a | --address]` is used to specify the gRIBI server address in address:port format, for e.g: `192.168.113.11:57400`

Multiple target addresses can be specified, either as comma separated values:

```bash
gribic --address 192.168.113.11:57400,192.168.113.12:57400 
```

or by using the `--address` flag multiple times:

```bash
gnmic -a 192.168.113.11:57400 --address 192.168.113.12:57400
```

The port number can be omitted, in which case the value fro m the flag --port will be appended to the address

### username

The username flag `[-u | --username]` is used to specify the target username as part of the user credentials

### password

The password flag `[-p | --password]` is used to specify the target password as part of the user credentials.

### port

### insecure

The insecure flag `[--insecure]` is used to indicate that the client wishes to establish an non-TLS enabled gRPC connection.

To disable certificate validation in a TLS-enabled connection use [`skip-verify`](#skip-verify) flag.

### skip-verify

The skip verify flag `[--skip-verify]` indicates that the target should skip the signature verification steps, in case a secure connection is used.  

### tls-ca

The TLS CA flag `[--tls-ca]` specifies the root certificates for verifying server certificates encoded in PEM format.

### tls-cert

The tls cert flag `[--tls-cert]` specifies the public key for the client encoded in PEM format.

### tls-key

The tls key flag `[--tls-key]` specifies the private key for the client encoded in PEM format.

### timeout

The timeout flag `[--timeout]` specifies the gRPC timeout after which the connection attempt fails.

Valid formats: 10s, 1m30s, 1h.  Defaults to 10s

### debug

The debug flag `[-d | --debug]` enables the printing of extra information when sending/receiving an RPC

### proxy-from-env

The proxy-from-env flag `[--proxy-from-env]` indicates that the gribic should use the HTTP/HTTPS proxy addresses defined in the environment variables `http_proxy` and `https_proxy` to reach the targets specified using the `--address` flag.

### format

### election-id

The Election ID flag `--election-id` is used to specify the election ID used with the Flush and Modify RPCs

It takes a string in the format `high:low` where both high and low are uint64 forming a uint128 election ID value.

`:`, `1:` and `:1` are valid values.

## Targets
