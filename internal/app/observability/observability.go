package observability

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cbtlms/internal/auth"

	"github.com/go-chi/chi/v5/middleware"
)

type key struct {
	Method string
	Path   string
	Status int
}

type stat struct {
	Count     int64
	LatencyMS float64
}

type Collector struct {
	db *sql.DB

	mu           sync.RWMutex
	requestStats map[key]stat
	startedAt    time.Time
}

func NewCollector(db *sql.DB) *Collector {
	return &Collector{
		db:           db,
		requestStats: make(map[key]stat),
		startedAt:    time.Now(),
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (c *Collector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		latencyMS := float64(time.Since(start).Microseconds()) / 1000.0
		path := normalizedPath(r.URL.Path)

		c.mu.Lock()
		k := key{Method: r.Method, Path: path, Status: rec.status}
		s := c.requestStats[k]
		s.Count++
		s.LatencyMS += latencyMS
		c.requestStats[k] = s
		c.mu.Unlock()

		requestID := middleware.GetReqID(r.Context())
		userID := int64(0)
		if u, ok := auth.CurrentUser(r.Context()); ok {
			userID = u.ID
		}
		attemptID := extractAttemptID(path)

		entry := map[string]any{
			"request_id": requestID,
			"user_id":    userID,
			"attempt_id": attemptID,
			"method":     r.Method,
			"path":       path,
			"status":     rec.status,
			"latency_ms": latencyMS,
			"remote_ip":  strings.TrimSpace(r.RemoteAddr),
		}
		b, _ := json.Marshal(entry)
		log.Printf("%s", string(b))
	})
}

func (c *Collector) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	statsCopy := make(map[key]stat, len(c.requestStats))
	for k, v := range c.requestStats {
		statsCopy[k] = v
	}
	startedAt := c.startedAt
	c.mu.RUnlock()

	keys := make([]key, 0, len(statsCopy))
	for k := range statsCopy {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Method != keys[j].Method {
			return keys[i].Method < keys[j].Method
		}
		if keys[i].Path != keys[j].Path {
			return keys[i].Path < keys[j].Path
		}
		return keys[i].Status < keys[j].Status
	})

	var sb strings.Builder
	sb.WriteString("# cbtlms observability metrics\n")
	sb.WriteString("# TYPE cbtlms_uptime_seconds gauge\n")
	sb.WriteString(fmt.Sprintf("cbtlms_uptime_seconds %.0f\n", time.Since(startedAt).Seconds()))

	sb.WriteString("# TYPE cbtlms_http_requests_total counter\n")
	sb.WriteString("# TYPE cbtlms_http_request_latency_ms_sum counter\n")
	sb.WriteString("# TYPE cbtlms_http_request_latency_ms_avg gauge\n")
	for _, k := range keys {
		s := statsCopy[k]
		labels := fmt.Sprintf("method=\"%s\",path=\"%s\",status=\"%d\"", k.Method, k.Path, k.Status)
		sb.WriteString(fmt.Sprintf("cbtlms_http_requests_total{%s} %d\n", labels, s.Count))
		sb.WriteString(fmt.Sprintf("cbtlms_http_request_latency_ms_sum{%s} %.3f\n", labels, s.LatencyMS))
		avg := 0.0
		if s.Count > 0 {
			avg = s.LatencyMS / float64(s.Count)
		}
		sb.WriteString(fmt.Sprintf("cbtlms_http_request_latency_ms_avg{%s} %.3f\n", labels, avg))
	}

	if c.db != nil {
		dbs := c.db.Stats()
		sb.WriteString("# TYPE cbtlms_db_open_connections gauge\n")
		sb.WriteString(fmt.Sprintf("cbtlms_db_open_connections %d\n", dbs.OpenConnections))
		sb.WriteString("# TYPE cbtlms_db_in_use_connections gauge\n")
		sb.WriteString(fmt.Sprintf("cbtlms_db_in_use_connections %d\n", dbs.InUse))
		sb.WriteString("# TYPE cbtlms_db_idle_connections gauge\n")
		sb.WriteString(fmt.Sprintf("cbtlms_db_idle_connections %d\n", dbs.Idle))
		sb.WriteString("# TYPE cbtlms_db_wait_count counter\n")
		sb.WriteString(fmt.Sprintf("cbtlms_db_wait_count %d\n", dbs.WaitCount))
		sb.WriteString("# TYPE cbtlms_db_wait_duration_ms counter\n")
		sb.WriteString(fmt.Sprintf("cbtlms_db_wait_duration_ms %.3f\n", float64(dbs.WaitDuration.Microseconds())/1000.0))
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sb.String()))
}

func normalizedPath(path string) string {
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if _, err := strconv.ParseInt(p, 10, 64); err == nil {
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

func extractAttemptID(path string) int64 {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "attempts" {
			if id, err := strconv.ParseInt(parts[i+1], 10, 64); err == nil {
				return id
			}
		}
	}
	return 0
}
