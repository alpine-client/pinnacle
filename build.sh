#env CGO_ENABLED=1 GOARCH=amd64 GOOS=windows \
#CC="zig cc -target x86_64-windows-gnu -Wl,--subsystem,windows -Wl,-s" \
#CXX="zig c++ -target x86_64-windows-gnu -Wl,--subsystem,windows -Wl,-s" \
#CGO_LDFLAGS="-static" \
#CPPFLAGS="-DCIMGUI_DEFINE_ENUMS_AND_STRUCTS" \
#go build -trimpath -p 4 -ldflags="-H=windowsgui -X main.version=${VERSION}" \
#-o bin/pinnacle-windows-amd64.exe .

env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
CC=x86_64-w64-mingw32-gcc \
CXX="zig c++ -target x86_64-windows-gnu -Wl,--subsystem,windows -Wl,-s" \
HOST=x86_64-w64-mingw32 \
go build -ldflags "-s -w -H=windowsgui" -p 4 -v -o pinnacle.exe
