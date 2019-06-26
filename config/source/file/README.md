# File Source

The file source reads config from a file. 

It uses the File extension to determine the Format e.g `config.yaml` has the yaml format. 
It does not make use of encoders or interpet the file data. If a file extension is not present 
the source Format will default to the Encoder in options.

## Example

A config file format in json

```json
{
    "hosts": {
        "database": {
            "address": "10.0.0.1",
            "port": 3306
        },
        "cache": {
            "address": "10.0.0.2",
            "port": 6379
        }
    }
}
```

## New Source

Specify file source with path to file. Path is optional and will default to `config.json`

```go
fileSource := file.NewSource(
	file.WithPath("/tmp/config.json"),
)
```

## File Format

To load different file formats e.g yaml, toml, xml simply specify them with their extension

```
fileSource := file.NewSource(
        file.WithPath("/tmp/config.yaml"),
)
```

If you want to specify a file without extension, ensure you set the encoder to the same format

```
e := toml.NewEncoder()

fileSource := file.NewSource(
        file.WithPath("/tmp/config"),
	source.WithEncoder(e),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load file source
conf.Load(fileSource)
```

