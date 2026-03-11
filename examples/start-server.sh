#!/bin/bash
# Start a simple HTTP server for testing Right Panel HTTP preview
# with CORS support

cd "$(dirname "$0")"

echo "Starting HTTP server on port 8000..."
echo "Serving files from: $(pwd)"
echo ""
echo "Test URLs:"
echo "  - http://localhost:8000/test-page.html"
echo "  - http://localhost:8000/sample.txt"
echo ""
echo "CORS enabled: Yes (Access-Control-Allow-Origin: *)"
echo ""
echo "Press Ctrl+C to stop"

# Use Python HTTP server with CORS support
python3 -c '
import http.server
import socketserver

PORT = 8000

class CORSHandler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "*")
        super().end_headers()

    def do_OPTIONS(self):
        self.send_response(200)
        self.end_headers()

with socketserver.TCPServer(("", PORT), CORSHandler) as httpd:
    print(f"Serving at http://localhost:{PORT}")
    httpd.serve_forever()
'
