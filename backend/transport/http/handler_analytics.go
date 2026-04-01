package http

import (
	"net/http"
	"time"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
)

func (s *Server) handleAnalyticsCumulative(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filter := parseAnalyticsFilter(r)
	points, err := s.analytics.GetCumulativeUsage(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if points == nil {
		points = []service.CumulativeDataPoint{}
	}

	writeJSON(w, map[string]any{
		"data":  points,
		"count": len(points),
	})
}

func (s *Server) handleAnalyticsProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filter := parseAnalyticsFilter(r)
	result, err := s.analytics.GetProviderComparison(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result == nil {
		result = make(map[domain.ProviderID]service.ProviderStats)
	}

	writeJSON(w, map[string]any{
		"providers": result,
	})
}

func (s *Server) handleAnalyticsModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filter := parseAnalyticsFilter(r)
	models, err := s.analytics.GetModelPerformance(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if models == nil {
		models = []service.ModelPerformance{}
	}

	writeJSON(w, map[string]any{
		"models": models,
		"count":  len(models),
	})
}

func (s *Server) handleAnalyticsHourly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filter := parseAnalyticsFilter(r)
	buckets, err := s.analytics.GetHourlyUsage(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if buckets == nil {
		buckets = []service.HourlyBucket{}
	}

	writeJSON(w, map[string]any{
		"buckets": buckets,
		"count":   len(buckets),
	})
}

// parseAnalyticsFilter builds a TimeRangeFilter from query params.
// Invalid timestamps are silently ignored.
func parseAnalyticsFilter(r *http.Request) port.TimeRangeFilter {
	q := r.URL.Query()
	var filter port.TimeRangeFilter
	if s := q.Get("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.Start = t
		}
	}
	if e := q.Get("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			filter.End = t
		}
	}
	if p := q.Get("provider"); p != "" {
		filter.Provider = domain.ProviderID(p)
	}
	return filter
}
