const http = require('http');
const fs = require('fs');
const path = require('path');

const port = 8080;
const staticFilesDir = 'dist';
const defaultMimeType = 'application/octet-stream';
const mimeTypeMap = {
    '.html': 'text/html',
    '.js': 'text/javascript',
    '.css': 'text/css',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpg',
    '.wav': 'audio/wav',
    '.wasm': 'application/wasm',
    '.gz': 'application/gzip',
};

const requestHandler = (request, response) => {
    const reqUrl = request.url;
    console.log('request:', request.method, reqUrl);
    const filePath = staticFilesDir + ((reqUrl === '/') ? '/index.html' : reqUrl);
    const extname = path.extname(filePath);
    const contentType = (extname in mimeTypeMap) ? mimeTypeMap[extname] : defaultMimeType;
    console.log('filePath', filePath, 'contentType', contentType);
    fs.readFile(filePath, (error, content) => {
        if (error) {
            console.error('failed to read the file', filePath, 'error', error);
            if (error.code === 'ENOENT') {
                response.writeHead(404);
                response.end('not found', 'utf-8');
                return;
            }
            response.writeHead(500);
            response.end('failed to read the file', 'utf-8');
            return;
        }
        console.log('content.byteLength', content.byteLength);
        response.setHeader('Content-Type', contentType);
        response.setHeader('Content-Length', `${content.byteLength}`);
        response.setHeader('Cross-Origin-Opener-Policy', 'same-origin');
        response.setHeader('Cross-Origin-Embedder-Policy', 'require-corp');
        response.writeHead(200);
        response.end(content);
    });
};

const server = http.createServer(requestHandler);
server.listen(port, () => console.log(`Server listening at http://127.0.0.1:${port}/`));
