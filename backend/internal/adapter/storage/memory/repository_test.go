package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/adapter/storage/memory"
	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

// makeUsage is a convenience constructor for test NormalizedUsage records.
func makeUsage(sessionID string, provider domain.ProviderID, ts time.Time) domain.NormalizedUsage {
	return domain.NormalizedUsage{
		ID:        sessionID + "-" + string(provider),
		SessionID: sessionID,
		Provider:  provider,
		Model:     "test-model",
		Timestamp: ts,
		Usage: domain.TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
		CostUSD: 0.001,
	}
}

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

func TestRepository_Save_AppendsSingle(t *testing.T) {
	repo := memory.NewRepository()
	u := makeUsage("s1", domain.ProviderOpenAI, time.Now())

	err := repo.Save(context.Background(), u)
	require.NoError(t, err)

	results, err := repo.FindBySession(context.Background(), "s1", 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestRepository_Save_AppendsMultiple(t *testing.T) {
	repo := memory.NewRepository()
	for i := 0; i < 5; i++ {
		err := repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
		require.NoError(t, err)
	}

	results, err := repo.FindBySession(context.Background(), "s1", 0)
	require.NoError(t, err)
	assert.Len(t, results, 5)
}

// ---------------------------------------------------------------------------
// FindBySession
// ---------------------------------------------------------------------------

func TestRepository_FindBySession_MostRecentFirst(t *testing.T) {
	repo := memory.NewRepository()
	base := time.Now()

	u1 := makeUsage("s1", domain.ProviderOpenAI, base.Add(-2*time.Second))
	u1.ID = "old"
	u2 := makeUsage("s1", domain.ProviderOpenAI, base.Add(-1*time.Second))
	u2.ID = "mid"
	u3 := makeUsage("s1", domain.ProviderOpenAI, base)
	u3.ID = "new"

	_ = repo.Save(context.Background(), u1)
	_ = repo.Save(context.Background(), u2)
	_ = repo.Save(context.Background(), u3)

	results, err := repo.FindBySession(context.Background(), "s1", 0)
	require.NoError(t, err)
	require.Len(t, results, 3)
	// Reverse order: most recent first
	assert.Equal(t, "new", results[0].ID)
	assert.Equal(t, "mid", results[1].ID)
	assert.Equal(t, "old", results[2].ID)
}

func TestRepository_FindBySession_LimitEnforced(t *testing.T) {
	repo := memory.NewRepository()
	for i := 0; i < 10; i++ {
		_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	}

	results, err := repo.FindBySession(context.Background(), "s1", 3)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestRepository_FindBySession_ZeroLimitReturnsAll(t *testing.T) {
	repo := memory.NewRepository()
	for i := 0; i < 7; i++ {
		_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	}

	results, err := repo.FindBySession(context.Background(), "s1", 0)
	require.NoError(t, err)
	assert.Len(t, results, 7)
}

func TestRepository_FindBySession_FiltersOtherSessions(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	_ = repo.Save(context.Background(), makeUsage("s2", domain.ProviderOpenAI, time.Now()))
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderAnthropic, time.Now()))

	results, err := repo.FindBySession(context.Background(), "s1", 0)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "s1", r.SessionID)
	}
}

func TestRepository_FindBySession_NotFound(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))

	results, err := repo.FindBySession(context.Background(), "nonexistent", 0)
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ---------------------------------------------------------------------------
// FindByTimeRange
// ---------------------------------------------------------------------------

func TestRepository_FindByTimeRange_TimeFilter(t *testing.T) {
	repo := memory.NewRepository()
	now := time.Now()

	old := makeUsage("s1", domain.ProviderOpenAI, now.Add(-10*time.Minute))
	recent := makeUsage("s1", domain.ProviderOpenAI, now.Add(-1*time.Minute))
	newest := makeUsage("s1", domain.ProviderOpenAI, now)

	_ = repo.Save(context.Background(), old)
	_ = repo.Save(context.Background(), recent)
	_ = repo.Save(context.Background(), newest)

	filter := port.TimeRangeFilter{
		Start: now.Add(-5 * time.Minute),
	}
	results, err := repo.FindByTimeRange(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, results, 2) // recent + newest
}

func TestRepository_FindByTimeRange_EndFilter(t *testing.T) {
	repo := memory.NewRepository()
	now := time.Now()

	old := makeUsage("s1", domain.ProviderOpenAI, now.Add(-10*time.Minute))
	recent := makeUsage("s1", domain.ProviderOpenAI, now.Add(-1*time.Minute))

	_ = repo.Save(context.Background(), old)
	_ = repo.Save(context.Background(), recent)

	filter := port.TimeRangeFilter{
		End: now.Add(-5 * time.Minute),
	}
	results, err := repo.FindByTimeRange(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, results, 1) // only old
}

func TestRepository_FindByTimeRange_ProviderFilter(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderAnthropic, time.Now()))
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))

	filter := port.TimeRangeFilter{Provider: domain.ProviderOpenAI}
	results, err := repo.FindByTimeRange(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, domain.ProviderOpenAI, r.Provider)
	}
}

func TestRepository_FindByTimeRange_SessionFilter(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	_ = repo.Save(context.Background(), makeUsage("s2", domain.ProviderOpenAI, time.Now()))

	filter := port.TimeRangeFilter{SessionID: "s2"}
	results, err := repo.FindByTimeRange(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "s2", results[0].SessionID)
}

func TestRepository_FindByTimeRange_EmptyFilter(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	_ = repo.Save(context.Background(), makeUsage("s2", domain.ProviderAnthropic, time.Now()))

	results, err := repo.FindByTimeRange(context.Background(), port.TimeRangeFilter{})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// ---------------------------------------------------------------------------
// SumByProvider
// ---------------------------------------------------------------------------

func TestRepository_SumByProvider_Aggregates(t *testing.T) {
	repo := memory.NewRepository()
	u1 := makeUsage("s1", domain.ProviderOpenAI, time.Now())
	u1.Usage.InputTokens = 200
	u1.Usage.OutputTokens = 100
	u1.CostUSD = 0.002

	u2 := makeUsage("s1", domain.ProviderOpenAI, time.Now())
	u2.Usage.InputTokens = 300
	u2.Usage.OutputTokens = 150
	u2.CostUSD = 0.003

	_ = repo.Save(context.Background(), u1)
	_ = repo.Save(context.Background(), u2)

	sums, err := repo.SumByProvider(context.Background(), port.TimeRangeFilter{})
	require.NoError(t, err)

	s := sums[domain.ProviderOpenAI]
	assert.Equal(t, int64(500), s.TotalInput)
	assert.Equal(t, int64(250), s.TotalOutput)
	assert.InDelta(t, 0.005, s.TotalCost, 0.0001)
	assert.Equal(t, int64(2), s.RequestCount)
}

func TestRepository_SumByProvider_MultipleProviders(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderAnthropic, time.Now()))

	sums, err := repo.SumByProvider(context.Background(), port.TimeRangeFilter{})
	require.NoError(t, err)
	assert.Len(t, sums, 2)
	assert.Contains(t, sums, domain.ProviderOpenAI)
	assert.Contains(t, sums, domain.ProviderAnthropic)
}

func TestRepository_SumByProvider_ProviderFilter(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderAnthropic, time.Now()))

	filter := port.TimeRangeFilter{Provider: domain.ProviderOpenAI}
	sums, err := repo.SumByProvider(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, sums, 1)
	_, ok := sums[domain.ProviderOpenAI]
	assert.True(t, ok)
}

func TestRepository_SumByProvider_Empty(t *testing.T) {
	repo := memory.NewRepository()
	sums, err := repo.SumByProvider(context.Background(), port.TimeRangeFilter{})
	require.NoError(t, err)
	assert.Empty(t, sums)
}

// ---------------------------------------------------------------------------
// Concurrency safety
// ---------------------------------------------------------------------------

func TestRepository_ConcurrentSaves(t *testing.T) {
	repo := memory.NewRepository()
	const goroutines = 20
	const writesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				u := makeUsage("s1", domain.ProviderOpenAI, time.Now())
				_ = repo.Save(context.Background(), u)
			}
		}()
	}
	wg.Wait()

	results, err := repo.FindBySession(context.Background(), "s1", 0)
	require.NoError(t, err)
	assert.Len(t, results, goroutines*writesPerGoroutine)
}

func TestRepository_ConcurrentReadWrite(t *testing.T) {
	repo := memory.NewRepository()

	// Pre-seed some records
	for i := 0; i < 10; i++ {
		_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = repo.FindBySession(context.Background(), "s1", 0)
		}()
	}

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = repo.Save(context.Background(), makeUsage("s1", domain.ProviderOpenAI, time.Now()))
		}()
	}

	wg.Wait()
	// No panic or data race — success
}
