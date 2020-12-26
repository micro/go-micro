# Vault Source

The vault source reads config from different secret engines in a Vault server. For example:
```
kv: secret/data/<my/secret>
database credentials: database/creds/<my-db-role>
```

## New Source

Specify source with data

```go
vaultSource := vault.NewSource(
	// mandatory: it specifies server address. 
	// It could have different formats:
	// 127.0.0.1 -> https://127.0.0.1:8200
	// http://127.0.0.1 -> http://127.0.0.1:8200
	// http://127.0.0.1:2233
	vault.WithAddress("http://127.0.0.1:8200"),
	// mandatory: it specifies a resource to been access
	vault.WithResourcePath("secret/data/my/secret"),
    // mandatory: it specifies a resource to been access
	vault.WithToken("<my-token>"),
	// optional: path to store my secret.
	// By default use resourcePath value 
	vault.WithSecretName("my/secret"),
	// optional: namespace.
    vault.WithNameSpace("myNameSpace"),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load file source
conf.Load(vaultSource)
```
