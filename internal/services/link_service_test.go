package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/nnnc-org/go-polr/testutil"
)

func newLinkSvc(t *testing.T) (*LinkService, *gorm.DB) {
	t.Helper()
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)
	return NewLinkService(db), db
}

func TestLinkService_Create_AutoShortCode(t *testing.T) {
	s, _ := newLinkSvc(t)

	res, err := s.Create(CreateLinkInput{
		LongURL: "https://example.com/very-long",
		Creator: "alice",
		IP:      "127.0.0.1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Link.ShortURL)
	assert.Equal(t, 6, len(res.Link.ShortURL), "default short code length")
	assert.False(t, res.Link.IsCustom)
	assert.Empty(t, res.SecretKey)
	assert.NotNil(t, res.Link.LongURLHash)
}

func TestLinkService_Create_CustomEnding(t *testing.T) {
	s, _ := newLinkSvc(t)

	res, err := s.Create(CreateLinkInput{
		LongURL:      "https://example.com",
		CustomEnding: "myCustom",
		Creator:      "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "myCustom", res.Link.ShortURL)
	assert.True(t, res.Link.IsCustom)
}

func TestLinkService_Create_InvalidURL(t *testing.T) {
	s, _ := newLinkSvc(t)

	_, err := s.Create(CreateLinkInput{LongURL: "not-a-url", Creator: "alice"})
	assert.ErrorIs(t, err, ErrInvalidURL)
}

func TestLinkService_Create_InvalidCustomEnding(t *testing.T) {
	s, _ := newLinkSvc(t)

	_, err := s.Create(CreateLinkInput{
		LongURL:      "https://example.com",
		CustomEnding: "has space!",
		Creator:      "alice",
	})
	assert.ErrorIs(t, err, ErrInvalidURL)
}

func TestLinkService_Create_DuplicateCustomEnding(t *testing.T) {
	s, _ := newLinkSvc(t)

	_, err := s.Create(CreateLinkInput{LongURL: "https://a.com", CustomEnding: "taken", Creator: "alice"})
	require.NoError(t, err)

	_, err = s.Create(CreateLinkInput{LongURL: "https://b.com", CustomEnding: "taken", Creator: "bob"})
	assert.ErrorIs(t, err, ErrShortURLTaken)
}

func TestLinkService_Create_Secret(t *testing.T) {
	s, _ := newLinkSvc(t)

	res, err := s.Create(CreateLinkInput{
		LongURL:  "https://example.com",
		IsSecret: true,
		Creator:  "alice",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.SecretKey)
	assert.Equal(t, res.SecretKey, res.Link.SecretKey)
	assert.True(t, res.Link.IsSecret())
}

func TestLinkService_GetByShortURL(t *testing.T) {
	s, _ := newLinkSvc(t)

	res, _ := s.Create(CreateLinkInput{LongURL: "https://example.com", CustomEnding: "abc", Creator: "alice"})

	got, err := s.GetByShortURL("abc")
	require.NoError(t, err)
	assert.Equal(t, res.Link.ID, got.ID)

	_, err = s.GetByShortURL("nonexistent")
	assert.ErrorIs(t, err, ErrLinkNotFound)
}

func TestLinkService_GetByID(t *testing.T) {
	s, _ := newLinkSvc(t)
	res, _ := s.Create(CreateLinkInput{LongURL: "https://example.com", Creator: "alice"})

	got, err := s.GetByID(res.Link.ID)
	require.NoError(t, err)
	assert.Equal(t, res.Link.ShortURL, got.ShortURL)

	_, err = s.GetByID(999999)
	assert.ErrorIs(t, err, ErrLinkNotFound)
}

func TestLinkService_GetLongURL(t *testing.T) {
	s, _ := newLinkSvc(t)

	// Plain link
	_, err := s.Create(CreateLinkInput{LongURL: "https://example.com/plain", CustomEnding: "plain", Creator: "alice"})
	require.NoError(t, err)

	// Secret link
	secret, err := s.Create(CreateLinkInput{LongURL: "https://example.com/secret", CustomEnding: "secret", IsSecret: true, Creator: "alice"})
	require.NoError(t, err)

	// Disabled link
	_, err = s.Create(CreateLinkInput{LongURL: "https://example.com/disabled", CustomEnding: "dis", Creator: "alice"})
	require.NoError(t, err)
	disabled, _ := s.GetByShortURL("dis")
	require.NoError(t, s.DisableLink(disabled.ID))

	t.Run("plain success", func(t *testing.T) {
		got, err := s.GetLongURL("plain", "")
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/plain", got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := s.GetLongURL("ghost", "")
		assert.ErrorIs(t, err, ErrLinkNotFound)
	})

	t.Run("disabled", func(t *testing.T) {
		_, err := s.GetLongURL("dis", "")
		assert.ErrorIs(t, err, ErrLinkDisabled)
	})

	t.Run("secret without key", func(t *testing.T) {
		_, err := s.GetLongURL("secret", "")
		assert.ErrorIs(t, err, ErrAccessDenied)
	})

	t.Run("secret wrong key", func(t *testing.T) {
		_, err := s.GetLongURL("secret", "wrongkey")
		assert.ErrorIs(t, err, ErrAccessDenied)
	})

	t.Run("secret right key", func(t *testing.T) {
		got, err := s.GetLongURL("secret", secret.SecretKey)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/secret", got)
	})
}

func TestLinkService_IncrementClicks(t *testing.T) {
	s, _ := newLinkSvc(t)
	res, _ := s.Create(CreateLinkInput{LongURL: "https://example.com", Creator: "alice"})

	// Sequential increments — verifies the `clicks + 1` UPDATE expression bumps
	// the counter correctly. The atomicity guarantee under concurrency comes
	// from the SQL expression itself (verified against MySQL in integration).
	const n = 10
	for i := 0; i < n; i++ {
		require.NoError(t, s.IncrementClicks(res.Link.ID))
	}

	got, err := s.GetByID(res.Link.ID)
	require.NoError(t, err)
	assert.Equal(t, n, got.Clicks)
}

func TestLinkService_GetLinksByCreator(t *testing.T) {
	s, _ := newLinkSvc(t)

	for i := 0; i < 3; i++ {
		_, err := s.Create(CreateLinkInput{LongURL: "https://example.com", Creator: "alice"})
		require.NoError(t, err)
	}
	_, err := s.Create(CreateLinkInput{LongURL: "https://example.com", Creator: "bob"})
	require.NoError(t, err)

	aliceLinks, total, err := s.GetLinksByCreator("alice", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, aliceLinks, 3)

	all, total, err := s.GetLinksByCreator("", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, all, 4)
}

func TestLinkService_SearchLinks(t *testing.T) {
	s, _ := newLinkSvc(t)

	_, err := s.Create(CreateLinkInput{LongURL: "https://example.com/apple", CustomEnding: "apple", Creator: "alice"})
	require.NoError(t, err)
	_, err = s.Create(CreateLinkInput{LongURL: "https://example.com/banana", CustomEnding: "banana", Creator: "bob"})
	require.NoError(t, err)
	_, err = s.Create(CreateLinkInput{LongURL: "https://example.com/cherry", CustomEnding: "cherry", Creator: "alice"})
	require.NoError(t, err)

	t.Run("filter by creator", func(t *testing.T) {
		links, total, err := s.SearchLinks(LinkSearchParams{Creator: "alice", Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, links, 2)
	})

	t.Run("search by short_url substring", func(t *testing.T) {
		links, total, err := s.SearchLinks(LinkSearchParams{Search: "appl", Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, links, 1)
		assert.Equal(t, "apple", links[0].ShortURL)
	})

	t.Run("search by creator substring", func(t *testing.T) {
		links, total, err := s.SearchLinks(LinkSearchParams{Search: "bob", Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, links, 1)
		assert.Equal(t, "bob", links[0].Creator)
	})

	t.Run("sort ascending by short_url", func(t *testing.T) {
		links, _, err := s.SearchLinks(LinkSearchParams{SortBy: "short_url", SortOrder: "asc", Limit: 10})
		require.NoError(t, err)
		require.Len(t, links, 3)
		assert.Equal(t, "apple", links[0].ShortURL)
		assert.Equal(t, "banana", links[1].ShortURL)
		assert.Equal(t, "cherry", links[2].ShortURL)
	})

	t.Run("invalid sort column falls back to created_at", func(t *testing.T) {
		// SQL injection / unknown column attempts must not error out.
		_, _, err := s.SearchLinks(LinkSearchParams{SortBy: "evil; DROP TABLE links; --", Limit: 10})
		assert.NoError(t, err)
	})
}

func TestLinkService_EnableDisable(t *testing.T) {
	s, _ := newLinkSvc(t)
	res, _ := s.Create(CreateLinkInput{LongURL: "https://example.com", Creator: "alice"})

	require.NoError(t, s.DisableLink(res.Link.ID))
	got, _ := s.GetByID(res.Link.ID)
	assert.True(t, got.IsDisabled)

	require.NoError(t, s.EnableLink(res.Link.ID))
	got, _ = s.GetByID(res.Link.ID)
	assert.False(t, got.IsDisabled)
}

func TestLinkService_DeleteLink(t *testing.T) {
	s, _ := newLinkSvc(t)
	res, _ := s.Create(CreateLinkInput{LongURL: "https://example.com", Creator: "alice"})

	require.NoError(t, s.DeleteLink(res.Link.ID))

	_, err := s.GetByID(res.Link.ID)
	assert.ErrorIs(t, err, ErrLinkNotFound)
}

// Sanity check that helpers.HashURL is being applied so duplicate-detection columns are populated.
func TestLinkService_Create_HashesURL(t *testing.T) {
	s, _ := newLinkSvc(t)
	res, _ := s.Create(CreateLinkInput{LongURL: "https://example.com/abc", Creator: "alice"})
	require.NotNil(t, res.Link.LongURLHash)
	assert.Len(t, *res.Link.LongURLHash, 8, "CRC32 hex string")
}
