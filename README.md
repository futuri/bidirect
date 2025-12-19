# BiDirect

A native Windows window that receives streaming content via UDP with per-pixel alpha (UpdateLayeredWindow).

## Build

```bash
GOOS=windows GOARCH=amd64 go build -o bidirect.exe ./cmd/heart
```

## Run

```bash
./bidirect.exe
```

### Flags

- `-fps=30` - Animation FPS
- `-animation=true` - Enable/disable pulse animation
- `-size=400` - Initial window size
- `-udp=false` - Enable UDP streaming receiver
- `-port=5555` - UDP port

## Features

- UDP streaming receiver for real-time content
- Per-pixel hit testing (transparent pixels are click-through)
- Draggable by clicking on the content
- Resizable with 1:1 aspect ratio maintained
- Right-click context menu: Quit, Restore Size, Always On Top, About
- Appears in taskbar and Alt+Tab
- Supports WebP, JPEG, PNG, GIF formats

## Test

```bash
go test ./...
```
