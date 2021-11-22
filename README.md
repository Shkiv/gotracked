# gotracked
Simple and fast working time tracker.

By default server listens `localhost:8080`

Methods:
* `/start` - start tracking
* `/stop` - stop tracking
* `/intervals` - show tracked intervals

## Building and running
### Linux
1. Install Go
2. Install GCC
3. Use `go run .` and `go build .` to run or build

### Windows
(one of the possible ways)
1. Install Go from [go.dev](https://go.dev/)
2. Install [MSYS2](https://www.msys2.org/)
3. Run MinGW and install GCC there: `pacman -S mingw-w64-x86_64-gcc`
4. Add `C:\msys64\mingw64\bin` and `C:\msys64\usr\bin` to Path environment variable
5. Use `go run .` and `go build .` from PowerShell or MinGW