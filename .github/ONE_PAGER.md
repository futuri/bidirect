# One‑Pager: Ventana nativa en forma de corazón (Go + Win32 Layered Window)

Resumen ejecutivo
- Objetivo: entregar una base robusta y minimalista en Go que cree una ventana nativa de Windows con forma de corazón usando per‑pixel alpha (UpdateLayeredWindow). Ventana sin bordes, con bordes suaves y transparencia por píxel, animación de pulso por defecto, movible/usuario‑redimensionable (manteniendo proporción), menú contextual (click derecho) y comportamiento natural de entrada (pixeles transparentes son click‑through). Esta base servirá como scaffolding para integrar, en fases posteriores, streaming de vídeo o frames remotos dentro de la forma.

Problema que resuelve
- Necesitamos una ventana con forma arbitraria y renderizado por‑píxel (bordes suavizados y alfa) que actúe como contenedor de contenido dinámico (video/frames) y que se comporte visualmente e interactiva como la forma que ves. Debe ser implementada en Go sin CGO y con código simple y mantenible.

Audiencia
- Equipos de producto y liderazgo de ingeniería interesados en:
  - Demostraciones y prototipos de UI nativa en Windows usando Go.
  - Un contenedor personalizado para mostrar contenido remoto (video/imágenes) con forma arbitraria.
  - Un scaffolding técnico para validar performance y UX antes de añadir streaming real.

Cliente ideal
- Equipo de desarrollo que:
  - Usa Go (acepta dependencias pequeñas, p.ej. golang.org/x/sys/windows).
  - Necesita una ventana con forma arbitraria y contenido dinámico.
  - Quiere evitar CGO y minimizar dependencias externas.
  - Valora simplicidad, capacidad de iteración rápida y apariencia pulida (bordes suaves, animaciones).

Plataforma / Tech Stack propuesta
- Lenguaje: Go (sin CGO).
- Dependencias: golang.org/x/sys/windows + llamadas directas a DLLs vía syscall.NewLazyDLL para User32/Gdi32.
- Win32 APIs clave:
  - WS_EX_LAYERED (estilo extendido).
  - UpdateLayeredWindow (render por‑pixel, blending).
  - CreateDIBSection / CreateCompatibleDC / SelectObject / DeleteObject (gestión de bitmap DIB 32bpp ARGB).
  - GetCursorPos / ScreenToClient / WM_NCHITTEST, WM_SIZING, WM_COMMAND, WM_DESTROY, etc.
  - CreatePopupMenu / AppendMenu / TrackPopupMenu / SetForegroundWindow (menú contextual).
  - SetWindowPos (toggle Always on Top).
- No CGO. No librerías GUI de alto nivel (por diseño del MVP).

Comportamiento y UX requeridos
- Visual:
  - Ventana rectangular a nivel OS pero visualmente con forma de corazón (per‑pixel alpha).
  - Relleno: color azul “Windows” (configurable en código).
  - Bordes suaves (antialiasing).
  - Animación: pulso ligero de alpha/tono por defecto (configurable).
- Interacción:
  - Movible/arrastrable: el usuario puede arrastrar la ventana arrastrando la forma del corazón.
  - Redimensionable: sí (no hay botón maximizar). Redimensionado mantiene proporción (1:1) por defecto; este comportamiento es configurable en el código.
  - Click‑through en píxeles transparentes: los píxeles con alfa por debajo de umbral (ej. <10) dejan pasar eventos a ventanas debajo.
  - Aparece en la barra de tareas y Alt+Tab (ventana normal).
  - Cierre con Alt+F4 (comportamiento nativo). También menú contextual → Salir.
- Menú contextual (clic derecho sobre la forma):
  - “Salir” (Quit).
  - “Restaurar tamaño” (reset a 400x400).
  - “Siempre encima” (toggle Always On Top).
  - “Acerca de” (información corta).
- Tamaño / apariencia inicial:
  - Tamaño inicial recomendado: 400x400 px.
  - Mantener relación 1:1 cuando se redimensione (configurable).
- Política de eventos:
  - WM_NCHITTEST implementado para devolver HTTRANSPARENT cuando pixel debajo del cursor es transparente; HTCAPTION/HTLEFT/HTRIGHT/etc si en borde/handle para soportar drag/resize.
  - WM_SIZING o ajuste en WM_GETMINMAXINFO para forzar proporción.

Arquitectura y flujo de datos (alto nivel)
1. Inicialización:
   - Crear ventana Win32 con WS_EX_LAYERED.
   - Crear DIBSection 32bpp (BGRA/ARGB) para uso como buffer de frontend.
   - Preparar estructura de configuración (tamanho inicial, color, threshold, fps, mantenerProporción).
2. Render loop (ticker, p.ej. 30–60 FPS si animación activa):
   - Software renderer en memoria: dibuja corazón en buffer ARGB (antialiasing por supersampling o función implícita con soft‑edge).
   - Llenar fondo con alpha=0 (transparente) y pintar el corazón con alfa y color.
   - Llamar a UpdateLayeredWindow con el HBITMAP/DIB para subir la imagen y blend.
3. Input handling:
   - En WM_NCHITTEST: convertir punto de cursor a coordenadas del bitmap y leer componente alpha del pixel correspondiente; si alpha < threshold => HTTRANSPARENT.
   - Para dragg/resize: devolver HTCAPTION o códigos de borde según proximidad a edges (p.ej. 4–8 px).
   - WM_RBUTTONUP -> crear/mostrar menú contextual.
   - WM_COMMAND -> manejar acciones de menú.
4. Resizing & aspect ratio:
   - Al recibir WM_SIZING ajustar RECT para mantener proporción 1:1.
   - Al restaurar tamaño, reconstruir DIBSection si cambió tamaño de cliente.
5. Always on Top:
   - Toggle con SetWindowPos(HWND_TOPMOST / HWND_NOTOPMOST).
6. Cierre / limpieza:
   - On WM_DESTROY -> DeleteObject/DC cleanup -> PostQuitMessage.

Detalles técnicos / decisiones claves
- Renderizado (anti‑aliased):
  - Opción A (recomendada MVP): CPU rasterizer con supersampling 2× o algoritmo de distancia‑filtrado para suavizar bordes. Permite control total sin añadir dependencias.
  - Opción B (alternativa): usar GDI+ para dibujar vectorial antialiased (recomendable si queremos paths complejos y menos código de rasterización, pero implica inicializar Gdiplus).
- Hit‑testing por‑pixel:
  - Mantener buffer ARGB en memoria, leer alpha para la coordenada del cursor. Umbral configurable (ej. 10/255).
- Gestión de DIB:
  - CrearDIBSection devuelve puntero a buffer; mantener referencia y actualizar píxeles cada frame; llamar UpdateLayeredWindow con PVOID y BLENDFUNCTION (AC_SRC_ALPHA).
- Performance:
  - Target inicial: 30 FPS para la animación de pulso. Con renderer optimizado en Go puro y UpdateLayeredWindow solo cuando hay cambio visual.
  - Para video real/streaming futuro, habrá que optimizar la deserialización y posiblemente usar WebSocket/TCP y codecs (Ffmpeg, hardware) fuera del MVP.
- Configuraciones en código (exportadas en top del paquete):
  - initialSize = 400
  - keepAspect = true (configurable)
  - heartColor = #0078D7 (Windows blue) — configurable
  - fps = 30
  - alphaThresholdHitTest = 10
  - animationEnabled = true (toggle)
  - borderGrabSize = 8 (px para detectar resize)
- Dependencias exactas propuestas:
  - golang.org/x/sys/windows
  - stdlib (image/color, time, sync, syscall, unsafe, etc.)

Non‑goals / límites del MVP
- No implementaremos en esta fase la ingestión de frames remotos (HTTP/WebSocket/RTSP). El diseño permite añadirlo después.
- No se implementarán codecs multimedia ni aceleración GPU en la primera entrega.
- No se implementará una suite completa de UI (solo la ventana, menú simple y demo de animación).

Plan de entrega / hitos (para conversación con ingeniería)
1. Hito 0 — Especificación (esta One‑Pager) — OK.
2. Hito 1 — Implementación básica:
   - Crear ventana WS_EX_LAYERED, DIBSection, UpdateLayeredWindow working.
   - Render de un corazón estático en ARGB y subirlo.
   - Per‑pixel hit test y devolver HTTRANSPARENT.
   - Menú contextual minimal (Salir, Restaurar tamaño).
   - Movible (arrastre) y resizable con mantenimiento de aspect ratio (WM_SIZING).
   - Aparece en taskbar / Alt+Tab.
   - Tests manuales básicos en Windows 10/11.
   - Tiempo estimado: 3–5 days de un ingeniero Go con experiencia Win32.
3. Hito 2 — Pulido:
   - Anti‑aliasing mejor (supersample o distancia).
   - Animation pulso a 30 FPS y opción de desactivar.
   - Toggle Always On Top y About.
   - Manejo correcto de recurso DC/HBITMAP cleanup.
   - Tiempo estimado: +2–3 days.
4. Hito 3 — Hardening y tests:
   - Pruebas de rendimiento (actualización continua), memory checks, edgecases de redimensionamiento.
   - Documentación y ejemplo de integración (cómo enviar frames en el futuro).
   - Tiempo estimado: +2 days.
Estimación total MVP: 1–2 semanas (1 ingeniero). Rango depende de experiencia previa con Win32 desde Go y si se elige GDI+ para render.

Riesgos y mitigaciones
- UpdateLayeredWindow coste CPU si se actualiza a alta frecuencia:
  - Mitigación: limitar fps, actualizar solo las regiones que cambian, medir CPU en pruebas.
- Implementación de antialias compleja:
  - Mitigación: comenzar con supersampling 2× o usar GDI+ si aceptable.
- Hit testing y sincronización entre buffer y ventana:
  - Mitigación: encapsular buffer en estructura con mutex y asegurar que WM_NCHITTEST sólo lee buffer inmutable o bloqueado.
- Gestión de recursos GDI (fugas):
  - Mitigación: diseño cuidadoso de lifecycle, tests y defer DeleteObject/ReleaseDC.

Requerimientos para arrancar / entregables
- Decisión técnica final:
  - Confirmar uso de golang.org/x/sys/windows + syscall.NewLazyDLL (recomendado) — (aprobado por product).
- Entregables MVP:
  - Repo Go con main.go y paquete windows_layered:
    - main: inicialización y flags (configurables).
    - renderer: rasterizador heart -> buffer ARGB.
    - window: Win32 message loop, hit test, menu handlers.
    - docs: README con instrucciones de build (go build), run y toggles config.
  - Build y prueba en Windows 10/11 (x64).
- Permisos:
  - Ninguno especial (app de usuario, ejecutable normal).

Siguientes pasos recomendados para product & engineering
1. Aprobación del scope MVP y estimación de tiempo.
2. Asignación de 1 ingeniero Go con familiaridad Win32 o apoyo de SME Win32.
3. Kickoff técnico: definir exacto color Windows blue (hex), confirmación de FPS y si usar GDI+ para rendering antialiased.
4. Entrega del Hito 1 (ci cerrable) y demo interno.
5. Iteración para añadir ingestión de frames (HTTP POST o WebSocket) según prioridades.

Anexo — APIs y mensajes clave (rápida referencia)
- Win32: CreateWindowEx, DefWindowProc, RegisterClassEx, ShowWindow, UpdateLayeredWindow, CreateDIBSection, CreateCompatibleDC, SelectObject, DeleteObject, ReleaseDC, GetCursorPos, ScreenToClient, SetWindowPos, CreatePopupMenu, AppendMenu, TrackPopupMenu, PostQuitMessage.
- Mensajes: WM_NCHITTEST, WM_MOUSEMOVE, WM_LBUTTONDOWN, WM_RBUTTONUP, WM_SIZING, WM_GETMINMAXINFO, WM_COMMAND, WM_DESTROY.
- Flags: WS_EX_LAYERED, WS_EX_APPWINDOW (para aparecer en taskbar), WS_POPUP | WS_VISIBLE (sin bordes), SWP_NOMOVE, SWP_NOSIZE.

Contacto / responsables
- (Indicar quién en tu equipo lidera el prototipo, o pedir que asignen un ingeniero). Sugerencia: asignar 1 owner técnico y 1 reviewer Win32.

Fin — si quieres, puedo ahora:
- Generar un esqueleto de código en Go (archivo principal) que implementa la base (ventana layered, UpdateLayeredWindow, renderer básico del corazón con pulso y hit testing por píxel), listo para compilar en Windows, o
- Preparar un ticket técnico detallado con checklist por hito para gestión/agile.

¿Qué prefieres como siguiente entregable?