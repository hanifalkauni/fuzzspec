import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const PORT = 8989;

const server = http.createServer((req, res) => {
  const url = new URL(req.url, `http://localhost:${PORT}`);
  const pathname = url.pathname;
  const method = req.method;

  // 1. Auto-Discovery Endpoints
  if ((pathname === '/openapi.json' || pathname === '/swagger/doc.json') && method === 'GET') {
    const yamlPath = path.join(__dirname, 'openapi.yaml');
    if (fs.existsSync(yamlPath)) {
      res.writeHead(200, { 'Content-Type': 'application/x-yaml' });
      res.end(fs.readFileSync(yamlPath, 'utf8'));
      return;
    }
  }

  // 2. GET /api/v1/products - [PATCHED: Robust Boundary Validation]
  if (pathname === '/api/v1/products' && method === 'GET') {
    const limitStr = url.searchParams.get('limit');
    if (limitStr !== null) {
      const parsedLimit = Number(limitStr);
      if (isNaN(parsedLimit) || !Number.isInteger(parsedLimit) || parsedLimit < 1 || parsedLimit > 100) {
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
          error: "Invalid query parameter",
          message: "Query parameter 'limit' must be an integer between 1 and 100"
        }));
        return;
      }
    }

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify([
      { id: 1, title: "Wireless Headphones", price: 79.99, in_stock: true },
      { id: 2, title: "Mechanical Keyboard", price: 129.50, in_stock: true }
    ]));
    return;
  }

  // 3. GET /api/v1/products/:id - [PATCHED: Safe ID Validation]
  if (pathname.startsWith('/api/v1/products/') && method === 'GET') {
    const idStr = pathname.replace('/api/v1/products/', '');
    const id = Number(idStr);
    if (isNaN(id) || !Number.isInteger(id) || id < 1) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: "Invalid product ID. Must be positive integer." }));
      return;
    }

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ id: id, title: "Test Product", price: 49.99, in_stock: true }));
    return;
  }

  // 4. POST /api/v1/orders - [PATCHED: Safe Validation Interceptor]
  if (pathname === '/api/v1/orders' && method === 'POST') {
    let body = '';
    req.on('data', chunk => { body += chunk; });
    req.on('end', () => {
      try {
        const payload = JSON.parse(body || '{}');

        // Validation interceptor returning handled 422
        if (payload.quantity === undefined || payload.quantity === null || typeof payload.quantity !== 'number' || payload.quantity < 1 || payload.quantity > 50) {
          res.writeHead(422, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({
            error: "Unprocessable Entity",
            message: "Validation failed: 'quantity' must be a valid integer between 1 and 50"
          }));
          return;
        }

        // Handled valid response
        res.writeHead(201, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
          order_id: "ord_" + Math.floor(Math.random() * 10000),
          status: "confirmed",
          total_price: payload.quantity * 25.00
        }));
      } catch (err) {
        // Handled bad json
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: "Malformed JSON payload" }));
      }
    });
    return;
  }

  // 5. POST /api/v1/products - [PATCHED: Conforming to OpenAPI contract]
  if (pathname === '/api/v1/products' && method === 'POST') {
    let body = '';
    req.on('data', chunk => { body += chunk; });
    req.on('end', () => {
      try {
        const payload = JSON.parse(body || '{}');
        if (!payload.title || typeof payload.title !== 'string' || typeof payload.price !== 'number' || payload.price < 0.01) {
          res.writeHead(422, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ error: "Validation failed on product schema" }));
          return;
        }

        res.writeHead(201, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
          id: Math.floor(Math.random() * 1000) + 1,
          title: payload.title,
          price: payload.price,
          in_stock: true
        }));
      } catch {
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: "Malformed JSON" }));
      }
    });
    return;
  }

  res.writeHead(404, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: "Not Found" }));
});

server.listen(PORT, () => {
  console.log(`🚀 Dummy Store API (PATCHED) running at http://localhost:${PORT}`);
});
