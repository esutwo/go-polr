package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nnnc-org/go-polr/internal/helpers"
	"github.com/nnnc-org/go-polr/internal/models"
	"github.com/nnnc-org/go-polr/testutil"
)

func newUserSvc(t *testing.T) *UserService {
	t.Helper()
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)
	return NewUserService(db)
}

func TestUserService_Create_Defaults(t *testing.T) {
	s := newUserSvc(t)

	user, err := s.Create(CreateUserInput{
		Username: "alice",
		Password: "P@ssword123!",
		Email:    "alice@example.com",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, models.RoleUser, user.Role, "default role should be user")
	assert.Equal(t, "1", user.Active)
	assert.Equal(t, "60", user.APIQuota)
	assert.NotEmpty(t, user.RecoveryKey)
	assert.NotEqual(t, "P@ssword123!", user.Password, "password must be hashed at rest")
	assert.True(t, helpers.CheckPassword("P@ssword123!", user.Password), "hashed password must verify")
}

func TestUserService_Create_ExplicitAdminRole(t *testing.T) {
	s := newUserSvc(t)

	user, err := s.Create(CreateUserInput{
		Username: "boss",
		Password: "P@ssword123!",
		Email:    "boss@example.com",
		Role:     models.RoleAdmin,
	})
	require.NoError(t, err)
	assert.Equal(t, models.RoleAdmin, user.Role)
}

func TestUserService_Create_DuplicateUsername(t *testing.T) {
	s := newUserSvc(t)

	_, err := s.Create(CreateUserInput{Username: "dupe", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)

	_, err = s.Create(CreateUserInput{Username: "dupe", Password: "P@ssword123!", Email: "b@x.com"})
	assert.ErrorIs(t, err, ErrUsernameTaken)
}

func TestUserService_Create_DuplicateEmail(t *testing.T) {
	s := newUserSvc(t)

	_, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "shared@x.com"})
	require.NoError(t, err)

	_, err = s.Create(CreateUserInput{Username: "bob", Password: "P@ssword123!", Email: "shared@x.com"})
	assert.ErrorIs(t, err, ErrEmailTaken)
}

func TestUserService_Authenticate(t *testing.T) {
	s := newUserSvc(t)
	_, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		u, err := s.Authenticate("alice", "P@ssword123!")
		require.NoError(t, err)
		assert.Equal(t, "alice", u.Username)
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := s.Authenticate("alice", "nope")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("missing user", func(t *testing.T) {
		_, err := s.Authenticate("ghost", "anything")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("inactive user", func(t *testing.T) {
		u, err := s.Create(CreateUserInput{Username: "deactivated", Password: "P@ssword123!", Email: "d@x.com"})
		require.NoError(t, err)
		require.NoError(t, s.DeactivateUser(u.ID))

		_, err = s.Authenticate("deactivated", "P@ssword123!")
		assert.ErrorIs(t, err, ErrUserInactive)
	})
}

func TestUserService_GetByID(t *testing.T) {
	s := newUserSvc(t)
	created, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)

	got, err := s.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice", got.Username)

	_, err = s.GetByID(999999)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_GetByUsername(t *testing.T) {
	s := newUserSvc(t)
	_, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)

	got, err := s.GetByUsername("alice")
	require.NoError(t, err)
	assert.Equal(t, "alice", got.Username)

	_, err = s.GetByUsername("ghost")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_GenerateAPIKey_StoresHash(t *testing.T) {
	s := newUserSvc(t)
	u, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)

	plaintext, err := s.GenerateAPIKey(u.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, plaintext)

	refreshed, err := s.GetByID(u.ID)
	require.NoError(t, err)
	require.NotNil(t, refreshed.APIKey)
	assert.NotEqual(t, plaintext, *refreshed.APIKey, "stored value must not be the plaintext key")
	assert.Equal(t, helpers.HashAPIKey(plaintext), *refreshed.APIKey)
	assert.True(t, refreshed.APIActive)
}

func TestUserService_GetByAPIKey_LooksUpHash(t *testing.T) {
	s := newUserSvc(t)
	u, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)

	plaintext, err := s.GenerateAPIKey(u.ID)
	require.NoError(t, err)

	// GetByAPIKey is called with the (already-hashed) value the caller supplies.
	// Confirm the stored value matches our HashAPIKey(plaintext).
	hashed := helpers.HashAPIKey(plaintext)
	found, err := s.GetByAPIKey(hashed)
	require.NoError(t, err)
	assert.Equal(t, u.ID, found.ID)

	// Plaintext lookup must NOT succeed.
	_, err = s.GetByAPIKey(plaintext)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_DisableAPIAccess(t *testing.T) {
	s := newUserSvc(t)
	u, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)
	_, err = s.GenerateAPIKey(u.ID)
	require.NoError(t, err)

	require.NoError(t, s.DisableAPIAccess(u.ID))

	refreshed, err := s.GetByID(u.ID)
	require.NoError(t, err)
	assert.False(t, refreshed.APIActive)
}

func TestUserService_UpdatePassword(t *testing.T) {
	s := newUserSvc(t)
	u, err := s.Create(CreateUserInput{Username: "alice", Password: "OldP@ss12345!", Email: "a@x.com"})
	require.NoError(t, err)

	require.NoError(t, s.UpdatePassword(u.ID, "NewP@ss12345!"))

	_, err = s.Authenticate("alice", "OldP@ss12345!")
	assert.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = s.Authenticate("alice", "NewP@ss12345!")
	assert.NoError(t, err)
}

func TestUserService_GetAllUsers_Pagination(t *testing.T) {
	s := newUserSvc(t)
	for i := 0; i < 5; i++ {
		_, err := s.Create(CreateUserInput{
			Username: usernameOf(i),
			Password: "P@ssword123!",
			Email:    usernameOf(i) + "@x.com",
		})
		require.NoError(t, err)
	}

	page1, total, err := s.GetAllUsers(2, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, page1, 2)

	page2, _, err := s.GetAllUsers(2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// No overlap between pages.
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestUserService_ActivateDeactivate(t *testing.T) {
	s := newUserSvc(t)
	u, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)

	require.NoError(t, s.DeactivateUser(u.ID))
	r, _ := s.GetByID(u.ID)
	assert.False(t, r.IsActive())

	require.NoError(t, s.ActivateUser(u.ID))
	r, _ = s.GetByID(u.ID)
	assert.True(t, r.IsActive())
}

func TestUserService_DeleteUser_SoftDelete(t *testing.T) {
	s := newUserSvc(t)
	u, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)

	require.NoError(t, s.DeleteUser(u.ID))

	_, err = s.GetByID(u.ID)
	assert.ErrorIs(t, err, ErrUserNotFound, "soft-deleted user should be filtered by GORM")
}

func TestUserService_UpdateUser(t *testing.T) {
	s := newUserSvc(t)
	u1, err := s.Create(CreateUserInput{Username: "alice", Password: "P@ssword123!", Email: "a@x.com"})
	require.NoError(t, err)
	u2, err := s.Create(CreateUserInput{Username: "bob", Password: "P@ssword123!", Email: "b@x.com"})
	require.NoError(t, err)

	t.Run("rename and update", func(t *testing.T) {
		require.NoError(t, s.UpdateUser(u1.ID, UpdateUserInput{
			Username: "alice2",
			Email:    "alice2@x.com",
			Role:     models.RoleAdmin,
			Active:   true,
			APIQuota: "120",
		}))
		got, _ := s.GetByID(u1.ID)
		assert.Equal(t, "alice2", got.Username)
		assert.Equal(t, models.RoleAdmin, got.Role)
		assert.Equal(t, "120", got.APIQuota)
	})

	t.Run("same identity is allowed", func(t *testing.T) {
		// Updating user to its own username/email must not collide with itself.
		err := s.UpdateUser(u1.ID, UpdateUserInput{
			Username: "alice2",
			Email:    "alice2@x.com",
			Role:     models.RoleAdmin,
			Active:   true,
			APIQuota: "120",
		})
		assert.NoError(t, err)
	})

	t.Run("username taken by another user", func(t *testing.T) {
		err := s.UpdateUser(u2.ID, UpdateUserInput{
			Username: "alice2",
			Email:    "b@x.com",
			Role:     models.RoleUser,
			Active:   true,
			APIQuota: "60",
		})
		assert.ErrorIs(t, err, ErrUsernameTaken)
	})

	t.Run("email taken by another user", func(t *testing.T) {
		err := s.UpdateUser(u2.ID, UpdateUserInput{
			Username: "bob",
			Email:    "alice2@x.com",
			Role:     models.RoleUser,
			Active:   true,
			APIQuota: "60",
		})
		assert.ErrorIs(t, err, ErrEmailTaken)
	})

	t.Run("deactivation via update", func(t *testing.T) {
		require.NoError(t, s.UpdateUser(u2.ID, UpdateUserInput{
			Username: "bob",
			Email:    "b@x.com",
			Role:     models.RoleUser,
			Active:   false,
			APIQuota: "60",
		}))
		got, _ := s.GetByID(u2.ID)
		assert.False(t, got.IsActive())
	})
}

func TestUserService_UpdateUserPassword(t *testing.T) {
	s := newUserSvc(t)
	u, err := s.Create(CreateUserInput{Username: "alice", Password: "OldP@ss12345!", Email: "a@x.com"})
	require.NoError(t, err)

	require.NoError(t, s.UpdateUserPassword(u.ID, "AdminSetP@ss123!"))

	_, err = s.Authenticate("alice", "AdminSetP@ss123!")
	assert.NoError(t, err)
}

func usernameOf(i int) string {
	return string(rune('a' + i))
}
