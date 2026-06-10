package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/nnnc-org/go-polr/internal/models"
	"github.com/nnnc-org/go-polr/testutil"
)

func newClickSvc(t *testing.T) (*ClickService, *gorm.DB) {
	t.Helper()
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)
	return NewClickService(db), db
}

func makeLink(t *testing.T, db *gorm.DB, shortURL string) *models.Link {
	t.Helper()
	l, err := testutil.CreateTestLink(db, shortURL, "https://example.com", "alice")
	require.NoError(t, err)
	return l
}

func TestClickService_RecordClick_ExtractsRefererHost(t *testing.T) {
	s, db := newClickSvc(t)
	link := makeLink(t, db, "abc")

	require.NoError(t, s.RecordClick(RecordClickInput{
		LinkID:    link.ID,
		IP:        "1.2.3.4",
		Referer:   "https://news.example.com/path?x=1",
		UserAgent: "Mozilla/5.0",
	}))

	clicks, total, err := s.GetClicksByLinkID(link.ID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	c := clicks[0]
	require.NotNil(t, c.Referer)
	assert.Equal(t, "https://news.example.com/path?x=1", *c.Referer)
	require.NotNil(t, c.RefererHost)
	assert.Equal(t, "news.example.com", *c.RefererHost)
	require.NotNil(t, c.UserAgent)
	assert.Equal(t, "Mozilla/5.0", *c.UserAgent)
}

func TestClickService_RecordClick_EmptyReferer(t *testing.T) {
	s, db := newClickSvc(t)
	link := makeLink(t, db, "abc")

	require.NoError(t, s.RecordClick(RecordClickInput{
		LinkID: link.ID,
		IP:     "1.2.3.4",
	}))

	clicks, _, err := s.GetClicksByLinkID(link.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, clicks, 1)
	assert.Nil(t, clicks[0].Referer)
	assert.Nil(t, clicks[0].RefererHost)
	assert.Nil(t, clicks[0].UserAgent)
}

func TestClickService_RecordClick_OverlongReferer_NotStored(t *testing.T) {
	s, db := newClickSvc(t)
	link := makeLink(t, db, "abc")

	// >2048 chars: per maxRefererLength, must be dropped (not stored at all).
	big := "https://example.com/" + strings.Repeat("a", 3000)
	require.NoError(t, s.RecordClick(RecordClickInput{
		LinkID:  link.ID,
		IP:      "1.2.3.4",
		Referer: big,
	}))

	clicks, _, _ := s.GetClicksByLinkID(link.ID, 10, 0)
	require.Len(t, clicks, 1)
	assert.Nil(t, clicks[0].Referer, "overlong referer must be dropped")
	assert.Nil(t, clicks[0].RefererHost)
}

func TestClickService_RecordClick_OverlongUserAgent_NotStored(t *testing.T) {
	s, db := newClickSvc(t)
	link := makeLink(t, db, "abc")

	big := strings.Repeat("U", 2000)
	require.NoError(t, s.RecordClick(RecordClickInput{
		LinkID:    link.ID,
		IP:        "1.2.3.4",
		UserAgent: big,
	}))

	clicks, _, _ := s.GetClicksByLinkID(link.ID, 10, 0)
	require.Len(t, clicks, 1)
	assert.Nil(t, clicks[0].UserAgent, "overlong user-agent must be dropped")
}

func TestClickService_RecordClick_MalformedReferer_NoHost(t *testing.T) {
	s, db := newClickSvc(t)
	link := makeLink(t, db, "abc")

	// Parse succeeds but Host is empty → RefererHost stays nil; Referer is still stored.
	require.NoError(t, s.RecordClick(RecordClickInput{
		LinkID:  link.ID,
		IP:      "1.2.3.4",
		Referer: "/relative/path",
	}))

	clicks, _, _ := s.GetClicksByLinkID(link.ID, 10, 0)
	require.Len(t, clicks, 1)
	require.NotNil(t, clicks[0].Referer)
	assert.Nil(t, clicks[0].RefererHost)
}

func TestClickService_GetClicksByLinkID_PaginationOrdering(t *testing.T) {
	s, db := newClickSvc(t)
	link := makeLink(t, db, "abc")

	for i := 0; i < 5; i++ {
		require.NoError(t, s.RecordClick(RecordClickInput{LinkID: link.ID, IP: "1.1.1.1"}))
	}

	page1, total, err := s.GetClicksByLinkID(link.ID, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, page1, 2)

	page2, _, err := s.GetClicksByLinkID(link.ID, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Ordered DESC by created_at → newer rows first, so later-inserted rows come earlier.
	assert.True(t, page1[0].ID >= page2[0].ID, "page 1 should contain newer clicks")
}

func TestClickService_GetRecentClicks(t *testing.T) {
	s, db := newClickSvc(t)
	link1 := makeLink(t, db, "abc")
	link2 := makeLink(t, db, "xyz")

	require.NoError(t, s.RecordClick(RecordClickInput{LinkID: link1.ID, IP: "1.1.1.1"}))
	require.NoError(t, s.RecordClick(RecordClickInput{LinkID: link2.ID, IP: "2.2.2.2"}))

	clicks, err := s.GetRecentClicks(10)
	require.NoError(t, err)
	require.Len(t, clicks, 2)

	// Preload should populate the Link relation.
	assert.NotZero(t, clicks[0].Link.ID, "preload should populate associated Link")
}
