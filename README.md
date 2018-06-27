# jstream

`jstream` is a streaming JSON parser and value extraction library for Go.

Unlike most JSON parsers, `jstream` is document position- and depth-aware, enabling the extraction of values at a specified depth, and eliminating the overhead of allocating encompassing arrays or objects; e.g:

```json
[
  {
    "desc": "RGB",
    "colors": [ "red", "green", "blue" ]
  },
  {
    "desc": "CMYK",
    "colors": [ "cyan", "magenta", "yellow", "black" ]
  }
]
```

Using the above example document, we can choose to extract and act only the objects within the top-level array:
```go
f, _ := os.Open("input.json")
decoder := jstream.NewDecoder(f, 1) // extract JSON values at a depth level of 1
for mv := range decoder.Stream() {
	fmt.Printf("%v\n ", mv.Value)
}
```

output:
```
map[desc:RGB colors:[red green blue]]
map[desc:CMYK colors:[cyan magenta yellow black]]
```

likewise, increasing depth level to `3` yields:
```
red
green
blue
cyan
magenta
yellow
black
```

## Installing 

```bash
go get github.com/bcicen/jstream
```

## Commandline

`jstream` comes with a cli tool for quick viewing of parsed values from JSON input:

```bash
cat input.json | jstream -v -d 1
depth	start	end	type   | value

1	004	069	object | {"colors":["red","green","blue"],"desc":"RGB"}
1	073	153	object | {"colors":["cyan","magenta","yellow","black"],"desc":"CMYK"}
```

### Options

Opt | Description
--- | ---
-d \<n\> | emit values at depth n. if n < 0, all values will be emitted
-v | output depth and offset details for each value
-h | display help dialog

## Benchmarks

Obligatory benchmarks performed using two file sizes -- regular (1.6mb, 1000 objects) and large (128mb, 100000 objects)

input size | lib | MB/s | alloc
--- | --- | --- | ---
regular | standard | 97 | 3.6MB
regular | jstream | 175 | 2.1MB
large | standard | 92 | 305MB
large | jstream | 404 | 69
