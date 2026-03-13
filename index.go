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
  html, body { height: 100%; background: #000; overflow: hidden; }
  #terminal { height: 100%; }
  #status {
    position: fixed; top: 0; right: 0;
    padding: 4px 10px; font: 12px monospace;
    color: #888; background: rgba(0,0,0,0.7); z-index: 10;
  }
  #status.connected { color: #4c4; }
  #status.error { color: #c44; }
</style>
</head>
<body>
<div id="status">connecting...</div>
<div id="terminal"></div>
<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/lib/xterm.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.10.0/lib/addon-fit.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-web-links@0.11.0/lib/addon-web-links.min.js"></script>
<script>
(function() {
  const status = document.getElementById('status');
  const term = new Terminal({
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

  const fitAddon = new FitAddon.FitAddon();
  term.loadAddon(fitAddon);
  term.loadAddon(new WebLinksAddon.WebLinksAddon());
  term.open(document.getElementById('terminal'));
  fitAddon.fit();

  function connect() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(proto + '//' + location.host + '/ws');
    ws.binaryType = 'arraybuffer';

    ws.onopen = function() {
      status.textContent = 'connected';
      status.className = 'connected';
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
      status.textContent = 'disconnected — click to reconnect';
      status.className = 'error';
      term.write('\r\n\x1b[31m[connection closed]\x1b[0m\r\n');
    };

    ws.onerror = function() {
      status.textContent = 'error';
      status.className = 'error';
    };

    // Terminal input -> WebSocket (binary)
    term.onData(function(data) {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data));
      }
    });

    // Handle resize
    function sendResize() {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'resize',
          data: { rows: term.rows, cols: term.cols }
        }));
      }
    }

    term.onResize(function() { sendResize(); });

    window.addEventListener('resize', function() {
      fitAddon.fit();
    });

    // Click to reconnect
    status.onclick = function() {
      if (ws.readyState === WebSocket.CLOSED) {
        term.reset();
        connect();
      }
    };
  }

  connect();
  term.focus();
})();
</script>
</body>
</html>`
