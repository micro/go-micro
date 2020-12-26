var wsUri;
var output;
var count;
var ws;

window.addEventListener("load", function(evt) {
  wsUri  = "ws://" + window.location.host + "/stream/stream"
  output = document.getElementById("output");
  count  = document.getElementById("count");

  var print = function(message) {
    var d       = document.createElement("div");
    d.innerHTML = message;
    output.appendChild(d);
  };

  var parseCount = function(evt) {
    return JSON.parse(evt.data).count
  };

  var newSocket = function() {
    ws           = new WebSocket(wsUri);
    ws.onopen = function(evt) {
      print('<span style="color: green;">Connection Open</span>');
    }
    ws.onclose = function(evt) {
      print('<span style="color: red;">Connection Closed</span>');
      ws = null;
    }
    ws.onmessage = function(evt) {
      print('<span style="color: blue;">Update: </span>' + parseCount(evt));
    }
    ws.onerror = function(evt) {
      print('<span style="color: red;">Error: </span>' + parseCount(evt));
    }
  };

  newSocket()

  document.getElementById("send").onclick = function(evt) {
    if (!ws) {
      return false
    }

    var msg = { count: parseInt(count.value) }

    req = JSON.stringify(msg)
    print('<span style="color: blue;">Sent request: </span>' + req);
    ws.send(JSON.stringify(msg));

    return false;
  };

  document.getElementById("cancel").onclick = function(evt) {
    if (!ws) {
      return false;
    }
    ws.close();
    print('<span style="color: red;">Request Canceled</span>');
    return false;
  };

  document.getElementById("open").onclick = function(evt) {
    if (!ws) {
      newSocket()
    }
    return false;
  };
})
