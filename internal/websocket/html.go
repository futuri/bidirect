package websocket

const htmlPage = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>BiDirect Streamer</title>
  <style>
    body { font-family: Arial; margin: 20px; background: #1e1e1e; color: #fff; }
    h1 { color: #0078D7; }
    video { border: 2px solid #0078D7; width: 100%; max-width: 640px; margin: 20px 0; }
    button { 
      padding: 10px 20px; 
      font-size: 16px; 
      cursor: pointer; 
      background: #0078D7; 
      color: white;
      border: none;
      border-radius: 4px;
      margin: 5px;
    }
    button:hover { background: #005a9e; }
    button:disabled { background: #666; cursor: not-allowed; }
    #status { 
      padding: 10px; 
      margin: 20px 0; 
      border-radius: 4px;
      background: #2a2a2a;
      max-width: 640px;
    }
    #status.connected { border-left: 4px solid #00ff00; }
    #status.disconnected { border-left: 4px solid #ff0000; }
    .stats { font-size: 12px; color: #aaa; margin-top: 10px; }
  </style>
</head>
<body>
  <h1>🎥 BiDirect WebSocket Streamer</h1>
  
  <div id="status" class="disconnected">
    Estado: <span id="statusText">Desconectado</span>
    <div class="stats">
      Frames: <span id="frameCount">0</span> | 
      Bytes: <span id="byteCount">0</span> |
      FPS: <span id="fpsCount">0</span>
    </div>
  </div>

  <video id="video" autoplay playsinline></video>

  <div>
    <button id="startBtn" onclick="startStream()">▶ Iniciar</button>
    <button id="stopBtn" onclick="stopStream()" disabled>⏹ Detener</button>
    <select id="fpsSelect" onchange="changeFPS()">
      <option value="15">15 FPS</option>
      <option value="24">24 FPS</option>
      <option value="30" selected>30 FPS</option>
      <option value="60">60 FPS</option>
    </select>
  </div>

  <script src="/client.js"></script>
</body>
</html>`
