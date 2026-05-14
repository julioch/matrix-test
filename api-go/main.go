package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"gonum.org/v1/gonum/mat"
)

type QRRequest struct {
	Matrix [][]float64 `json:"matrix"`
}

type QRResponse struct {
	Input [][]float64    `json:"input"`
	Q     [][]float64    `json:"q"`
	R     [][]float64    `json:"r"`
	Stats map[string]any `json:"stats,omitempty"`
	Meta  map[string]any `json:"meta,omitempty"`
	Error string         `json:"error,omitempty"`
}

type TokenRequest struct {
	Sub string `json:"sub"`
}

// JWT_SECRET y API_KEY se leen desde variables de entorno para no hardcodear credenciales.
func jwtSecretFromEnv() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET no está configurado")
	}
	return []byte(secret), nil
}

func apiKeyFromEnv() (string, error) {
	key := os.Getenv("API_KEY")
	if key == "" {
		return "", errors.New("API_KEY no está configurado")
	}
	return key, nil
}

// issueToken emite un JWT HS256 de corta duración para proteger llamadas.
func issueToken(sub string, secret []byte) (string, error) {
	if strings.TrimSpace(sub) == "" {
		sub = "user"
	}
	claims := jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func bearerFromHeader(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

func authMiddleware(secret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}
		if c.Path() == "/health" || c.Path() == "/token" {
			return c.Next()
		}

		raw := c.Get("Authorization")
		tokenStr, ok := bearerFromHeader(raw)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Falta Authorization: Bearer <token>"})
		}

		parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("algoritmo JWT inválido")
			}
			return secret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !parsed.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "JWT inválido o expirado"})
		}

		return c.Next()
	}
}

// validateRectangularMatrix asegura que la entrada sea una matriz rectangular y con valores finitos.
func validateRectangularMatrix(m [][]float64) (rows int, cols int, err error) {
	if len(m) == 0 {
		return 0, 0, errors.New("matrix debe ser un array de arrays no vacío")
	}
	if len(m[0]) == 0 {
		return 0, 0, errors.New("matrix debe tener al menos una columna")
	}
	cols = len(m[0])
	for i := 0; i < len(m); i++ {
		if len(m[i]) != cols {
			return 0, 0, errors.New("matrix debe ser rectangular (todas las filas del mismo tamaño)")
		}
		for j := 0; j < cols; j++ {
			v := m[i][j]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return 0, 0, errors.New("matrix solo puede contener números finitos")
			}
		}
	}
	return len(m), cols, nil
}

func denseFromMatrix(m [][]float64) *mat.Dense {
	rows := len(m)
	cols := len(m[0])
	data := make([]float64, 0, rows*cols)
	for i := 0; i < rows; i++ {
		data = append(data, m[i]...)
	}
	return mat.NewDense(rows, cols, data)
}

func denseToSlices(d mat.Matrix) [][]float64 {
	r, c := d.Dims()
	out := make([][]float64, r)
	for i := 0; i < r; i++ {
		row := make([]float64, c)
		for j := 0; j < c; j++ {
			row[j] = d.At(i, j)
		}
		out[i] = row
	}
	return out
}

// thinQR calcula una factorización QR "thin" para matrices con m>=n.
func thinQR(a *mat.Dense) (q [][]float64, r [][]float64, meta map[string]any, err error) {
	m, n := a.Dims()
	if m < n {
		return nil, nil, nil, errors.New("matrix debe tener filas >= columnas para la factorización QR (m>=n)")
	}

	var qr mat.QR
	qr.Factorize(a)

	var qFull mat.Dense
	qr.QTo(&qFull)

	var rFull mat.Dense
	qr.RTo(&rFull)

	k := n
	qThin := qFull.Slice(0, m, 0, k)
	rThin := rFull.Slice(0, k, 0, n)

	return denseToSlices(qThin), denseToSlices(rThin), map[string]any{
		"m": m,
		"n": n,
		"k": k,
	}, nil
}

func postJSON(ctx context.Context, url string, body any, headers map[string]string) (map[string]any, int, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode >= 400 {
		return out, resp.StatusCode, errors.New("error de API Node")
	}

	return out, resp.StatusCode, nil
}

// newApp construye la API.
// Endpoints:
// - GET /health (sin auth)
// - POST /token (x-api-key, sin auth) => emite JWT para consumir endpoints protegidos
// - POST /qr (con JWT) => calcula QR y, si nodeBaseURL está configurado, consulta /stats en la API Node.
func newApp(secret []byte, apiKey string, nodeBaseURL string) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "reto-interseguro-api-go",
	})

	allowOrigins := os.Getenv("CORS_ORIGIN")
	if allowOrigins == "" {
		allowOrigins = "http://localhost:5173,http://127.0.0.1:5173"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-API-Key",
		AllowMethods: "GET,POST,OPTIONS",
	}))

	app.Use(authMiddleware(secret))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Post("/token", func(c *fiber.Ctx) error {
		if c.Get("X-API-Key") != apiKey && c.Get("x-api-key") != apiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "API key inválida"})
		}

		var req TokenRequest
		if err := c.BodyParser(&req); err != nil {
			req.Sub = "user"
		}

		token, err := issueToken(req.Sub, secret)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "no se pudo emitir token"})
		}

		return c.JSON(fiber.Map{"token": token})
	})

	app.Post("/qr", func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error interno al calcular QR"})
			}
		}()

		var req QRRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "JSON inválido"})
		}

		rows, cols, err := validateRectangularMatrix(req.Matrix)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		a := denseFromMatrix(req.Matrix)
		q, r, meta, err := thinQR(a)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		resp := QRResponse{
			Input: req.Matrix,
			Q:     q,
			R:     r,
			Stats: nil,
			Meta: map[string]any{
				"rows": rows,
				"cols": cols,
				"qr":   meta,
			},
		}

		if nodeBaseURL != "" {
			ctx, cancel := context.WithTimeout(c.Context(), 6*time.Second)
			defer cancel()

			serviceToken, tokenErr := issueToken("api-go", secret)
			headers := map[string]string{}
			if tokenErr == nil {
				headers["Authorization"] = "Bearer " + serviceToken
			}

			stats, _, statsErr := postJSON(ctx, nodeBaseURL+"/stats", map[string]any{
				"matrices": map[string]any{
					"q": q,
					"r": r,
				},
			}, headers)

			if statsErr == nil {
				resp.Stats = stats
			} else {
				resp.Meta["statsError"] = true
			}
		}

		return c.JSON(resp)
	})

	return app
}

func main() {
	_ = godotenv.Load()

	secret, err := jwtSecretFromEnv()
	if err != nil {
		panic(err)
	}
	apiKey, err := apiKeyFromEnv()
	if err != nil {
		panic(err)
	}

	nodeBaseURL := os.Getenv("NODE_API_URL")
	if nodeBaseURL == "" {
		nodeBaseURL = "http://localhost:3000"
	}

	app := newApp(secret, apiKey, nodeBaseURL)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if _, err := strconv.Atoi(port); err != nil {
		port = "8080"
	}

	if err := app.Listen(":" + port); err != nil {
		panic(err)
	}
}
