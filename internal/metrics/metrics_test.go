package metrics

import (
	"testing"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestRecordReviewCommentsPosted(t *testing.T) {
	before := counterValue(t, reviewCommentsTotal)

	DefaultRecorder.RecordReviewCommentsPosted(3)
	DefaultRecorder.RecordReviewCommentsPosted(0)
	DefaultRecorder.RecordReviewCommentsPosted(-1)

	require.Equal(t, before+3, counterValue(t, reviewCommentsTotal))
}

func TestRecordLLMError(t *testing.T) {
	reviewErrors := llmErrorsTotal.WithLabelValues("agent", OperationReview)
	replyErrors := llmErrorsTotal.WithLabelValues("openai", OperationReply)

	reviewErrorsBefore := counterValue(t, reviewErrors)
	replyErrorsBefore := counterValue(t, replyErrors)

	DefaultRecorder.RecordLLMError(OperationReview, &config.Config{
		Agent: config.Agent{Command: "codex exec -"},
	})
	DefaultRecorder.RecordLLMError(OperationReply, &config.Config{
		OpenAI: config.OpenAI{APIKey: "key"},
	})

	require.Equal(t, reviewErrorsBefore+1, counterValue(t, reviewErrors))
	require.Equal(t, replyErrorsBefore+1, counterValue(t, replyErrors))
}

func counterValue(t *testing.T, metric prometheus.Metric) float64 {
	t.Helper()

	var dtoMetric dto.Metric
	require.NoError(t, metric.Write(&dtoMetric))
	require.NotNil(t, dtoMetric.Counter)

	return dtoMetric.Counter.GetValue()
}
