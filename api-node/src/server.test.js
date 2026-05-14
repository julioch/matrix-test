const request = require("supertest");
const { createApp } = require("./server");

function authHeader(token) {
  return { Authorization: `Bearer ${token}` };
}

describe("api-node", () => {
  beforeEach(() => {
    process.env.JWT_SECRET = "test-secret";
    process.env.API_KEY = "test-api-key";
  });

  test("GET /health no requiere JWT", async () => {
    const app = createApp();
    const res = await request(app).get("/health");
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ status: "ok" });
  });

  test("POST /stats requiere JWT", async () => {
    const app = createApp();
    const res = await request(app).post("/stats").send({ q: [[1]], r: [[2]] });
    expect(res.status).toBe(401);
  });

  test("POST /token emite JWT con API key", async () => {
    const app = createApp();
    const res = await request(app).post("/token").set("x-api-key", "test-api-key").send({ sub: "julio" });
    expect(res.status).toBe(200);
    expect(typeof res.body.token).toBe("string");
    expect(res.body.token.length).toBeGreaterThan(20);
  });

  test("POST /stats calcula estadísticas", async () => {
    const app = createApp();
    const tokenRes = await request(app).post("/token").set("x-api-key", "test-api-key").send({ sub: "tester" });
    const token = tokenRes.body.token;

    const payload = {
      matrices: {
        q: [
          [1, 0],
          [0, 2]
        ],
        r: [
          [3, 4],
          [0, 5]
        ]
      }
    };

    const res = await request(app).post("/stats").set(authHeader(token)).send(payload);
    expect(res.status).toBe(200);
    expect(res.body.min).toBe(0);
    expect(res.body.max).toBe(5);
    expect(res.body.sum).toBe(15);
    expect(res.body.avg).toBeCloseTo(15 / 8);
    expect(res.body.isDiagonalQ).toBe(true);
    expect(res.body.isDiagonalR).toBe(false);
  });
});

