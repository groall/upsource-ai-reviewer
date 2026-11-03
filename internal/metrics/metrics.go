package metrics

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	OperationReview = "review"
	OperationReply  = "reply"

	// CodexCapabilityCommandExecution identifies Codex shell command execution.
	CodexCapabilityCommandExecution = "command_execution"
	// CodexCapabilityNodaKnowledgeMCP identifies calls to the Noda Knowledge MCP server.
	CodexCapabilityNodaKnowledgeMCP = "noda_knowledge_mcp"

	// CodexCapabilityUnknown means no attempt has been observed yet.
	CodexCapabilityUnknown = "unknown"
	// CodexCapabilitySuccess means the last observed attempt completed successfully.
	CodexCapabilitySuccess = "success"
	// CodexCapabilityFailure means the last observed attempt failed or was denied.
	CodexCapabilityFailure = "failure"
)

type Recorder interface {
	RecordReviewReviewed()
	RecordReplySent()
	RecordReviewCommentsPosted(count int)
	RecordLLMError(operation, currentProvider string)
	RecordCodexCapability(capability, result string)
}

type prometheusRecorder struct{}

var (
	DefaultRecorder Recorder = prometheusRecorder{}

	reviewsReviewedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "upsource_ai_reviewer_reviews_reviewed_total",
		Help: "Total number of reviews processed by the AI reviewer.",
	})
	repliesSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "upsource_ai_reviewer_replies_sent_total",
		Help: "Total number of follow-up replies sent by the AI reviewer.",
	})
	reviewCommentsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "upsource_ai_reviewer_review_comments_total",
		Help: "Total number of review comments posted by the AI reviewer.",
	})
	llmErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "upsource_ai_reviewer_llm_errors_total",
		Help: "Total number of errors received from the current LLM provider.",
	}, []string{"provider", "operation"})
	codexCapabilityState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "upsource_ai_reviewer_codex_capability_state",
		Help: "Last observed Codex capability result, represented as one-hot state labels; unknown means the capability has not been observed.",
	}, []string{"capability", "result"})
	codexCapabilityLastChecked = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "upsource_ai_reviewer_codex_capability_last_checked_timestamp_seconds",
		Help: "Unix timestamp of the last observed Codex capability result.",
	}, []string{"capability"})
)

func init() {
	for _, provider := range []string{"agent", "openai", "gemini", "anthropic"} {
		for _, operation := range []string{OperationReview, OperationReply} {
			llmErrorsTotal.WithLabelValues(provider, operation)
		}
	}

	for _, capability := range []string{CodexCapabilityCommandExecution, CodexCapabilityNodaKnowledgeMCP} {
		codexCapabilityState.WithLabelValues(capability, CodexCapabilityUnknown).Set(1)
		codexCapabilityState.WithLabelValues(capability, CodexCapabilitySuccess).Set(0)
		codexCapabilityState.WithLabelValues(capability, CodexCapabilityFailure).Set(0)
		codexCapabilityLastChecked.WithLabelValues(capability).Set(0)
	}
}

func StartServer(ctx context.Context, cfg config.Metrics) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = ":2112"
	}
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.Path, promhttp.Handler())

	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Starting metrics server on %s%s", cfg.ListenAddress, cfg.Path)
		errCh <- server.ListenAndServe()
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Failed to shut down metrics server: %v", err)
		}
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("metrics server failed: %w", err)
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (prometheusRecorder) RecordReviewReviewed() {
	reviewsReviewedTotal.Inc()
}

func (prometheusRecorder) RecordReplySent() {
	repliesSentTotal.Inc()
}

func (prometheusRecorder) RecordReviewCommentsPosted(count int) {
	if count <= 0 {
		return
	}

	reviewCommentsTotal.Add(float64(count))
}

func (prometheusRecorder) RecordLLMError(operation, currentProvider string) {
	llmErrorsTotal.WithLabelValues(currentProvider, operation).Inc()
}

func (prometheusRecorder) RecordCodexCapability(capability, result string) {
	if capability != CodexCapabilityCommandExecution && capability != CodexCapabilityNodaKnowledgeMCP {
		return
	}
	if result != CodexCapabilitySuccess && result != CodexCapabilityFailure {
		return
	}

	for _, state := range []string{CodexCapabilityUnknown, CodexCapabilitySuccess, CodexCapabilityFailure} {
		codexCapabilityState.WithLabelValues(capability, state).Set(0)
	}
	codexCapabilityState.WithLabelValues(capability, result).Set(1)
	codexCapabilityLastChecked.WithLabelValues(capability).SetToCurrentTime()
}
