package main

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>webnetd</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/css/xterm.min.css">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  html, body { height: 100%; background: #1a1a2e; overflow: hidden; font-family: sans-serif; }

  #app { display: none; flex-direction: column; height: 100%; }
  #app.active { display: flex; }

  #status {
    padding: 4px 10px; font: 12px monospace;
    color: #888; background: #0d0d1a; text-align: right;
  }
  #status.connected { color: #4c4; }
  #status.error { color: #c44; cursor: pointer; }

  #terminal { flex: 1; overflow: hidden; }

  /* Bottom input bar */
  #input-bar {
    display: flex; align-items: center; gap: 8px;
    padding: 8px 12px; background: #16213e;
    border-top: 1px solid #0f3460;
  }
  #cmd-input {
    flex: 1; padding: 10px 14px;
    font: 14px 'Menlo', 'Monaco', 'Courier New', monospace;
    background: #1a1a2e; color: #e0e0e0;
    border: 1px solid #0f3460; border-radius: 6px;
    outline: none;
  }
  #cmd-input:focus { border-color: #4c4; }
  #cmd-input::placeholder { color: #555; }
  #send-btn, #upload-btn {
    padding: 10px 14px; background: #0f3460; color: #ccc;
    border: none; border-radius: 6px; cursor: pointer; font-size: 14px;
    display: flex; align-items: center; gap: 4px;
  }
  #send-btn:hover, #upload-btn:hover { background: #1a4a8a; color: #fff; }
  #upload-btn { padding: 10px; }
  #upload-btn svg { width: 18px; height: 18px; fill: currentColor; }

  /* Login screen */
  #login-screen {
    display: none; height: 100%;
    justify-content: center; align-items: center;
  }
  #login-box {
    background: #16213e; border: 1px solid #0f3460;
    border-radius: 8px; padding: 40px; text-align: center;
    min-width: 320px;
  }
  #login-box h1 { color: #e0e0e0; margin-bottom: 8px; font-size: 22px; }
  #login-box p { color: #888; margin-bottom: 24px; font-size: 14px; }
  #login-box input {
    width: 100%; padding: 12px; font-size: 24px; text-align: center;
    letter-spacing: 8px; background: #1a1a2e; border: 1px solid #0f3460;
    border-radius: 4px; color: #e0e0e0; outline: none;
  }
  #login-box input:focus { border-color: #4c4; }
  #login-box button {
    margin-top: 16px; width: 100%; padding: 10px;
    background: #0f3460; color: #e0e0e0; border: none;
    border-radius: 4px; font-size: 16px; cursor: pointer;
  }
  #login-box button:hover { background: #1a4a8a; }
  #login-error { color: #c44; margin-top: 12px; font-size: 14px; display: none; }

  /* Upload overlay */
  #upload-overlay {
    display: none; position: fixed; inset: 0; z-index: 20;
    background: rgba(0,0,0,0.7); justify-content: center; align-items: center;
  }
  #upload-overlay.active { display: flex; }
  #upload-box {
    background: #16213e; border: 1px solid #0f3460;
    border-radius: 8px; padding: 30px; text-align: center;
    min-width: 360px; color: #e0e0e0;
  }
  #upload-box h2 { margin-bottom: 16px; font-size: 18px; }
  #drop-zone {
    border: 2px dashed #0f3460; border-radius: 8px; padding: 40px 20px;
    margin-bottom: 16px; cursor: pointer; transition: border-color 0.2s;
  }
  #drop-zone.dragover { border-color: #4c4; }
  #drop-zone p { color: #888; font-size: 14px; }
  #upload-status { font-size: 13px; margin-top: 8px; }
  #upload-status.ok { color: #4c4; }
  #upload-status.err { color: #c44; }
  #upload-close {
    margin-top: 16px; padding: 8px 24px;
    background: #0f3460; color: #e0e0e0; border: none;
    border-radius: 4px; cursor: pointer;
  }
</style>
</head>
<body>

<div id="login-screen">
  <div id="login-box">
    <h1>webnetd</h1>
    <p>Enter the PIN displayed on the server</p>
    <form id="login-form">
      <input type="text" id="pin-input" maxlength="6" pattern="[0-9]{6}"
             placeholder="000000" autocomplete="off" inputmode="numeric">
      <button type="submit">Connect</button>
    </form>
    <div id="login-error"></div>
  </div>
</div>

<div id="app">
  <div id="status">connecting...</div>
  <div id="terminal"></div>
  <div id="input-bar">
    <button id="upload-btn" title="Upload file">
      <svg viewBox="0 0 24 24"><path d="M9 16h6v-6h4l-7-7-7 7h4zm-4 2h14v2H5z"/></svg>
    </button>
    <input type="text" id="cmd-input" placeholder="Type a command..." autocomplete="off" spellcheck="false">
    <button id="send-btn">Send</button>
  </div>
</div>

<div id="upload-overlay">
  <div id="upload-box">
    <h2>Upload File</h2>
    <div id="drop-zone">
      <p>Drop a file here or click to select</p>
      <input type="file" id="file-input" style="display:none">
    </div>
    <div id="upload-status"></div>
    <button id="upload-close">Close</button>
  </div>
</div>

<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/lib/xterm.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.10.0/lib/addon-fit.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-web-links@0.11.0/lib/addon-web-links.min.js"></script>
<script>
(function() {
  var AUTH_ENABLED = {{AUTH_ENABLED}};
  var authToken = '';
  var ws = null;

  var loginScreen = document.getElementById('login-screen');
  var loginForm = document.getElementById('login-form');
  var pinInput = document.getElementById('pin-input');
  var loginError = document.getElementById('login-error');
  var app = document.getElementById('app');
  var statusEl = document.getElementById('status');
  var terminalEl = document.getElementById('terminal');
  var cmdInput = document.getElementById('cmd-input');
  var sendBtn = document.getElementById('send-btn');

  // Upload elements
  var uploadBtn = document.getElementById('upload-btn');
  var uploadOverlay = document.getElementById('upload-overlay');
  var uploadClose = document.getElementById('upload-close');
  var dropZone = document.getElementById('drop-zone');
  var fileInput = document.getElementById('file-input');
  var uploadStatusEl = document.getElementById('upload-status');

  var term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1a1a2e',
      foreground: '#e0e0e0',
      cursor: '#e0e0e0',
      selectionBackground: '#44475a',
    },
  });

  var fitAddon = new FitAddon.FitAddon();
  term.loadAddon(fitAddon);
  term.loadAddon(new WebLinksAddon.WebLinksAddon());

  // --- Command input bar ---
  function sendCommand() {
    var cmd = cmdInput.value;
    if (cmd === '' || !ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(new TextEncoder().encode(cmd + '\n'));
    cmdInput.value = '';
    cmdInput.focus();
  }

  sendBtn.onclick = sendCommand;
  cmdInput.addEventListener('keydown', function(e) {
    if (e.key === 'Enter') {
      e.preventDefault();
      sendCommand();
    }
  });

  // --- Auth flow ---
  if (AUTH_ENABLED) {
    loginScreen.style.display = 'flex';
    pinInput.focus();

    loginForm.onsubmit = function(e) {
      e.preventDefault();
      loginError.style.display = 'none';
      var pin = pinInput.value.trim();
      if (pin.length !== 6) {
        loginError.textContent = 'PIN must be 6 digits';
        loginError.style.display = 'block';
        return;
      }

      var body = new URLSearchParams();
      body.append('pin', pin);

      fetch('/login', { method: 'POST', body: body })
        .then(function(res) {
          if (!res.ok) throw new Error('Invalid PIN');
          return res.json();
        })
        .then(function(data) {
          authToken = data.token;
          loginScreen.style.display = 'none';
          startTerminal();
        })
        .catch(function(err) {
          loginError.textContent = err.message;
          loginError.style.display = 'block';
          pinInput.value = '';
          pinInput.focus();
        });
    };
  } else {
    startTerminal();
  }

  function startTerminal() {
    app.classList.add('active');
    term.open(terminalEl);
    fitAddon.fit();
    connect();
    cmdInput.focus();
  }

  function connect() {
    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    var wsUrl = proto + '//' + location.host + '/ws';
    if (authToken) {
      wsUrl += '?token=' + encodeURIComponent(authToken);
    }
    ws = new WebSocket(wsUrl);
    ws.binaryType = 'arraybuffer';

    ws.onopen = function() {
      statusEl.textContent = 'connected';
      statusEl.className = 'connected';
      sendResize();
    };

    ws.onmessage = function(ev) {
      if (ev.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(ev.data));
      } else {
        term.write(ev.data);
      }
    };

    ws.onclose = function() {
      statusEl.textContent = 'disconnected \u2014 click to reconnect';
      statusEl.className = 'error';
      term.write('\r\n\x1b[31m[connection closed]\x1b[0m\r\n');
    };

    ws.onerror = function() {
      statusEl.textContent = 'error';
      statusEl.className = 'error';
    };

    // Direct xterm input also still works (clicking on terminal)
    term.onData(function(data) {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data));
      }
    });

    function sendResize() {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'resize',
          data: { rows: term.rows, cols: term.cols }
        }));
      }
    }

    term.onResize(function() { sendResize(); });
    window.addEventListener('resize', function() { fitAddon.fit(); });

    statusEl.onclick = function() {
      if (ws && ws.readyState === WebSocket.CLOSED) {
        term.reset();
        connect();
      }
    };
  }

  // --- File upload ---
  uploadBtn.onclick = function() {
    uploadOverlay.classList.add('active');
    uploadStatusEl.textContent = '';
    uploadStatusEl.className = '';
  };

  uploadClose.onclick = function() {
    uploadOverlay.classList.remove('active');
    cmdInput.focus();
  };

  uploadOverlay.onclick = function(e) {
    if (e.target === uploadOverlay) {
      uploadOverlay.classList.remove('active');
      cmdInput.focus();
    }
  };

  dropZone.onclick = function() { fileInput.click(); };

  dropZone.ondragover = function(e) {
    e.preventDefault();
    dropZone.classList.add('dragover');
  };
  dropZone.ondragleave = function() {
    dropZone.classList.remove('dragover');
  };
  dropZone.ondrop = function(e) {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    if (e.dataTransfer.files.length > 0) {
      doUpload(e.dataTransfer.files[0]);
    }
  };

  fileInput.onchange = function() {
    if (fileInput.files.length > 0) {
      doUpload(fileInput.files[0]);
      fileInput.value = '';
    }
  };

  function doUpload(file) {
    uploadStatusEl.textContent = 'Uploading ' + file.name + '...';
    uploadStatusEl.className = '';

    var form = new FormData();
    form.append('file', file);

    var url = '/upload';
    if (authToken) {
      url += '?token=' + encodeURIComponent(authToken);
    }

    fetch(url, { method: 'POST', body: form })
      .then(function(res) {
        if (!res.ok) throw new Error('Upload failed: ' + res.statusText);
        return res.json();
      })
      .then(function(data) {
        uploadStatusEl.textContent = 'Uploaded ' + data.name + ' (' + formatSize(data.size) + ') to ' + data.path;
        uploadStatusEl.className = 'ok';
      })
      .catch(function(err) {
        uploadStatusEl.textContent = err.message;
        uploadStatusEl.className = 'err';
      });
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }
})();
</script>
</body>
</html>`
