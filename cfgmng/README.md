# cfgmng - Simple Configuration Manager for Go
cfgmng is a lightweight Go package that simplifies loading configuration from YAML files using viper. It provides a generic function to automatically load configuration into any struct.

## Installation
```sh
go get github.com/yourusername/cfgmng
```
## Usage
1. Define Your Configuration Struct
   Create a struct representing your configuration fields and use struct tags to map them to the YAML file.

```go
package main

import (
"fmt"
"log"

	"github.com/seb7887/gofw/cfgmng"
)

type AppConfig struct {
AppName string `mapstructure:"app_name"`
Port    int    `mapstructure:"port"`
Debug   bool   `mapstructure:"debug"`
}

func main() {
cfg, err := cfgmng.LoadConfig[AppConfig](".", "config")
if err != nil {
log.Fatalf("Failed to load configuration: %v", err)
}

	fmt.Println("App Name:", cfg.AppName)
	fmt.Println("Port:", cfg.Port)
	fmt.Println("Debug Mode:", cfg.Debug)
}
```

2. Create a YAML Configuration File
   Save the following as config.yaml in your project root:

```yaml
app_name: "MyApp"
port: 8080
debug: true
```
## Features
✅ Generic Implementation – Works with any struct.

✅ YAML-Based Configuration – Load settings from a config.yaml file.

✅ Environment Variable Support – Allows overriding values using environment variables.

✅ Minimal API – Just call LoadConfig[T](path, filename) and get your structured config.

## Error Handling
If the configuration file is missing or cannot be parsed, LoadConfig will return an error:

```go
cfg, err := cfgmng.LoadConfig[AppConfig](".", "config")
if err != nil {
log.Fatalf("Failed to load configuration: %v", err)
}
```

## License
This project is licensed under the MIT License.