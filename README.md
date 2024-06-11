# Flutties

Is a [pixelflut](github.com/defnull/pixelflut) server written in Go.

## Features

supports the pixelflut implementation
- `HELP`: displays a help message
- `SIZE`: return the size of the canvas
- `PX <x> <y>`: get the color of a pixel
- `PX <x> <y> rrggbb(aa)`: set the color of a pixel

It also supports some extra commands
- `PX <x> <y> ww`: set the color of a pixel to a grey value
- `IPX`: all the same as PX. but for the icoflut
- `ISIZE`: gives the size of the icoflut canvas
