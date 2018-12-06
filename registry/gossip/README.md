# Gossip Registry

Gossip is a zero dependency registry which uses hashicorp/memberlist to broadcast registry information 
via the SWIM protocol. 

## Usage

Start with the registry flag or env var

```bash
MICRO_REGISTRY=gossip go run service.go
```

On startup you'll see something like

```bash
2018/12/06 18:17:48 Registry Listening on 192.168.1.65:56390
```

To join this gossip ring set the registry address using flag or env var

```bash
MICRO_REGISTRY_ADDRESS= 192.168.1.65:56390 
```
