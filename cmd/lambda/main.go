package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MGeovany/rivalo-server/internal/auth"
	"github.com/MGeovany/rivalo-server/internal/badge"
	"github.com/MGeovany/rivalo-server/internal/config"
	"github.com/MGeovany/rivalo-server/internal/db"
	"github.com/MGeovany/rivalo-server/internal/httpapi"
	"github.com/MGeovany/rivalo-server/internal/logger"
	"github.com/MGeovany/rivalo-server/internal/pitch"
	"github.com/MGeovany/rivalo-server/internal/profile"
	"github.com/MGeovany/rivalo-server/internal/session"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var handler http.Handler

func main() {
	logger.Init()
	handler = buildHandler()
	lambda.Start(handle)
}

func buildHandler() http.Handler {
	cfg := config.Load()

	var pinger httpapi.Pinger
	var profiles profile.Store
	var sessions session.Store
	var pitches pitch.Store
	var badges badge.Store

	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		database, err := db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Error("database_connect_failed", logger.SafeErr(err))
		} else {
			pinger = database
			profiles = profile.NewPostgresStore(database.Pool)
			sessions = session.NewPostgresStore(database.Pool)
			pitches = pitch.NewPostgresStore(database.Pool)
			badges = badge.NewPostgresStore(database.Pool)
			logger.Info("database_ready")
		}
	} else {
		logger.Warn("database_disabled")
	}

	verifier := buildVerifier(cfg)
	logger.Info("auth_ready", slog.Bool("configured", verifier.Configured()))

	return httpapi.NewRouter(httpapi.Deps{
		DB:       pinger,
		Profiles: profiles,
		Sessions: sessions,
		Pitches:  pitches,
		Badges:   badges,
		Verifier: verifier,
	})
}

func handle(ctx context.Context, event events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	method := event.RequestContext.HTTP.Method
	if method == "" {
		method = http.MethodGet
	}

	body, err := requestBody(event)
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusBadRequest, Body: "invalid request body"}, nil
	}

	path := event.RawPath
	if path == "" {
		path = "/"
	}
	if event.RawQueryString != "" {
		path += "?" + event.RawQueryString
	}

	req, err := http.NewRequestWithContext(ctx, method, path, bytes.NewReader(body))
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusBadRequest, Body: "invalid request"}, nil
	}
	for key, value := range event.Headers {
		req.Header.Set(key, value)
	}

	rec := &responseRecorder{headers: http.Header{}, statusCode: http.StatusOK}
	handler.ServeHTTP(rec, req)

	return events.LambdaFunctionURLResponse{
		StatusCode: rec.statusCode,
		Headers:    flattenHeaders(rec.headers),
		Body:       rec.body.String(),
	}, nil
}

func requestBody(event events.LambdaFunctionURLRequest) ([]byte, error) {
	if event.Body == "" {
		return nil, nil
	}
	if !event.IsBase64Encoded {
		return []byte(event.Body), nil
	}
	body, err := base64.StdEncoding.DecodeString(event.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type responseRecorder struct {
	headers    http.Header
	body       bytes.Buffer
	statusCode int
}

func (r *responseRecorder) Header() http.Header { return r.headers }

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	return r.body.Write(body)
}

func flattenHeaders(headers http.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		out[key] = strings.Join(values, ", ")
	}
	return out
}

func buildVerifier(cfg config.Config) auth.Verifier {
	if cfg.SupabaseURL != "" {
		jwksURL := strings.TrimRight(cfg.SupabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
		verifier, err := auth.NewJWKSVerifier(jwksURL)
		if err != nil {
			logger.Error("auth_jwks_failed", logger.SafeErr(err))
			return auth.Verifier{}
		}
		return verifier
	}
	return auth.NewVerifier(cfg.SupabaseJWTSecret)
}
