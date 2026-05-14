require("dotenv").config();

const express = require("express");
const jwt = require("jsonwebtoken");

function isFiniteNumber(value) {
  return typeof value === "number" && Number.isFinite(value);
}

// JWT_SECRET y API_KEY se consumen desde variables de entorno para evitar credenciales en el código.
function getJwtSecret() {
  const secret = process.env.JWT_SECRET;
  if (!secret) {
    throw new Error("JWT_SECRET no está configurado");
  }
  return secret;
}

// issueToken emite un JWT HS256 de corta duración para proteger llamadas a endpoints.
function issueToken(sub, secret) {
  return jwt.sign({ sub }, secret, { algorithm: "HS256", expiresIn: "15m" });
}

function requireApiKey(req) {
  const expected = process.env.API_KEY;
  if (!expected) {
    return { ok: false, error: "API_KEY no está configurado" };
  }
  const received = req.header("x-api-key");
  if (!received || received !== expected) {
    return { ok: false, error: "API key inválida" };
  }
  return { ok: true };
}

// authMiddleware protege los endpoints. /health y /token quedan sin JWT para permitir health-checks y emisión de tokens.
function authMiddleware(req, res, next) {
  if (req.path === "/health" || req.path === "/token") return next();

  const auth = req.header("authorization");
  if (!auth || !auth.toLowerCase().startsWith("bearer ")) {
    return res.status(401).json({ error: "Falta Authorization: Bearer <token>" });
  }

  const token = auth.slice(7).trim();
  try {
    const secret = getJwtSecret();
    req.user = jwt.verify(token, secret, { algorithms: ["HS256"] });
    return next();
  } catch (_err) {
    return res.status(401).json({ error: "JWT inválido o expirado" });
  }
}

function validateRectangularMatrix(matrix, name) {
  if (!Array.isArray(matrix) || matrix.length === 0) {
    return { ok: false, error: `${name} debe ser un array de arrays no vacío` };
  }

  const firstRow = matrix[0];
  if (!Array.isArray(firstRow) || firstRow.length === 0) {
    return { ok: false, error: `${name} debe tener al menos una columna` };
  }

  const cols = firstRow.length;
  for (let i = 0; i < matrix.length; i += 1) {
    const row = matrix[i];
    if (!Array.isArray(row) || row.length !== cols) {
      return { ok: false, error: `${name} debe ser rectangular (todas las filas del mismo tamaño)` };
    }
    for (let j = 0; j < row.length; j += 1) {
      if (!isFiniteNumber(row[j])) {
        return { ok: false, error: `${name} solo puede contener números finitos` };
      }
    }
  }

  return { ok: true, rows: matrix.length, cols };
}

function flattenMatrix(matrix) {
  const values = [];
  for (const row of matrix) {
    for (const v of row) values.push(v);
  }
  return values;
}

function isDiagonalMatrix(matrix, eps = 1e-9) {
  const rows = matrix.length;
  const cols = matrix[0].length;
  if (rows !== cols) return false;

  for (let i = 0; i < rows; i += 1) {
    for (let j = 0; j < cols; j += 1) {
      if (i === j) continue;
      if (Math.abs(matrix[i][j]) > eps) return false;
    }
  }
  return true;
}

function computeStats(matrices) {
  const all = [];
  for (const matrix of matrices) {
    all.push(...flattenMatrix(matrix));
  }

  let sum = 0;
  let min = all[0];
  let max = all[0];
  for (const v of all) {
    sum += v;
    if (v < min) min = v;
    if (v > max) max = v;
  }

  return {
    max,
    min,
    sum,
    avg: sum / all.length
  };
}

function createApp() {
  const app = express();
  app.use(express.json({ limit: "1mb" }));
  app.use(authMiddleware);

  app.get("/health", (_req, res) => {
    res.json({ status: "ok" });
  });

  // /token: se exige API key y se retorna un JWT para consumir endpoints protegidos.
  app.post("/token", (req, res) => {
    const apiKey = requireApiKey(req);
    if (!apiKey.ok) return res.status(401).json({ error: apiKey.error });

    const sub = typeof req.body?.sub === "string" && req.body.sub.trim() ? req.body.sub.trim() : "user";
    const secret = getJwtSecret();
    const token = issueToken(sub, secret);
    return res.json({ token });
  });

  app.post("/stats", (req, res) => {
    const body = req.body ?? {};

    const q = body?.matrices?.q ?? body?.q;
    const r = body?.matrices?.r ?? body?.r;

    if (q == null || r == null) {
      return res.status(400).json({ error: "Se requieren matrices q y r" });
    }

    const qVal = validateRectangularMatrix(q, "q");
    if (!qVal.ok) return res.status(400).json({ error: qVal.error });

    const rVal = validateRectangularMatrix(r, "r");
    if (!rVal.ok) return res.status(400).json({ error: rVal.error });

    const stats = computeStats([q, r]);
    const eps = isFiniteNumber(body?.eps) ? body.eps : 1e-9;

    return res.json({
      ...stats,
      isDiagonalQ: isDiagonalMatrix(q, eps),
      isDiagonalR: isDiagonalMatrix(r, eps)
    });
  });

  return app;
}

if (require.main === module) {
  const port = Number.parseInt(process.env.PORT ?? "3000", 10);
  const app = createApp();
  app.listen(port, () => {
    process.stdout.write(`api-node escuchando en el puerto ${port}\n`);
  });
}

module.exports = { createApp };
