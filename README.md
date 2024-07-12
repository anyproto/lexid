
# Lexid

Lexid is a Go library for generating and managing lexicographically sorted strings. It supports customizable character sets and configurable block sizes, allowing for flexible and efficient string generation.

## Features

- Generate lexicographically sorted strings
- Support for customizable character sets
- Configurable block and step sizes

## Installation

To install the library, use:

```sh
go get github.com/anyproto/lexid
```

## Usage


### Configuration

#### blockSize

`blockSize` determines the initial length of the strings. If you use a small value (e.g., 2 or 3), you will get shorter strings initially, but new blocks will be added more frequently as capacity ends. This means if you plan to have a small number of strings, a smaller block size is preferable. For larger datasets (e.g., millions of values), a block size of 4-6 is more suitable.

#### stepSize

`stepSize` controls the increment between successive strings. A larger `stepSize` will make the sequence more sparse, allowing for the insertion of more strings between existing strings without increasing the size of the result. This is useful for creating strings that are spread out more widely in the lexicographical order.

### Example

```go
package main

import (
    "fmt"
    "log"
    "github.com/anyproto/lexid"
)

func main() {
    var blockSize = 3
    var stepSize = 10 

    lid, err := lexid.New(lexid.CharsAlphanumeric, blockSize, stepSize)
    if err != nil {
        log.Fatalf("Error creating Lexid: %v", err)
    }

    // Generate the next string
    firstStr := lid.Next("")
    fmt.Println(firstStr) // Output: "001"
	
    secondStr := lid.Next(firstStr)
    fmt.Println(secondStr) // Output: "00b"


    // Generate a string before another
    nextBeforeStr, err := lid.NextBefore(firstStr, secondStr)
    if err != nil {
        log.Fatalf("Error generating NextBefore string: %v", err)
    }
    fmt.Println(nextBeforeStr) // Output: "003"
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE.md) file for details.
