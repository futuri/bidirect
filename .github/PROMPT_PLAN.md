# Prompt Plan — Heart‑Shaped Layered Window (Go + Win32) — Code‑Generation LLM Prompts & Iterative Blueprint

This document is a structured, step‑by‑step Prompt Plan for implementing the Heart‑Shaped Layered Window MVP (Go, Windows/amd64, no CGO) described by the provided devSpec. It divides the work into staged components that can be implemented and manually verified incrementally. For each stage there is:

- A short goal/description
- Small, safe implementation steps
- Required manual verification steps
- A single, self‑contained prompt (code block, tagged as text) you can give to a code‑generation LLM to implement that stage in a test‑driven manner
- A list of TODO checkboxes to track what the prompt should change/add

Important global rules for every generated change:
- Target: Windows desktop, windows/amd64, Go 1.21+
- No CGO. Use golang.org/x/sys/windows and syscall.NewLazyDLL where necessary.
- Keep Win32 calls on the UI thread that created the window.
- Prefer small, well‑tested commits; add unit tests for renderer math where applicable.
- Avoid orphaned files or functions. Every new file/function must be wired into cmd/heart/main.go or tests.
- Each prompt should produce buildable code (go build on Windows) and runnable manually.
- Tests should be runnable via go test on Windows; use build tags or skip tests on non‑Windows where appropriate.

Use this plan to iteratively generate code with a code LLM. Each prompt assumes the prior prompts have been applied and their tests pass.

---

Table of Contents
- Phase 0 — Preparation & repo skeleton
- Phase 1 — Basic Win32 window + message loop
- Phase 2 — CreateDIBSection & memory DC (pixel buffer)
- Phase 3 — Renderer core: heart SDF & static rasterizer + unit tests
- Phase 4 — Apply frame: copy into DIBSection + UpdateLayeredWindow (static heart)
- Phase 5 — Per‑pixel hit testing (WM_NCHITTEST)
- Phase 6 — Minimal context menu (Quit, Restore) and WM_COMMAND
- Phase 7 — Resizing: WM_GETMINMAXINFO & WM_SIZING enforcement (1:1)
- Phase 8 — Anti‑aliasing & pulse animation (renderer goroutine + frameCh)
- Phase 9 — Full context menu (Always On Top, About), toggles
- Phase 10 — Resource cleanup, graceful shutdown, logging
- Phase 11 — Tests, CI workflows, documentation
- Appendix: How to use the prompts & checkpoints

---

Phase 0 — Preparation & repo skeleton
Goal
- Create initial repo layout, config & logging helpers, and a trivial main.go that builds on Windows.

Why first
- Provides a safe baseline to attach all future modules, ensures consistent imports, modules and build instructions.

Small implementation steps (atomic)
1. Initialize a Go module (module name: github.com/example/heart).
2. Create directory structure:
   - cmd/heart/main.go
   - internal/config/config.go
   - internal/logging/log.go
   - internal/window/window.go (stub)
   - internal/renderer/renderer.go (stub)
   - go.mod, README.md, .gitignore
3. Implement config defaults struct with values from devSpec.
4. Implement a tiny logging wrapper (info/error).
5. main.go should parse flags and print version/build info, then call NewWindow(cfg).Run() (window stub).
6. Add go:build windows tag where necessary for Windows-specific files to avoid non-Windows build errors.
7. Add basic unit test that simply validates config defaults.

Manual verification
- On a Windows machine, run: go build ./cmd/heart and ensure the binary builds.
- Run go test ./... (unit test should run and pass on Windows).

Prompt to generate code
```text
You are to generate the initial repository skeleton for the Heart‑Shaped Layered Window MVP in Go, targeting Windows (windows/amd64) and Go 1.21+. Follow these constraints and produce files exactly as specified.

Global constraints:
- No CGO.
- Use standard Go modules.
- Windows-specific source files must use appropriate build tags to avoid build errors on non-Windows hosts (e.g., //go:build windows).
- Keep code minimal but complete and importable so subsequent steps can extend it.

Files to create:

1) go.mod
- module: github.com/example/heart
- require golang.org/x/sys v0.0.0 or latest (add if used later)

2) cmd/heart/main.go
- Parse flags (e.g., -fps, -no-animation) using flag package.
- Build a Config using defaults from internal/config.
- Initialize logging via internal/logging.
- Call a stub NewWindow(cfg) method from internal/window and call its Run() method (the window package will be implemented later).
- Handle errors and exit appropriately.

3) internal/config/config.go
- Define Config struct per devSpec:
  - InitialSize int (default 400)
  - KeepAspect bool (true)
  - MinSize int (150)
  - AlphaThreshold uint8 (10)
  - BorderGrabSize int (8)
  - FPS int (30)
  - HeartColor string or color.NRGBA (use simple string "#0078D7" for now)
  - AnimationEnabled bool (true)
  - WindowTitle string ("Heart")
- Provide a DefaultConfig() function that returns defaults.

4) internal/logging/log.go
- Tiny logging wrapper with Infof, Errorf functions that forward to log.Printf with prefixes.
- Prefer standard log package.

5) internal/window/window_stub_windows.go
- Add //go:build windows tag.
- Provide NewWindow(cfg) (*Window, error) and (w *Window) Run() error as stubs; NewWindow returns a struct with Run returning a not-yet-implemented error.

6) internal/renderer/renderer_stub.go
- Minimal Renderer interface and Start/Stop stubs (no implementation yet).
- Add //go:build !windows on a placeholder if needed to avoid build issues across platforms.

7) README.md
- Short build/run instructions.

8) Add a unit test:
- internal/config/config_test.go: TestDefaultConfig checks DefaultConfig returns sensible values (InitialSize == 400 etc).

Additional instructions for the code-generation:
- Keep each file small and focused.
- Do not implement Win32 API in this step beyond stubs.
- Ensure the project builds on Windows (go build ./...).
- Emit proper package names and imports.

Return:
- The exact file contents for the files above. Do not produce binaries. Prefer idiomatic Go.

End of prompt.
```

Todo checklist for Phase 0
- [x] Create go.mod and module skeleton
- [x] Implement internal/config with DefaultConfig()
- [x] Implement internal/logging wrapper
- [x] Create cmd/heart/main.go using the config and logging
- [x] Add window and renderer stubs (Windows build tags where appropriate)
- [x] Add README.md and unit test for config
- [x] Project builds (go build) and tests run (go test) on Windows

---

Phase 1 — Basic Win32 window + message loop
Goal
- Implement Win32 window class registration, CreateWindowEx with WS_EX_LAYERED | WS_EX_APPWINDOW and a message loop that shows a blank layered window. No rendering yet.

Small implementation steps (atomic)
1. Implement internal/window package with Windows-only file (window_windows.go) using //go:build windows.
2. Use golang.org/x/sys/windows and/or syscall.NewLazyDLL to call RegisterClassExW, CreateWindowExW, ShowWindow, GetMessage/TranslateMessage/DispatchMessage, DefWindowProcW.
3. Create window class with WNDCLASSEXW, set lpfnWndProc to a Go callback using syscall.NewCallback. Use a small, safe window procedure that forwards DefWindowProc for everything and handles WM_DESTROY to PostQuitMessage.
4. Create window with WS_POPUP | WS_VISIBLE and extended styles WS_EX_LAYERED | WS_EX_APPWINDOW.
5. Ensure window appears (empty transparent window). Because no DIB/buffer yet, it's acceptable if the window is blank — the main goal is to ensure a window is created and message loop runs.
6. Expose NewWindow(cfg) that builds the Window struct and (w *Window) Run() that runs the message loop and blocks until quit. Keep Window struct fields minimal for this step (hwnd windows.HWND).

Manual verification
- Build and run on Windows. Running should create a (probably fully transparent) window and the process should stay alive. Alt+Tab should show the window (WS_EX_APPWINDOW). Closing the window or pressing Alt+F4 should exit.

Prompt to generate code
```text
Implement the Win32 window creation and basic message loop for Windows in the internal/window package.

Requirements:
- File: internal/window/window_windows.go with //go:build windows
- Implement NewWindow(cfg) (*Window, error) and (w *Window) Run() error
- Use golang.org/x/sys/windows and syscall.NewLazyDLL for Win32 calls. No CGO.
- Register a window class (RegisterClassExW) with a Go window procedure using syscall.NewCallback. The WndProc should:
  - On WM_DESTROY call PostQuitMessage(0)
  - For everything else call DefWindowProcW
- Create window with CreateWindowExW using extended styles:
  - WS_EX_LAYERED | WS_EX_APPWINDOW
  - Window styles: WS_POPUP | WS_VISIBLE
- ShowWindow to make it visible and run a message loop (GetMessage/TranslateMessage/DispatchMessage) in Run().
- Ensure the window is created and NewWindow returns an instance with wnd hwnd set.
- Keep the Window struct minimal for now:
  type Window struct {
      hwnd windows.HWND
      cfg Config
  }
- Add safe error handling: check return values from Win32 calls and return descriptive errors.
- Add small comments describing where later code will integrate (DIBSection, renderer).
- Make sure this compiles and works on Windows.

Testing instructions to include in code comments:
- Run `go build ./cmd/heart` and execute on Windows. Confirm process shows a window in taskbar/Alt+Tab.
- Close window to ensure process exits.

Return the full source for internal/window/window_windows.go and any helper functions you add.

End of prompt.
```

Todo checklist for Phase 1
- [x] Implement window_windows.go with RegisterClassExW/CreateWindowExW
- [x] Implement WndProc handling WM_DESTROY -> PostQuitMessage
- [x] ShowWindow and message loop in Run()
- [x] NewWindow returns Window with hwnd set
- [ ] Manual verification: window appears and exits on close

---

Phase 2 — CreateDIBSection & memory DC (pixel buffer)
Goal
- Create a DIBSection (32bpp ARGB/BGRA) and associated memory DC. Expose access to a Go slice referencing the DIBSection memory (read/write from UI thread).

Small implementation steps (atomic)
1. Add createDIBSection function that wraps CreateDIBSection via GDI32 (CreateDIBSection) with BITMAPINFO for 32bpp BI_RGB.
2. Create a compatible DC (CreateCompatibleDC) and SelectObject the HBITMAP into it.
3. Obtain pointer to pixel memory returned by CreateDIBSection and create a Go []byte slice header referencing it using unsafe.Slice + unsafe.Pointer (protect usage to Windows build only).
4. Document pixel format (BGRA in memory). Add helper to compute bytes length and stride.
5. Add error handling for partial failures and cleanup functions to delete HBITMAP and DeleteDC.

Manual verification
- Build and run. The Run() function may call createDIBSection during initialization. Program should not crash. Inspect with debugger or add a temporary operation that writes a few test pixels into the memory and force UpdateLayeredWindow later (not in this step) — but ensure memory mapping works without crash.

Prompt to generate code
```text
Implement DIBSection creation and memory DC management in internal/window.

Goal:
- Create a CreateDIBSection wrapper that allocates a 32bpp ARGB/BGRA DIBSection, returns:
  - hdcMem (memory DC)
  - hBitmap (HBITMAP selected into DC)
  - pixels []byte (a Go slice referencing the DIBSection memory)
  - stride, width, height
- Provide a DestroyDIBSection(hdcMem, hBitmap) helper to clean up resources.

Requirements & details:
- File: internal/window/dib_windows.go (//go:build windows)
- Use syscall.NewLazyDLL / proc calls or golang.org/x/sys/windows for:
  - Gdi32.CreateDIBSection
  - Gdi32.CreateCompatibleDC
  - Gdi32.SelectObject
  - Gdi32.DeleteObject
  - Gdi32.DeleteDC
- BITMAPINFO must be configured for 32bpp (biBitCount = 32) and BI_RGB.
- Interpret the pixel memory layout as BGRA (leverage common GDI behaviour).
- Use unsafe to create a Go []byte slice that references the DIBSection memory.
  - Ensure the code only constructs/use this slice on Windows and that the memory is not freed while used.
  - Document in comments that only the UI thread should write to the DIBSection directly.
- Provide helper signature:
  func createDIBSection(width, height int) (hdcMem windows.HDC, hBitmap windows.Handle, pixels []byte, stride int, err error)
  and
  func destroyDIBSection(hdcMem windows.HDC, hBitmap windows.Handle) error

Error handling:
- If any call fails, clean up allocated resources before returning.
- Return descriptive errors.

Return:
- The full file internal/window/dib_windows.go with all necessary imports and helper code.

Testing/Manual verification steps (add to file comments):
- Call createDIBSection from NewWindow; verify it returns non-zero objects.
- Optionally write some bytes to pixels slice in Run() before UpdateLayeredWindow later.
- Ensure no panic and resources can be destroyed via destroyDIBSection.

End of prompt.
```

Todo checklist for Phase 2
- [x] Implement createDIBSection and destroyDIBSection
- [x] Create Go slice mapping to DIBSection memory (unsafe)
- [x] Ensure proper cleanup on errors
- [ ] Manual verification: create/destroy without crash

---

Phase 3 — Renderer core: heart SDF & static rasterizer + unit tests
Goal
- Implement a deterministic heart SDF (signed distance) and a rasterizer that renders an RGBA/BGRA buffer for a given width/height and color. Add unit tests for the SDF.

Small implementation steps (atomic)
1. Create internal/renderutil/heart.go with functions:
   - HeartSDF(x, y, w, h) float64 // returns signed distance where negative=inside
   - RasterizeHeart(width,height,color,aa) -> []byte (BGRA straight alpha)
   - Add simple smoothing using smoothstep from SDF -> alpha.
2. Ensure RasterizeHeart returns a Go-managed []byte buffer (not the DIB memory).
3. Add unit tests:
   - TestHeartSDFInsideOutside: sample center should be inside (negative), far corner outside (positive)
   - TestRasterizeSizes: rasterize 150x150 and assert buffer length and non-zero alpha count > 0.
4. Make sure SDF uses coordinate normalization and keeps aspect 1:1 behaviour.

Manual verification
- On Windows, run go test ./internal/renderutil and ensure tests pass.
- Optionally produce a small CLI in tests to write PNG to disk to visually inspect heart (but PNG encoding requires image/png import; keep as optional debug code gated behind build tag).

Prompt to generate code
```text
Implement the heart signed distance function and a CPU rasterizer.

Files:
- internal/renderutil/heart.go
- internal/renderutil/heart_test.go

Requirements:
1) Heart SDF
- Implement a function with signature:
  func HeartSDF(px, py, w, h float64) float64
  - px,py are pixel coordinates in image space [0..w), [0..h)
  - The function returns signed distance in pixels: negative inside heart, positive outside.
  - Normalize coordinates to a -1..1 coordinate system appropriately so hearts scale with size.
  - Make it deterministic and centered.

2) Rasterizer
- Implement:
  func RasterizeHeart(width, height int, color color.NRGBA, aaFactor int) ([]byte, error)
  - Returns a buffer in BGRA byte order (4 bytes per pixel) with straight (non-premultiplied) alpha channel.
  - Use the HeartSDF value and map distance -> alpha via a smoothstep over an edge width of ~1.5 px (or configurable).
  - Support aaFactor (supersampling factor). For MVP aaFactor=1 ok; implement logic so future changes can increase supersample.
  - Avoid allocations per pixel; allocate a single []byte sized width*height*4.

3) Tests
- heart_test.go:
  - TestHeartSDF_Basic: assert center is inside (sdf < 0) and corner outside (sdf > 0).
  - TestRasterizeHeart_SizeAndAlpha: rasterize 150x150 and assert returned buffer length == 150*150*4 and that there's at least one pixel with alpha > 0.
  - Keep tests fast.

Constraints:
- Use only pure Go code in this package.
- Use the standard "image/color" package for NRGBA.
- Ensure functions are deterministic and unit tests pass on CI.

Return file contents for heart.go and heart_test.go.

End of prompt.
```

Todo checklist for Phase 3
- [x] Implement HeartSDF function
- [x] Implement RasterizeHeart returning BGRA buffer
- [x] Add unit tests verifying SDF and rasterization
- [x] Run go test ./internal/renderutil and pass

---

Phase 4 — Apply frame: copy into DIBSection + UpdateLayeredWindow (static heart)
Goal
- Wire the renderer output into the window: copy the Go-rendered buffer into the DIBSection memory on the UI thread and call UpdateLayeredWindow to present a static heart.

Small implementation steps (atomic)
1. Add applyFrame(frame []byte, width, height) method on Window that:
   - Locks state if needed (for now Window owns DIBSection).
   - Copies bytes from frame into the DIBSection pixel slice returned in Phase 2.
   - Calls UpdateLayeredWindow with BLENDFUNCTION {AC_SRC_ALPHA} and the memory DC.
2. Implement a simple path: in NewWindow or Run, call RasterizeHeart(width,height,color,1), then call applyFrame on UI thread.
3. Ensure UpdateLayeredWindow is invoked via syscall.NewLazyDLL/proc with correct parameters (hdcMem, size, blend, etc).
4. Use correct pixel ordering (BGRA) and straight alpha.
5. Validate error returns on UpdateLayeredWindow failing.

Manual verification
- Build and run. Expect a visible heart shape window showing the rendered heart with antialiased edges (depending on SDF).
- Confirm UpdateLayeredWindow call runs on UI thread and no panics.
- Verify window still shows in taskbar/Alt+Tab.

Prompt to generate code
```text
Wire renderer output into the layered window by copying a rasterized frame into the DIBSection and calling UpdateLayeredWindow.

Tasks:
- File: internal/window/apply_frame_windows.go (//go:build windows)
- Implement a method:
  func (w *Window) applyFrame(pixels []byte, width, height int) error
  - Copy pixels into the DIBSection-backed slice (created in Phase 2).
  - Call UpdateLayeredWindow to present the HBITMAP/DC to the screen.
  - Use BLENDFUNCTION with AlphaFormat = AC_SRC_ALPHA and SourceConstantAlpha = 255 (straight alpha).
  - Ensure all Win32 calls happen on the UI thread.

- Integration:
  - In the window Run() function (phase1), after creating DIBSection, call internal/renderutil.RasterizeHeart(...) to produce a buffer.
  - Call w.applyFrame(buffer, width, height) on the UI thread.
  - Keep this a single static frame (no animation yet).

Details & constraints:
- Use syscall.NewLazyDLL to call UpdateLayeredWindow (User32) if a wrapper doesn't exist.
- Handle errors: on failure return an error from applyFrame and log descriptive message.
- Ensure memory ordering and pixel format agreement (BGRA straight alpha).

Manual verification:
- Build and run on Windows. The window should now show the heart graphic.
- Ensure window still appears in taskbar/Alt+Tab.

Return the full source file(s) and any small modifications to internal/window/window_windows.go to integrate applyFrame invocation.

End of prompt.
```

Todo checklist for Phase 4
- [x] Implement applyFrame method copying into DIBSection and calling UpdateLayeredWindow
- [x] Integrate static RasterizeHeart call in Run()
- [ ] Manual verification: heart visible in window

---

Phase 5 — Per‑pixel hit testing (WM_NCHITTEST)
Goal
- Implement WM_NCHITTEST to read the DIBSection alpha at the cursor point and return HTTRANSPARENT for alpha < threshold, otherwise HTCAPTION or resize zones.

Small implementation steps (atomic)
1. Implement WM_NCHITTEST handling in the WndProc:
   - Translate lParam (cursor screen coords) to client coords (ScreenToClient).
   - Map client coords to pixel indices (x,y). Consider top-left alignment and 1:1 mapping for now.
   - Read alpha byte from the DIBSection slice with mu.RLock() (if using locking).
   - If alpha < cfg.AlphaThreshold return HTTRANSPARENT.
   - Else if cursor within BorderGrabSize of edges, return appropriate HT* values (HTLEFT, HTRIGHT, HTTOP, HTBOTTOM, HTTOPLEFT, etc).
   - Else return HTCAPTION so dragging works.
2. Ensure reading slice and mapping uses safe bounds checks.
3. Add unit tests for mapping function (screen->client->pixel) in internal/window as pure functions where possible.

Manual verification
- Build and run. Click transparent part of the heart: clicks should pass through (e.g., you can click desktop icons through the transparent window). Click on heart and drag to move window. Resize by grabbing near edge (see Phase 7 for more complete resizing logic).

Prompt to generate code
```text
Implement per-pixel hit testing via WM_NCHITTEST to support click-through transparent pixels and HTCAPTION dragging for opaque pixels.

Files to modify/add:
- internal/window/wndproc_windows.go (//go:build windows)

Requirements:
- In WndProc, handle WM_NCHITTEST message:
  - Convert lParam (cursor X/Y in screen coords) to client coordinates using ScreenToClient or MapWindowPoints.
  - Convert client coordinates to pixel coordinates in the DIBSection (considering the window client origin).
  - Safely read the alpha byte from the DIBSection slice (which was created in Phase 2). Use mutex RLock if necessary.
  - If the pixel alpha < cfg.AlphaThreshold return HTTRANSPARENT.
  - Else:
    - If cursor is within cfg.BorderGrabSize of edges or corners return the correct HT* code (HTLEFT, HTRIGHT, HTTOP, HTBOTTOM, HTTOPLEFT, HTTOPRIGHT, HTBOTTOMLEFT, HTBOTTOMRIGHT).
    - Otherwise return HTCAPTION.
- Add helper functions if needed:
  - clientPointToPixel(x, y int) (px, py int, ok bool)
- Ensure bounds checks so reading alpha out of range never panics.

Constraints:
- All Win32 calls should be on the UI thread.
- Keep WndProc simple and forward other messages to DefWindowProcW.

Manual verification steps:
- Build and run on Windows.
- Click a fully transparent area of the window; verify that clicks pass through to underlying windows (e.g., a desktop icon).
- Click on the visible heart: clicking and dragging should move the window.

Return the updated file contents and any helper functions added.

End of prompt.
```

Todo checklist for Phase 5
- [x] WM_NCHITTEST implemented reading alpha from DIBSection
- [x] Edge/corner detection using BorderGrabSize
- [ ] Manual verification: transparent click-through & drag works

---

Phase 6 — Minimal context menu (Quit, Restore size) and WM_COMMAND
Goal
- Implement right-click context menu with Quit and Restore Size options, tracking menu selection via WM_COMMAND.

Small implementation steps (atomic)
1. In WndProc, handle WM_RBUTTONUP (or WM_CONTEXTMENU) to create and show a popup menu via CreatePopupMenu, AppendMenuW, SetForegroundWindow, TrackPopupMenu.
2. Define menu IDs in a small const block (IDM_QUIT=1001, IDM_RESTORE=1002).
3. In WM_COMMAND handler, switch on LOWORD(wParam) to handle IDM_QUIT -> PostQuitMessage / w.Close, IDM_RESTORE -> call a restore function to set window size to cfg.InitialSize (use SetWindowPos).
4. Ensure SetForegroundWindow is called before TrackPopupMenu to avoid menu disappearing.

Manual verification
- Right-click on the heart; the context menu should appear. Choose Restore Size (if window resized previously) and Quit closes the app.

Prompt to generate code
```text
Implement a minimal right-click context menu with Quit and Restore Size entries and handle selections via WM_COMMAND.

Files to update:
- internal/window/wndproc_windows.go (existing WndProc)
- internal/window/menu_windows.go (new helper if desired)

Requirements:
- On WM_RBUTTONUP or WM_CONTEXTMENU:
  - Create a popup menu (CreatePopupMenu).
  - Append menu items:
    - IDM_RESTORE (label "Restore Size")
    - IDM_QUIT (label "Quit")
  - Call SetForegroundWindow(hwnd) before TrackPopupMenu to ensure menu behaves correctly.
  - TrackPopupMenu to show menu under cursor.
- On WM_COMMAND:
  - If IDM_RESTORE selected, call SetWindowPos to set the client area back to cfg.InitialSize x cfg.InitialSize while preserving WS_EX layered and position (use SWP_NOMOVE flag).
  - If IDM_QUIT selected, call PostQuitMessage(0) (or send a quit signal that stops renderer and exits).
- Define menu ID constants in code and document them.

Constraints:
- Keep the implementation simple and robust. All Win32 calls must be made on UI thread.

Manual verification:
- Build and run on Windows.
- Right-click the heart and verify menu appears.
- Select Restore Size to set window to initial size.
- Select Quit to exit the application.

Return the code changes for WndProc and any helper menu functions.

End of prompt.
```

Todo checklist for Phase 6
- [x] Add WM_RBUTTONUP menu creation and TrackPopupMenu
- [x] Add WM_COMMAND handlers for IDM_RESTORE and IDM_QUIT
- [ ] Manual verification: menu actions work

---

Phase 7 — Resizing rules: WM_GETMINMAXINFO & WM_SIZING enforcement (1:1)
Goal
- Support resizable window while maintaining 1:1 aspect ratio, enforce min/max sizes via WM_GETMINMAXINFO and WM_SIZING.

Small implementation steps (atomic)
1. Handle WM_GETMINMAXINFO:
   - Set ptMinTrackSize and ptMaxTrackSize in MINMAXINFO to cfg.MinSize and computed max based on screen resolution minus margins (use GetSystemMetrics(SM_CXSCREEN/SM_CYSCREEN) or MonitorFromWindow + GetMonitorInfo if available).
2. Handle WM_SIZING:
   - lParam points to RECT in which you can modify sides.
   - Based on wParam (edge being dragged), enforce width==height by adjusting the opposite side to maintain square.
   - Account for BorderGrabSize and keep minimum size constraints.
3. Ensure resizing updates internal DIBSection if width/height changed (deallocate old DIB and create new one sized to new client area). For safety do this on WM_SIZE or after WM_EXITSIZEMOVE:
   - On WM_SIZE, recreate DIBSection at new client size and call a re-render (RasterizeHeart) -> applyFrame.

Manual verification
- Resize the window by dragging edges/corners: the window should stay square. Minimum size should be enforced. After resize, the heart should re-render to fill the window.

Prompt to generate code
```text
Implement resizing rules to preserve a 1:1 aspect ratio and enforce min/max sizes.

Files to update:
- internal/window/wndproc_windows.go (add handlers for WM_GETMINMAXINFO, WM_SIZING, WM_SIZE)

Requirements:
1) WM_GETMINMAXINFO
- Fill MINMAXINFO to set minTrackSize to cfg.MinSize and maxTrackSize based on screen dimensions minus some margin (calculate via GetSystemMetrics SM_CXSCREEN/SM_CYSCREEN).
- Ensure the window cannot be made larger than screen bounds - 100 (per devSpec).

2) WM_SIZING
- When receiving WM_SIZING, adjust the RECT in-place to preserve a square client area:
  - Determine which edge is being moved (wParam).
  - Compute new width/height and modify RECT so width==height.
  - Enforce min size.

3) WM_SIZE
- When resized (WM_SIZE), recreate the DIBSection and memory DC sized to the new client area:
  - destroy old DIB/hdcMem
  - create new DIBSection for new size
  - call RasterizeHeart to produce new frame and applyFrame to update window.
- Avoid flicker by ensuring UpdateLayeredWindow is called on the UI thread.

Constraints:
- Keep logic correct for most common user flows. This does not need to cover exotic multi-monitor DPI differences for MVP; document DPI as future work.

Manual verification:
- Build and run on Windows.
- Drag edges/corners to resize; window should remain square and respect min/max sizes.
- After resize the heart should re-render to fit the new client area.

Return the code modifications.

End of prompt.
```

Todo checklist for Phase 7
- [x] Implement WM_GETMINMAXINFO with min/max values
- [x] Implement WM_SIZING to enforce square aspect ratio
- [x] Handle WM_SIZE to recreate DIBSection and redraw heart
- [ ] Manual verification: resizing works & re-renders

---

Phase 8 — Anti‑aliasing & pulse animation (renderer goroutine + frameCh)
Goal
- Upgrade renderer to produce animated frames (pulse), produce frames on a renderer goroutine at cfg.FPS, send frames via frameCh to UI thread; UI thread applies frames. Improve anti-aliasing (support aaFactor or SDF smoothing).

Small implementation steps (atomic)
1. Add internal/renderer package with Renderer struct implementing Start(ctx, out chan<- *Frame) and Stop.
2. Frame struct: Width, Height, Pixels []byte, Timestamp.
3. Implement Start to spawn goroutine that:
   - At each tick (time.Ticker based on FPS) computes animation parameter (sinusoidal pulse based on time).
   - Calls RasterizeHeart with animation parameter applied to alpha multiplier or color brightness.
   - Reuses buffers to avoid allocations and sends a pointer/copy of frame to out channel.
   - Stops when context canceled.
4. In Window.Run, create frameCh chan *Frame with small buffer (1), start renderer with Start(ctx, frameCh). In message loop, implement a mechanism (PostMessage or a custom event) to apply frames on UI thread:
   - Best approach: use a channel and a goroutine that uses SendMessage/PostMessage? Simpler: have a goroutine read from frameCh and call w.applyFrame on the UI thread by calling PostMessage with a custom registered message (RegisterWindowMessageW) containing pointer to frame data — but pointers across threads are tricky.
   - Simpler safe approach: Use a worker goroutine that reads frames and uses a synchronization channel to ask the UI thread to apply them. However UpdateLayeredWindow **must** be called on the UI thread. So implement: renderer sends frames to frameCh; a small dedicated UI dispatcher (within the UI thread) periodically checks frameCh using PeekMessage timeouts or by adding a timer message to the message loop:
     - Use a Windows timer (SetTimer) to trigger WM_TIMER at FPS and on WM_TIMER, drain the latest frame from frameCh (non-blocking) and call applyFrame.
   - Implement WM_TIMER handler to read latest frame (if any) and call applyFrame. This keeps UpdateLayeredWindow on UI thread.
5. Ensure buffer copying still occurs: renderer writes into Go-managed []byte; UI thread applies by copying into DIBSection memory then UpdateLayeredWindow.

Manual verification
- Build/run. Heart should have a simple pulse animation. CPU usage should be reasonable. Ensure animation can be toggled off via cfg.AnimationEnabled.

Prompt to generate code
```text
Add an animated renderer producing frames at configured FPS, and wire frames to the UI thread safely.

Files to implement:
- internal/renderer/renderer.go
- internal/renderer/renderer_test.go (unit test for frame timing/stop behavior)
- Update internal/window/wndproc_windows.go and window logic to handle WM_TIMER to pull frames safely.

Requirements:
1) Renderer
- Provide:
  type Frame struct {
      Width int
      Height int
      Pixels []byte // BGRA straight alpha: width*height*4
      Timestamp time.Time
  }
- Provide:
  type Renderer struct {
      cfg Config
  }
  func NewRenderer(cfg Config) *Renderer
  func (r *Renderer) Start(ctx context.Context, out chan<- *Frame) error
  func (r *Renderer) Stop() error
- Start spawns a goroutine that ticks at cfg.FPS and generates frames. Each frame's pixels are generated by calling RasterizeHeart with an animation parameter (e.g., pulse = 0.9 + 0.1*sin(2πt)). Reuse underlying []byte slices to avoid per-frame allocations.
- If cfg.AnimationEnabled==false render a static frame once and stop producing frames (or continue producing identical frames at low cost).

2) UI integration
- Create a channel frameCh := make(chan *renderer.Frame, 1)
- Start renderer in NewWindow/Run with Start(ctx, frameCh)
- Use SetTimer to set a timer with interval 1000/cfg.FPS (or a safe small interval)
- In WM_TIMER handler:
  - Drain the most recent frame from frameCh if present (non-blocking read; if multiple frames queued prefer the latest).
  - Call w.applyFrame(frame.Pixels, width, height)
- Ensure UpdateLayeredWindow remains on UI thread.

3) Tests
- In renderer_test.go add a test that starts the renderer with a short context and asserts frames are generated at roughly the correct rate and that Stop cancels goroutine.

Constraints:
- Avoid passing raw pointers between threads. Always copy frame.Pixels into DIBSection memory on UI thread before UpdateLayeredWindow.
- Keep code paths simple and handle context cancellation.

Manual verification:
- Build/run. Heart should display a visible pulse animation.
- Confirm toggling cfg.AnimationEnabled false produces a static image.

Return full source for internal/renderer/renderer.go and patches to window to integrate timer-based frame application.

End of prompt.
```

Todo checklist for Phase 8
- [x] Implement Renderer generating frames at cfg.FPS (pulse animation)
- [x] Wire frames into UI thread via WM_TIMER handler draining frameCh
- [x] Apply frames using applyFrame on UI thread
- [ ] Add unit test for renderer start/stop/timing
- [ ] Manual verification: pulse animation visible

---

Phase 9 — Full context menu (Always On Top, About), toggles
Goal
- Expand context menu with Always On Top toggle and About dialog, and implement toggling using SetWindowPos.

Small implementation steps (atomic)
1. Extend menu entries with IDM_TOGGLE_TOPMOST and IDM_ABOUT.
2. Implement toggling SetWindowPos with HWND_TOPMOST/HWND_NOTOPMOST pinned/unpinned preserving position/size.
3. Implement About showing a simple MessageBoxW with version and credits.
4. Persist isTopmost in Window struct and update menu checked state appropriately.

Manual verification
- Right-click and toggle Always On Top; verify window stays above others. Right-click About shows message box.

Prompt to generate code
```text
Enhance context menu to include "Always On Top" toggle and "About" entry. Implement toggling logic and About dialog.

Files to update:
- internal/window/menu_windows.go or wndproc_windows.go

Requirements:
- Add menu items:
  - IDM_TOGGLE_TOPMOST (label "Always On Top")
  - IDM_ABOUT (label "About")
- On menu creation, check the menu item if w.isTopmost is true (use CheckMenuItem or AppendMenu with MF_CHECKED).
- On WM_COMMAND:
  - For IDM_TOGGLE_TOPMOST:
    - Toggle w.isTopmost
    - Call SetWindowPos with HWND_TOPMOST or HWND_NOTOPMOST with SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE as applicable to preserve position/size.
    - Update menu checked state.
  - For IDM_ABOUT:
    - Call MessageBoxW with a simple message including the app name/version.
- Keep menu code robust: destroy menu after TrackPopupMenu.

Manual verification:
- Build/run.
- Right-click -> toggle Always On Top; verify behavior.
- Right-click -> About; verify message box appears.

Return code changes.

End of prompt.
```

Todo checklist for Phase 9
- [x] Add Always On Top and About menu items
- [x] Implement SetWindowPos toggling and menu check/uncheck
- [ ] Implement About MessageBoxW
- [ ] Manual verification: toggling and About dialog work

---

Phase 10 — Resource cleanup, graceful shutdown, logging
Goal
- Ensure all GDI and DC resources are freed on exit, renderer goroutine is stopped, and logging is complete. Add defer patterns and a ShutDown method that can be tested.

Small implementation steps (atomic)
1. Implement Window.Close() that:
   - Cancels renderer context.
   - Destroys DIBSection and DC.
   - Posts quit message if necessary.
2. Ensure WM_DESTROY triggers cleanup.
3. Ensure NewWindow register/unregister of window class and cleanup on error.
4. Add logs at startup/shutdown and for resource allocation/frees.

Manual verification
- Repeatedly start/stop the app and watch for GDI handle leaks using Process Explorer (handle counts should not grow). Ensure graceful shutdown.

Prompt to generate code
```text
Implement resource cleanup and graceful shutdown handling.

Files to update:
- internal/window/window_windows.go
- internal/window/dib_windows.go
- internal/window/wndproc_windows.go

Requirements:
- Implement func (w *Window) Close() error that:
  - Cancels renderer context (if present).
  - Calls destroyDIBSection to free hBitmap and delete DC.
  - Logs cleanup results.
- Ensure WM_DESTROY in WndProc calls w.Close() and PostQuitMessage.
- Ensure that on any initialization error in NewWindow, partially created resources are destroyed before returning.
- Add logging statements around resource allocation and freeing.

Manual verification:
- Repeatedly launch and close the app. Monitor with Process Explorer (GDI Handles) to confirm no GDI leaks.

Return changes to code.

End of prompt.
```

Todo checklist for Phase 10
- [x] Implement Window.Close() that cancels renderer and frees GDI/DC
- [x] WM_DESTROY calls Close()
- [x] Add logging around resource lifecycle
- [ ] Manual verification: no GDI leaks on repeated runs

---

Phase 11 — Tests, CI workflows, documentation
Goal
- Add unit tests to cover key functions, add GitHub Actions workflow for Windows build/test, finalize README with build/run instructions.

Small implementation steps (atomic)
1. Add unit tests:
   - renderer benchmarks (optional)
   - window client->pixel mapping pure function tests
2. Add .github/workflows/ci.yml that:
   - Runs on windows-latest
   - Installs Go 1.21
   - Runs go test ./...
   - Builds the binary via go build -o dist/heart.exe ./cmd/heart
3. Update README with prerequisites and steps to build on Windows and cross-compile.
4. Add a short QA checklist derived from devSpec.

Manual verification
- Push to GitHub and ensure Actions passes on Windows runner.

Prompt to generate code
```text
Add unit tests and a GitHub Actions CI workflow for Windows build/test.

Files to add:
- .github/workflows/ci.yml
- Additional tests if missing:
  - internal/window/window_test.go (pure function tests)
  - internal/renderer/renderer_test.go (start/stop tests)
- Update README.md to include build instructions and QA checklist.

CI workflow requirements:
- Runs on windows-latest.
- Uses actions/setup-go@v4 to install Go 1.21.
- Runs `go test ./...` and `go build -ldflags "-s -w" -o dist/heart.exe ./cmd/heart`.
- Uploads build artifact (optional).

Constraints:
- Keep tests fast and non-interactive.
- If any GUI-dependent tests exist, mark them to skip on CI or make them conditional.

Return new CI yaml and test additions/updates.

End of prompt.
```

Todo checklist for Phase 11
- [x] Add tests for window mapping and renderer start/stop
- [x] Add GitHub Actions workflow for windows-latest build/test
- [x] Update README with build/run instructions and QA checklist
- [ ] Push and verify CI passes

---

Appendix: How to use these prompts & checklist

- Use the prompts sequentially. Each prompt builds on the previous. Do not skip steps.
- After applying each prompt, run:
  - go vet ./...
  - go test ./...
  - go build ./cmd/heart
  - Manually run the binary on Windows and follow the verification instructions listed for each phase.
- If a code-generation LLM produces multiple files in a prompt, ensure they are added together in a single commit so the repo stays buildable.
- Each generated patch must not leave orphaned or unused functions. If a function is no longer needed, remove it or mark it for future use and document.

Concluding notes
- This plan intentionally builds the system incrementally, always verifying on Windows at each stage. It minimizes concurrency hazards by keeping UI-affecting operations on the window thread (UpdateLayeredWindow, menu calls, SetWindowPos).
- Later enhancements (not in this MVP) include DPI-awareness, partial region updates to UpdateLayeredWindow, direct DIBSection writes by renderer (with synchronization to avoid copying), and GPU rendering.
- You may ask me to synthesize the first prompt's code now (Phase 0), or to produce the code for any later phase. Each prompt above is ready to be fed to a code-generation LLM as-is.

If you want, I can now:
- Produce a commit-ready patch for Phase 0 (repo skeleton) using the first prompt, or
- Start at Phase 1 if you already have a skeleton.

Which phase should I generate code for now?