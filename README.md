# Nginx Configuration Parser

## Example

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	nginxparser "github.com/faceair/nginx-parser"
)

func main() {
	directives, err := nginxparser.New(nil).ParseFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	body, err := json.MarshalIndent(directives, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}
```

## License

[MIT](LICENSE)
