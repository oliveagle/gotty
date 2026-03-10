#!/bin/bash
# Start a simple HTTP server for testing Right Panel HTTP preview

cd "$(dirname "$0")"

echo "Starting HTTP server on port 8000..."
echo "Serving files from: $(pwd)"
echo ""
echo "Test URLs:"
echo "  - http://localhost:8000/test-page.html"
echo "  - http://localhost:8000/sample.txt"
echo ""
echo "Press Ctrl+C to stop"

python3 -m http.server 8000
