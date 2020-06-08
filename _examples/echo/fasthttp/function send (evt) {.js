function connect(addr) {
  var ws = new WebSocket(addr);

  ws.onopen = () => {
    console.log("CONECTED");
    ws.send("Hello World");

    ws.close();

    connect(addr);
  };
}
