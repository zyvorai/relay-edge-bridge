// Custom HTTPS server for production deploys that don't sit behind a
// reverse proxy already terminating TLS. Dev (`npm run dev`) still uses
// Next's own plain-HTTP server; this is only used by `npm run start:https`.
const { createServer } = require("https");
const { parse } = require("url");
const fs = require("fs");
const next = require("next");

const port = parseInt(process.env.PORT || "3443", 10);
const app = next({ dev: false });
const handle = app.getRequestHandler();

const httpsOptions = {
  key: fs.readFileSync(process.env.TLS_KEY_FILE || "./certs/key.pem"),
  cert: fs.readFileSync(process.env.TLS_CERT_FILE || "./certs/cert.pem"),
};

app.prepare().then(() => {
  createServer(httpsOptions, (req, res) => {
    handle(req, res, parse(req.url, true));
  }).listen(port, () => {
    console.log(`relay-edge-console listening on https://0.0.0.0:${port}`);
  });
});
