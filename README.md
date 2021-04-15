# svgg

```svgg``` is an automated SVG path string parser and rendering tool. It is based on the package [```srwiley/oksvg```](https://github.com/srwiley/oksvg) but modified to draw directly to an SVG context using the [```fogleman/gg```](https://github.com/fogleman/gg) rendering engine.

*Warning*: This is a work in progress. Only M line commands are currently implemented.

## Installation

```bash
go get github.com/engelsjk/svgg
```

## Usage

*Note*: ```svgg``` does not include an SVG/XML parser, so you'll need a way to first extract a path string.

```go

dc := gg.NewContext(150, 200)

parser := svgg.NewParser(dc)

dpath := "M75 0, 0 200, 150 200 Z"

parser.CompilePath(dpath)

dc.SetRGB(0, 0, 0)
dc.Fill()

dc.SavePNG("image.png")
```

![](images/demo.png)