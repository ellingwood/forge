// Package server provides the development HTTP server with live reload support
// for the Forge static site generator.
package server

import (
	"bytes"
	"fmt"
	"regexp"
)

// liveReloadScript is the JavaScript injected into HTML pages to enable
// automatic browser reloading when content changes. The first %s is the
// nonce; the second %d is the WebSocket port.
const liveReloadScript = `<script nonce="%s">
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
// The nonce is included in the script tag for CSP compliance.
func InjectLiveReload(html []byte, port int, nonce string) []byte {
	script := fmt.Appendf(nil, liveReloadScript, nonce, port)

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

// scriptTagRe matches opening <script ...> tags (including the closing >).
var scriptTagRe = regexp.MustCompile(`(?i)<script([^>]*)>`)

// InjectScriptNonces adds a nonce attribute to all inline <script> tags
// (those without a src= attribute) that don't already have a nonce.
func InjectScriptNonces(html []byte, nonce string) []byte {
	nonceAttr := []byte(fmt.Sprintf(` nonce="%s"`, nonce))
	return scriptTagRe.ReplaceAllFunc(html, func(match []byte) []byte {
		lower := bytes.ToLower(match)
		// Skip <script> tags that have a src attribute (external scripts).
		if bytes.Contains(lower, []byte("src=")) {
			return match
		}
		// Skip <script> tags that already have a nonce.
		if bytes.Contains(lower, []byte("nonce=")) {
			return match
		}
		// Skip non-JavaScript script types (e.g., application/json, application/ld+json).
		if bytes.Contains(lower, []byte("type=")) &&
			!bytes.Contains(lower, []byte("text/javascript")) &&
			!bytes.Contains(lower, []byte("module")) {
			return match
		}
		// Insert nonce after "<script".
		idx := bytes.Index(lower, []byte("<script"))
		if idx == -1 {
			return match
		}
		insertPos := idx + len("<script")
		result := make([]byte, 0, len(match)+len(nonceAttr))
		result = append(result, match[:insertPos]...)
		result = append(result, nonceAttr...)
		result = append(result, match[insertPos:]...)
		return result
	})
}
