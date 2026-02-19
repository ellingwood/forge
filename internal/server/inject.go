// Package server provides the development HTTP server with live reload support
// for the Forge static site generator.
package server

import (
	"bytes"
	"fmt"
)

// liveReloadScript is the JavaScript injected into HTML pages to enable
// automatic browser reloading when content changes.
const liveReloadScript = `<script>
(function() {
  var url = "ws://" + location.hostname + ":%d/__forge/ws";
  var ws;
  function connect() {
    ws = new WebSocket(url);
    ws.onmessage = function(e) {
      if (e.data === "reload") {
        location.reload();
      }
    };
    ws.onclose = function() {
      setTimeout(connect, 1000);
    };
  }
  connect();
})();
</script>`

// InjectLiveReload inserts the live reload WebSocket script into the HTML
// document. If a </body> tag is found, the script is inserted immediately
// before it. Otherwise the script is appended to the end of the document.
func InjectLiveReload(html []byte, port int) []byte {
	script := fmt.Appendf(nil, liveReloadScript, port)

	idx := bytes.LastIndex(html, []byte("</body>"))
	if idx == -1 {
		// No </body> tag found; append script at the end.
		return append(html, script...)
	}

	// Insert script before </body>.
	result := make([]byte, 0, len(html)+len(script))
	result = append(result, html[:idx]...)
	result = append(result, script...)
	result = append(result, html[idx:]...)
	return result
}
