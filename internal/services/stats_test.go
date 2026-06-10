package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/nnnc-org/go-polr/internal/models"
	"github.com/nnnc-org/go-polr/testutil"
)

func newStatsSvc(t *testing.T) (*StatsService, *gorm.DB) {
	t.Helper()
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)
	return NewStatsService(db), db
}

// insertClick is a low-level helper that lets us control the created_at and
// optional country field directly, since the click service overrides timing.
func insertClick(t *testing.T, db *gorm.DB, linkID uint, ip string, refererHost, country *string, createdAt time.Time) {
	t.Helper()
	c := &models.Click{
		LinkID:      linkID,
		IP:          ip,
		RefererHost: refererHost,
		Country:     country,
	}
	require.NoError(t, db.Create(c).Error)
	require.NoError(t, db.Model(c).Update("created_at", createdAt).Error)
}

func TestStatsService_GetLinkStats(t *testing.T) {
	s, db := newStatsSvc(t)
	link, _ := testutil.CreateTestLink(db, "abc", "https://example.com", "alice")

	now := time.Now()
	insertClick(t, db, link.ID, "1.1.1.1", nil, nil, now)
	insertClick(t, db, link.ID, "1.1.1.1", nil, nil, now) // dupe IP
	insertClick(t, db, link.ID, "2.2.2.2", nil, nil, now)

	stats, err := s.GetLinkStats(link.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalClicks)
	assert.Equal(t, int64(2), stats.UniqueClicks, "unique by IP")
}

func TestStatsService_GetDayStats_Bucketing(t *testing.T) {
	s, db := newStatsSvc(t)
	link, _ := testutil.CreateTestLink(db, "abc", "https://example.com", "alice")

	day1 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 2, 8, 0, 0, 0, time.UTC)

	insertClick(t, db, link.ID, "1.1.1.1", nil, nil, day1)
	insertClick(t, db, link.ID, "1.1.1.1", nil, nil, day1.Add(2*time.Hour))
	insertClick(t, db, link.ID, "2.2.2.2", nil, nil, day2)

	stats, err := s.GetDayStats(link.ID,
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, stats, 2)

	// Build a map for assertion (order is ASC but driver date formatting may vary).
	counts := map[string]int64{}
	for _, s := range stats {
		counts[s.Date] = s.Count
	}
	assert.Equal(t, int64(2), counts["2026-06-01"])
	assert.Equal(t, int64(1), counts["2026-06-02"])
}

func TestStatsService_GetDayStats_RangeFilter(t *testing.T) {
	s, db := newStatsSvc(t)
	link, _ := testutil.CreateTestLink(db, "abc", "https://example.com", "alice")

	insertClick(t, db, link.ID, "1.1.1.1", nil, nil, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	insertClick(t, db, link.ID, "1.1.1.1", nil, nil, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))

	stats, err := s.GetDayStats(link.ID,
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, stats, 1, "click outside the range must be excluded")
}

func TestStatsService_GetCountryStats_CoalescesNull(t *testing.T) {
	s, db := newStatsSvc(t)
	link, _ := testutil.CreateTestLink(db, "abc", "https://example.com", "alice")

	us := "US"
	now := time.Now()
	insertClick(t, db, link.ID, "1.1.1.1", nil, &us, now)
	insertClick(t, db, link.ID, "2.2.2.2", nil, &us, now)
	insertClick(t, db, link.ID, "3.3.3.3", nil, nil, now) // NULL country

	stats, err := s.GetCountryStats(link.ID, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)

	got := map[string]int64{}
	for _, s := range stats {
		got[s.Country] = s.Count
	}
	assert.Equal(t, int64(2), got["US"])
	assert.Equal(t, int64(1), got["Unknown"], "NULL country must be coalesced to 'Unknown'")
}

func TestStatsService_GetRefererStats_CoalescesAndLimits(t *testing.T) {
	s, db := newStatsSvc(t)
	link, _ := testutil.CreateTestLink(db, "abc", "https://example.com", "alice")

	news := "news.example.com"
	now := time.Now()
	insertClick(t, db, link.ID, "1.1.1.1", &news, nil, now)
	insertClick(t, db, link.ID, "2.2.2.2", &news, nil, now)
	insertClick(t, db, link.ID, "3.3.3.3", nil, nil, now) // direct

	stats, err := s.GetRefererStats(link.ID, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)

	got := map[string]int64{}
	for _, s := range stats {
		got[s.RefererHost] = s.Count
	}
	assert.Equal(t, int64(2), got["news.example.com"])
	assert.Equal(t, int64(1), got["Direct"], "NULL referer_host must be coalesced to 'Direct'")
}

func TestStatsService_GetDashboardStats(t *testing.T) {
	s, db := newStatsSvc(t)

	// Two users
	_, err := testutil.CreateTestUser(db, "alice", "p", models.RoleUser)
	require.NoError(t, err)
	_, err = testutil.CreateTestUser(db, "bob", "p", models.RoleUser)
	require.NoError(t, err)

	// Two links: one today, one in the past
	linkToday, _ := testutil.CreateTestLink(db, "today", "https://example.com", "alice")
	linkOld, _ := testutil.CreateTestLink(db, "old", "https://example.com", "alice")
	require.NoError(t, db.Model(linkOld).Update("created_at", time.Now().Add(-48*time.Hour)).Error)

	now := time.Now()
	insertClick(t, db, linkToday.ID, "1.1.1.1", nil, nil, now)
	insertClick(t, db, linkOld.ID, "2.2.2.2", nil, nil, now.Add(-48*time.Hour))

	stats, err := s.GetDashboardStats()
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalUsers)
	assert.Equal(t, int64(2), stats.TotalLinks)
	assert.Equal(t, int64(2), stats.TotalClicks)
	assert.Equal(t, int64(1), stats.LinksToday, "only today's link counts")
	assert.Equal(t, int64(1), stats.ClicksToday, "only today's click counts")
}

func TestStatsService_GetUserStats_JoinedOnCreator(t *testing.T) {
	s, db := newStatsSvc(t)

	aliceLink, _ := testutil.CreateTestLink(db, "a1", "https://example.com", "alice")
	bobLink, _ := testutil.CreateTestLink(db, "b1", "https://example.com", "bob")

	now := time.Now()
	insertClick(t, db, aliceLink.ID, "1.1.1.1", nil, nil, now)
	insertClick(t, db, aliceLink.ID, "2.2.2.2", nil, nil, now)
	insertClick(t, db, bobLink.ID, "3.3.3.3", nil, nil, now)

	stats, err := s.GetUserStats("alice")
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalLinks)
	assert.Equal(t, int64(2), stats.TotalClicks, "clicks must be joined via links.creator, excluding bob's")
}
