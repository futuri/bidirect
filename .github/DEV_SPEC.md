# Developer Specification — Heart‑Shaped Layered Window (Go + Win32, MVP)

Version: 1.0  
Target platform: Windows (desktop)  
Language: Go (no CGO)  
Target Go version: Go 1.21+  
Primary architecture (MVP): windows/amd64

This document is a developer‑ready specification for the MVP: a borderless, per‑pixel alpha Windows window in the shape of a heart, implemented in Go without CGO. It describes requirements, architecture, data flows, API usage, rendering strategy, synchronization, error handling, testing plan, deliverables and an implementation checklist so a developer can begin work immediately.

Contents
- Goals & scope
- High‑level behavior & UX
- Non‑goals / constraints
- Technical choices & rationale
- System architecture & flow
- Data structures & APIs (Go signatures)
- Rendering & pixel buffer details
- Win32 integration & message handling
- Hit testing & input behavior
- Resizing / aspect ratio rules
- Menu & commands
- Resource lifecycle & cleanup
- Error handling & logging strategy
- Concurrency model & synchronization
- Performance & profiling guidance
- Testing plan (unit, integration, manual)
- CI / build / run instructions
- File layout & deliverables
- Implementation milestones / checklist
- Risks & mitigations
- Appendix: Key Win32 calls & constants

---

## Goals & scope

Deliver a lightweight, maintainable Go codebase that:

- Creates a native Windows window using WS_EX_LAYERED and UpdateLayeredWindow.
- Renders a heart shape into a 32bpp ARGB DIBSection and presents it as the window content (per‑pixel alpha).
- Smooth edges (antialias) via CPU rasterizer (supersampling/filter).
- Simple pulse animation (default) at a configurable frame rate.
- Per‑pixel hit testing: transparent pixels are click‑through.
- Movable and resizable by user; resizing keeps aspect ratio (1:1).
- Context menu (right click) with Quit, Restore size, Always on Top toggle, About.
- Appears in taskbar / Alt+Tab.
- No CGO, no high level GUI libs. Uses golang.org/x/sys/windows + syscall/ NewLazyDLL where needed.

Non‑goals (MVP)
- No remote video/streaming ingest. (Design allows future integration.)
- No GPU accelerated rendering (no Direct2D/Direct3D).
- No full multimedia codecs or complex UI widgets.

---

## High‑level behavior & UX

Default configuration
- Initial size: 400 x 400 (client area)
- Keep aspect: true (1:1)
- Min size: 150 x 150
- Max size: min(screenWidth, screenHeight) - 100 (calculated at runtime)
- Alpha threshold for hit test: 10 (0–255)
- Border grab size for resize handles: 8 px
- Default FPS for animation: 30
- Default color: Windows blue (#0078D7), configurable in code
- Animation enabled by default (pulse)

User interactions
- Drag anywhere on the visible heart (non‑transparent pixels) to move window.
- Resize by grabbing near edges/corners (8px grab zone), preserving 1:1.
- Right‑click on the heart opens context menu with actions.
- Alt+F4 closes window. Taskbar and Alt+Tab behavior preserved.

---

## Technical choices & rationale

- Go 1.21+ (modern stdlib & module behavior).
- Target windows/amd64 for MVP to simplify testing and avoid ARM edge cases.
- No CGO: use golang.org/x/sys/windows and syscall.NewLazyDLL to call User32/Gdi32 functions.
- CPU rasterizer (Go): recommended to avoid Gdiplus initialization complexity and keep implementation pure Go. Use supersampling (2×) and/or distance filter to produce anti‑aliased edges.
- UpdateLayeredWindow DIBSection (32bpp ARGB) for per‑pixel alpha composition.

Why these choices:
- Minimal dependencies, easier to build and ship as a single Go binary.
- Full control over rendering algorithm; easy to test and adjust.
- Avoids GDI+ lifetime, threading and initialization pitfalls.

---

## System architecture & flow

Components
- main package: app entry, configuration parsing (flags), create & run window.
- window package: Win32 registration, message loop, DIBSection management, UpdateLayeredWindow calls, hit testing, WM handling, menu handling.
- renderer package: CPU rasterizer producing ARGB buffer, supports static heart and animated pulse; exposes frame producer interface.
- util package: math helpers (heart shape signed distance or vector path), color helpers, screen metrics.
- tests: unit tests for renderer math & rasterization.

Runtime flow (summary)
1. main parses flags/config and registers window class.
2. CreateWindowEx with WS_EX_LAYERED | WS_EX_APPWINDOW | WS_EX_TOPMOST optional handling.
3. CreateDIBSection (32bpp ARGB) to obtain pixel buffer pointer and HBITMAP; create compatible DC and select the HBITMAP into it.
4. Spawn renderer goroutine producing frames at desired FPS. Renderer writes into a Go-managed buffer (or into the DIBSection memory directly if safe), and notifies the UI thread when new frame is ready via a channel.
5. UI (owner) thread receives frame-ready event and calls UpdateLayeredWindow with BLENDFUNCTION {AC_SRC_ALPHA} to commit pixels to screen.
6. WM_NCHITTEST uses the current pixel buffer alpha value to return HTTRANSPARENT when alpha < threshold, otherwise HTCLIENT or HTCAPTION depending on border proximity and resizing logic.
7. WM_SIZING and WM_GETMINMAXINFO enforce the 1:1 aspect and min/max size rules.
8. Context menu via CreatePopupMenu / TrackPopupMenu / WM_COMMAND handling.
9. On WM_DESTROY, free DCs, DeleteObject HBITMAP, and PostQuitMessage.

Important threading note: UpdateLayeredWindow and many Win32 window operations are safest when called from the same thread that created the window (the UI thread). The renderer must therefore produce pixel data and send it to the UI thread; the UI thread must perform UpdateLayeredWindow.

---

## Data structures & Go API outlines

Configuration struct
```go
type Config struct {
    InitialSize           int    // 400
    KeepAspect            bool   // true
    MinSize               int    // 150
    AlphaThreshold        uint8  // 10
    BorderGrabSize        int    // 8
    FPS                   int    // 30
    HeartColor            color.NRGBA // e.g. #0078D7
    AnimationEnabled      bool
    WindowTitle           string
}
```

Window state
```go
type Window struct {
    hwnd        windows.HWND
    hdcMem      windows.HDC
    hBitmap     windows.HBITMAP
    dibPixels   []byte // slice backed by pointer from CreateDIBSection (BGRA or RGBA ordering documented)
    width, height int

    cfg         Config
    mu          sync.RWMutex // protects dibPixels + width/height
    frameCh     chan *Frame   // frames from renderer
    quitCh      chan struct{}
    isTopmost   bool
}
```

Renderer API
```go
type Frame struct {
    Width  int
    Height int
    Pixels []byte // ARGB/BGRA byte order; document the expected layout
    Timestamp time.Time
}

type Renderer interface {
    Start(ctx context.Context, out chan<- *Frame) error
    Stop() error
}
```

Functions (important)
- NewWindow(cfg Config) (*Window, error)
- (*Window) Run() error // registers class, creates window, starts message loop (blocks)
- (*Window) applyFrame(frame *Frame) error // called on UI thread to call UpdateLayeredWindow
- createDIBSection(width, height) (hdcMem, hBitmap, pixelsPtr, err)
- destroyResources()
- rasterizeHeart(width, height, color, aaFactor) -> []byte

---

## Rendering & pixel buffer details

Pixel format
- Use 32bpp (8 bits per channel) with alpha: BGRA or ARGB depending on CreateDIBSection usage. Decide and document exactly which ordering is used (GDI typically uses BGRA in memory for 32bpp). For UpdateLayeredWindow, provide pointer to DIBSection. We'll use BGRA order and ensure the renderer writes in that layout.

Renderer approach (MVP)
- CPU rasterizer implementing heart shape SDF (signed distance field) or path rasterization.
- Anti‑aliasing: 2× supersampling (render at 2× resolution and downsample) or compute per‑pixel coverage from SDF and use smoothstep to compute alpha.
- Output: width x height pixel buffer with pre‑multiplied alpha? UpdateLayeredWindow expects per‑pixel alpha; BLENDFUNCTION with AC_SRC_ALPHA uses straight alpha (not premultiplied). Using straight alpha is simpler. Confirm behavior on test: use AC_SRC_ALPHA and supply straight alpha.
- Animation: pulse changes alpha multiplier or color brightness over time (sinusoidal with small amplitude). Renderer should accept an animation parameter (0..1) to adjust alpha.

Renderer details
- Provide deterministic function: for pixel center (x+0.5,y+0.5), compute distance to heart shape implicit function. Map distance to alpha via a smoothstep function using an edge width (e.g., 1.5 px). Use 2× supersample if further smoothing required.
- For efficiency: keep lookups and calculations minimal; avoid allocations per frame (reuse buffers).

Buffer ownership & lifecycle
- Preferred approach: allocate DIBSection via CreateDIBSection and obtain pointer to memory. Create a Go slice header pointing at that memory region using unsafe (but do not free it directly). The renderer writes into a separate Go-managed buffer (safe to write in goroutine), then the UI thread copies or memmove to the DIBSection memory before UpdateLayeredWindow. That adds a copy but avoids concurrent writes to the DIBSection memory.
- Alternate (safe) approach: allocate two buffers and swap with proper locking; but UpdateLayeredWindow must use the DIBSection memory or HBITMAP; so final copy to DIBSection is still required, or use DIBSection pointer directly only from UI thread.

Recommendation (MVP): Renderer writes into a reusable Go []byte buffer (frame.Pixels). When frame ready, send pointer to frame to UI thread. UI thread obtains write lock, copies bytes into DIBSection memory, then calls UpdateLayeredWindow. This avoids concurrent writes to the DIBSection and keeps UpdateLayeredWindow on the UI thread.

Memory layout and copy:
- pixelCount = width * height
- bytesPerPixel = 4
- bytes = pixelCount * 4
- Use copy(dibSlice, frame.Pixels) — ensure both slices refer to same pixel ordering.

---

## Win32 integration & message handling

Window creation
- RegisterClassEx + CreateWindowEx
- Extended styles: WS_EX_LAYERED | WS_EX_APPWINDOW (to show in taskbar) | optionally WS_EX_TRANSPARENT is *not* used globally; hit testing handles transparency.
- Window styles: WS_POPUP | WS_VISIBLE (no border)

DIB & Update
- CreateDIBSection with BITMAPINFO where biBitCount = 32, biCompression = BI_RGB.
- CreateCompatibleDC and SelectObject(HBITMAP).
- When applying frame: use UpdateLayeredWindow with POINT (0,0)/size (width,height), hdcMem, and BLENDFUNCTION with SourceConstantAlpha = 255 and AlphaFormat = AC_SRC_ALPHA.

Key Win32 messages to handle
- WM_NCHITTEST — return HTTRANSPARENT for transparent pixels (alpha < threshold), otherwise return HTCAPTION or border codes for resize.
- WM_SIZING — enforce 1:1 aspect ratio by modifying RECT in place (lParam pointer to RECT).
- WM_GETMINMAXINFO — set min/max size (minWidth/minHeight and max tracking size based on screen resolution).
- WM_RBUTTONUP — show context menu.
- WM_COMMAND — handle menu selections.
- WM_DESTROY — cleanup resources and PostQuitMessage.
- WM_MOUSEMOVE, WM_LBUTTONDOWN/UP as needed (for custom behaviors), but dragging is achievable via HTCAPTION returned from WM_NCHITTEST.

Hit testing algorithm (WM_NCHITTEST)
- Convert mouse screen point to client coordinates (ScreenToClient).
- Map client coords to bitmap pixel coordinates considering scaling (if any).
- Lock buffer with mu.RLock() and read alpha byte at pixel index. If alpha < AlphaThreshold (10) return HTTRANSPARENT.
- If alpha >= threshold:
  - If cursor within BorderGrabSize from edges, return respective HTLEFT/HTRIGHT/HTTOP/HTBOTTOM or corner codes like HTTOPLEFT.
  - Else return HTCAPTION to allow dragging.

Resizing (WM_SIZING & WM_GETMINMAXINFO)
- In WM_SIZING, get pointer to RECT, compute width/height and enforce width==height by adjusting sides depending on which edge is being moved (wParam indicates edge).
- On WM_GETMINMAXINFO, set minTrackSize (150) and maxTrackSize (screenMin-100).

Context menu
- CreatePopupMenu -> AppendMenu items:
  - IDM_QUIT
  - IDM_RESTORE_SIZE
  - IDM_TOGGLE_TOPMOST
  - IDM_ABOUT
- On TrackPopupMenu, call SetForegroundWindow(hwnd) before TrackPopupMenu (required by Win32 for popup menu).
- WM_COMMAND receives LOWORD(wParam) menu id; handle accordingly.

Always On Top
- Toggle SetWindowPos(hwnd, HWND_TOPMOST/HWND_NOTOPMOST, SWP_NOMOVE|SWP_NOSIZE)

Taskbar/Alt+Tab
- To appear in taskbar, include WS_EX_APPWINDOW; avoid WS_EX_TOOLWINDOW.

---

## Resource lifecycle & cleanup

- Always track created GDI objects and DCs and ensure they are deleted (DeleteObject(HBITMAP), DeleteDC, ReleaseDC where appropriate).
- On errors during initialization, free any already created resources to avoid leaks.
- On WM_DESTROY:
  - Signal renderer to stop (close quitCh).
  - DeleteObject(hBitmap)
  - DeleteDC(hdcMem)
  - PostQuitMessage(0)
- Use defer whenever possible for allocated resources in initialization routines.
- Keep a small function to log and attempt cleanup for partially initialized resources.

---

## Error handling & logging strategy

- Centralized logging: use standard log package or a lightweight wrapper for levelled logs (INFO/WARN/ERROR).
- Fail fast for unrecoverable init errors: return error from NewWindow or main init and exit with non‑zero code.
- For transient rendering errors: log but continue (e.g., if a frame fails to apply, skip frame).
- Validate return values of Windows API calls (non‑NULL/zero checks). Wrap Windows calls in small helper functions that convert error codes to Go errors with context.
- Example error pattern:
  - if ptr == 0 { return fmt.Errorf("CreateDIBSection failed: %v", err) }
- For UpdateLayeredWindow failures, log error and try again on next frame; if persistent, provide an error indicator and allow graceful shutdown.

---

## Concurrency model & synchronization

Threads/goroutines
- UI thread: main goroutine that creates the window and runs GetMessage/DispatchMessage loop; must perform all Win32 calls that require window thread context (UpdateLayeredWindow, TrackPopupMenu, ShowWindow, SetWindowPos).
- Renderer goroutine: runs independently producing Frame objects at configured FPS.
- Optional watchdog goroutine: monitors renderer health and restarts if necessary.

Communication
- frameCh chan *Frame: renderer sends completed frames to UI thread.
- quitCh chan struct{}: signals renderer to stop when shutting down.
- mu sync.RWMutex: protects reads/writes to shared state (dibPixels backing memory, width/height). UI thread typically holds write lock when copying into DIBSection, WM_NCHITTEST obtains RLock when reading alpha.

Memory safety
- No concurrent writes to DIBSection memory. UI thread performs final copy into DIBSection right before UpdateLayeredWindow.
- Renderer must not write into DIBSection memory directly unless synchronization ensures UI thread not reading it simultaneously.

Performance optimization
- Minimize allocations per frame: reuse frame buffer slices.
- If copy cost becomes bottleneck, consider double buffering where renderer writes directly to DIBSection but only when UI thread yields (complex), or use Lock/Unlock pattern with mutex and ensure UpdateLayeredWindow called while holding data stable.

---

## Performance targets & profiling

Target
- Default animation: 30 FPS. UpdateLayeredWindow only when frame actually changes (or on every tick for animation).
- Aim for CPU cost < ~5–10% on modern desktop when animation enabled; this is heuristic and depends on renderer.

Profiling
- Use pprof CPU and heap profiling to find hotspots.
- Measure time for:
  - rasterize -> frame generation (renderer goroutine)
  - copy into DIBSection (memcopy)
  - UpdateLayeredWindow call (Win32)
- If UpdateLayeredWindow becomes dominant:
  - Reduce FPS, update only regions that change (complex), or reduce resolution.
- If rasterizer is dominant:
  - Optimize SDF math, reduce supersample factor, precompute static masks where possible.

---

## Testing plan

Unit tests (automated)
- renderer:
  - Tests for heart SDF: known coordinates inside/outside the shape return expected sign.
  - Alpha mapping: edge smoothing produces intermediate alpha values (assert ranges).
  - Rasterization stability: identical inputs produce identical outputs.
  - Benchmarks: BenchmarkRasterize for several sizes (150, 400, 800) to gauge performance.
- util:
  - Pixel coordinate mapping tests (client->bitmap mapping, scaling).

Integration tests (manual and automated)
- CI build on windows-latest runs go test (unit) and build the binary.
- Automated smoke test (optional): use PowerShell script on windows-latest to run the executable and verify the process exists and window appears via UIAutomation (if configured). Minimal automation is acceptable for MVP.

Manual QA checklist
- Build & run on Windows 10 and 11 (x64), verify:
  - Window appears in taskbar and Alt+Tab.
  - Heart shape rendered with smooth edges.
  - Transparent areas are click‑through (try clicking a desktop icon under the transparent area).
  - Dragging (click+drag) moves window.
  - Resizing preserves 1:1 and stops at min/max sizes.
  - Right click shows context menu and commands work (Quit, Restore, Topmost, About).
  - Alt+F4 closes the window.
  - Toggling Always On Top behaves correctly.
  - No GDI leakage (monitor with Process Explorer for GDI handles during repeated open/close: GDI handle count shouldn't grow).

Edge cases to verify
- Resize to minimum and back.
- Rapid resize and drag operations — monitor for races, glitches.
- Multiple monitors with different DPI settings — verify mapping pixel coordinates correctly. (Note: MVP may not fully support DPI scaling; document as next step.)

Fuzz / regression tests
- WM_NCHITTEST across many points to ensure no surprising HTTRANSPARENT returns.
- Stress test: run animation for several minutes and monitor CPU & memory.

---

## CI / build / run

Recommended CI: GitHub Actions (windows-latest runner)

Example workflow steps
- Setup Go 1.21
- go vet, go test ./...
- go build -ldflags "-s -w" -o dist/heart.exe ./cmd/heart
- artifact upload

Build locally
- On Windows with Go 1.21:
  - go build -o heart.exe ./cmd/heart
- Cross compile on non-Windows:
  - GOOS=windows GOARCH=amd64 go build -o heart.exe ./cmd/heart

Run
- heart.exe [flags]
- Provide flags for enabling/disabling animation, toggling fps, and colors.

---

## File layout & deliverables

Suggested repository layout
- cmd/heart/main.go — entrypoint with flags and config
- internal/window/window.go — Win32 window registration, message loop, DIB management
- internal/renderer/renderer.go — rasterizer logic, animation loop
- internal/renderutil/heart.go — SDF & shape math
- internal/config/config.go — Config struct & defaults
- internal/logging/log.go — small logging wrapper
- assets/ (if any)
- README.md — build/run instructions, behavior notes
- docs/ — architecture notes, API docs
- tests/ — unit test helpers
- .github/workflows/ci.yml — build/test CI

Deliverables for MVP
- Working binary for Windows x64 built from repo.
- Source code with documented public functions and a README.
- Basic unit tests for renderer math.
- CI pipeline to build and test on Windows runner.

---

## Implementation milestones / checklist

HITO 1 — Basic window + static heart (3–5 days)
- [ ] Project skeleton + config defaults.
- [ ] Register Win32 class and CreateWindowEx with WS_EX_LAYERED, WS_EX_APPWINDOW.
- [ ] CreateDIBSection and HBITMAP selection into memory DC.
- [ ] Implement CPU rasterizer to produce a static heart image.
- [ ] Implement UpdateLayeredWindow call on UI thread to present the static heart.
- [ ] Implement WM_NCHITTEST to return HTTRANSPARENT for alpha < 10 and HTCAPTION else.
- [ ] Context menu minimal: Quit, Restore size.
- [ ] Manual basic QA.

HITO 2 — Polishing (2–3 days)
- [ ] Implement 2× supersampling or SDF-based antialiasing.
- [ ] Add pulse animation at configurable FPS.
- [ ] Implement full context menu (Always On Top, About).
- [ ] Implement resizable behavior with border grab size 8 and WM_SIZING to enforce 1:1.
- [ ] Add min/max size enforcement via WM_GETMINMAXINFO.
- [ ] Ensure DIB/GDI resource lifecycle properly cleaned.

HITO 3 — Hardening & tests (2 days)
- [ ] Unit tests for rendering math and rasterization.
- [ ] CI build with windows-latest.
- [ ] Performance profiling and minor optimizations.
- [ ] Documentation and README.

Total MVP estimate: 1–2 weeks for 1 experienced Go engineer with Win32 familiarity.

---

## Risks & mitigations

1. UpdateLayeredWindow CPU cost when updating full window each frame
   - Mitigation: limit animation FPS (default 30), only UpdateLayeredWindow when visual changes, consider partial updates later.

2. Anti‑alias/rasterization complexity
   - Mitigation: start with simple 2× supersample; optimize SDF later if needed; fallback to GDI+ later (not recommended in MVP).

3. Hit testing race situations
   - Mitigation: renderer writes to separate buffer; UI thread copies into DIBSection under lock before UpdateLayeredWindow. WM_NCHITTEST reads using RLock.

4. GDI/HBITMAP/DC leaks
   - Mitigation: explicit tracking and cleanup, unit tests, and manual verification via Process Explorer.

5. DPI / scaling on multi-monitor setups
   - Mitigation: document as caveat in MVP; consider mapping screen -> client coords scaling using GetDpiForWindow in follow‑up.

---

## Appendix: Key Win32 APIs & constants to call

APIs (via syscall.NewLazyDLL or golang.org/x/sys/windows wrappers)
- User32:
  - RegisterClassExW
  - CreateWindowExW
  - DefWindowProcW
  - ShowWindow
  - UpdateLayeredWindow
  - GetCursorPos / ScreenToClient
  - TrackPopupMenu / SetForegroundWindow / CreatePopupMenu / AppendMenuW
  - SetWindowPos
  - PostQuitMessage
- Gdi32:
  - CreateCompatibleDC
  - CreateDIBSection
  - SelectObject
  - DeleteObject
  - DeleteDC
- Messages:
  - WM_NCHITTEST, WM_SIZING, WM_GETMINMAXINFO, WM_COMMAND, WM_DESTROY, WM_RBUTTONUP, WM_LBUTTONDOWN/UP
- Blend flags:
  - AC_SRC_ALPHA

Important constants
- Window styles: WS_EX_LAYERED, WS_EX_APPWINDOW, WS_POPUP, WS_VISIBLE
- Hit test codes: HTTRANSPARENT, HTCAPTION, HTLEFT, HTRIGHT, HTTOP, HTBOTTOM, HTTOPLEFT, etc.
- SWP_NOMOVE, SWP_NOSIZE, HWND_TOPMOST, HWND_NOTOPMOST

---

If you want, I can now:
- Generate an initial skeleton repository with the main.go, window.go, renderer.go stubs and comments for each TODO; or
- Generate a complete single-file MVP (main.go) implementing the creation, DIBSection, simple rasterizer (static heart + pulse), UpdateLayeredWindow, WM_NCHITTEST and context menu ready to build on Windows x64.

Which do you prefer?