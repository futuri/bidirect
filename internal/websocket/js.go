package websocket

const jsClient = `
let ws = null;
let video = document.getElementById('video');
let canvas = document.createElement('canvas');
let ctx = canvas.getContext('2d');
let streaming = false;
let frameCount = 0;
let byteCount = 0;
let fps = 30;
let lastTime = Date.now();
let fpsCounter = 0;

const statusEl = document.getElementById('status');
const statusText = document.getElementById('statusText');
const frameCountEl = document.getElementById('frameCount');
const byteCountEl = document.getElementById('byteCount');
const fpsCountEl = document.getElementById('fpsCount');

function updateStatus(connected) {
  if (connected) {
    statusEl.className = 'connected';
    statusText.textContent = '✓ Conectado';
  } else {
    statusEl.className = 'disconnected';
    statusText.textContent = '✗ Desconectado';
  }
}

function connectWebSocket() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(protocol + '//' + window.location.host + '/stream');
  ws.binaryType = 'arraybuffer';
  
  ws.onopen = () => {
    console.log('WebSocket conectado');
    updateStatus(true);
  };
  
  ws.onclose = () => {
    console.log('WebSocket cerrado');
    updateStatus(false);
    streaming = false;
    document.getElementById('startBtn').disabled = false;
    document.getElementById('stopBtn').disabled = true;
  };
  
  ws.onerror = (err) => {
    console.error('WebSocket error:', err);
    updateStatus(false);
  };
}

async function startStream() {
  try {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      connectWebSocket();
      await new Promise(r => setTimeout(r, 500));
    }

    const stream = await navigator.mediaDevices.getUserMedia({ 
      video: { width: 640, height: 480 } 
    });
    
    video.srcObject = stream;
    video.onloadedmetadata = () => {
      canvas.width = video.videoWidth;
      canvas.height = video.videoHeight;
      console.log('Webcam:', canvas.width + 'x' + canvas.height);
      streaming = true;
      document.getElementById('startBtn').disabled = true;
      document.getElementById('stopBtn').disabled = false;
      frameCount = 0;
      byteCount = 0;
      lastTime = Date.now();
      streamLoop();
      fpsLoop();
    };
  } catch (err) {
    alert('Error accediendo webcam: ' + err.message);
  }
}

function stopStream() {
  streaming = false;
  if (video.srcObject) {
    video.srcObject.getTracks().forEach(t => t.stop());
  }
  document.getElementById('startBtn').disabled = false;
  document.getElementById('stopBtn').disabled = true;
}

function changeFPS() {
  fps = parseInt(document.getElementById('fpsSelect').value);
}

function captureFrame() {
  if (!streaming || !ws || ws.readyState !== WebSocket.OPEN) return;
  
  ctx.drawImage(video, 0, 0);
  canvas.toBlob((blob) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      blob.arrayBuffer().then(buffer => {
        // Protocolo: 4 bytes tamaño + datos
        const size = buffer.byteLength;
        const packet = new ArrayBuffer(4 + size);
        const view = new DataView(packet);
        view.setUint32(0, size, true); // little endian
        new Uint8Array(packet, 4).set(new Uint8Array(buffer));
        
        ws.send(packet);
        frameCount++;
        fpsCounter++;
        byteCount += packet.byteLength;
        frameCountEl.textContent = frameCount;
        byteCountEl.textContent = (byteCount / 1024 / 1024).toFixed(2) + ' MB';
      });
    }
  }, 'image/webp', 0.8);
}

function streamLoop() {
  if (!streaming) return;
  captureFrame();
  setTimeout(streamLoop, 1000 / fps);
}

function fpsLoop() {
  if (!streaming) return;
  const now = Date.now();
  const elapsed = now - lastTime;
  if (elapsed >= 1000) {
    fpsCountEl.textContent = fpsCounter;
    fpsCounter = 0;
    lastTime = now;
  }
  setTimeout(fpsLoop, 100);
}

connectWebSocket();
`
