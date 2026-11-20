//go:build integration

package mongo

import (
	"context"
	stderrors "errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const integrationCheckRetention = 30 * 24 * time.Hour

func integrationDatabase(t *testing.T) (*driver.Database, context.Context) {
	t.Helper()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		t.Skip("MONGO_URI is unset; MongoDB integration tests were skipped")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	client, err := driver.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		cancel()
		t.Fatalf("connect to MongoDB: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_ = client.Disconnect(cleanupCtx)
		cancel()
		t.Skipf("MongoDB is unavailable; integration tests were skipped: %v", err)
	}

	database := client.Database("vigil_integration_" + bson.NewObjectID().Hex())
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := database.Drop(cleanupCtx); err != nil {
			t.Errorf("drop integration database: %v", err)
		}
		if err := client.Disconnect(cleanupCtx); err != nil {
			t.Errorf("disconnect MongoDB client: %v", err)
		}
		cancel()
	})
	if err := CreateIndexes(ctx, database, integrationCheckRetention); err != nil {
		t.Fatalf("create integration indexes: %v", err)
	}
	return database, ctx
}

func TestIntegrationUserSessionAndPasswordResetRepositories(t *testing.T) {
	database, ctx := integrationDatabase(t)
	users := NewUserRepository(database)
	sessions := NewSessionRepository(database)
	resets := NewPasswordResetRepository(database)

	user := &domain.User{
		Name:       "Integration User",
		Email:      "integration@example.test",
		StatusSlug: integrationSlug(),
		GoogleSub:  "integration-google-sub",
		CreatedAt:  time.Now().UTC(),
	}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	for name, lookup := range map[string]func() (*domain.User, error){
		"id":    func() (*domain.User, error) { return users.GetByID(ctx, user.ID) },
		"email": func() (*domain.User, error) { return users.GetByEmail(ctx, user.Email) },
		"slug":  func() (*domain.User, error) { return users.GetBySlug(ctx, user.StatusSlug) },
		"google subject": func() (*domain.User, error) {
			return users.GetByGoogleSub(ctx, user.GoogleSub)
		},
	} {
		got, err := lookup()
		if err != nil || got.ID != user.ID {
			t.Fatalf("get user by %s = (%v, %v), want ID %s", name, got, err, user.ID)
		}
	}
	user.Name = "Updated Integration User"
	user.Notify.EmailAlerts = true
	if err := users.Update(ctx, user); err != nil {
		t.Fatalf("update user: %v", err)
	}
	updated, err := users.GetByID(ctx, user.ID)
	if err != nil || updated.Name != user.Name || !updated.Notify.EmailAlerts {
		t.Fatalf("updated user = (%v, %v), want updated fields", updated, err)
	}

	firstSession := &domain.Session{UserID: user.ID, TokenHash: "session-token-1", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	keepSession := &domain.Session{UserID: user.ID, TokenHash: "session-token-2", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	for _, session := range []*domain.Session{firstSession, keepSession} {
		if err := sessions.Create(ctx, session); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
	if got, err := sessions.GetByTokenHash(ctx, keepSession.TokenHash); err != nil || got.ID != keepSession.ID {
		t.Fatalf("get session = (%v, %v), want ID %s", got, err, keepSession.ID)
	}
	if err := sessions.DeleteByUserExcept(ctx, user.ID, keepSession.ID); err != nil {
		t.Fatalf("delete other sessions: %v", err)
	}
	if _, err := sessions.GetByTokenHash(ctx, firstSession.TokenHash); !stderrors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("deleted session lookup error = %v, want ErrNotFound", err)
	}
	if err := sessions.Delete(ctx, keepSession.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	resetOne := &domain.PasswordReset{UserID: user.ID, TokenHash: "reset-token-1", ExpiresAt: time.Now().Add(time.Hour)}
	resetTwo := &domain.PasswordReset{UserID: user.ID, TokenHash: "reset-token-2", ExpiresAt: time.Now().Add(time.Hour)}
	for _, reset := range []*domain.PasswordReset{resetOne, resetTwo} {
		if err := resets.Create(ctx, reset); err != nil {
			t.Fatalf("create password reset: %v", err)
		}
	}
	if got, err := resets.GetByTokenHash(ctx, resetOne.TokenHash); err != nil || got.ID != resetOne.ID {
		t.Fatalf("get password reset = (%v, %v), want ID %s", got, err, resetOne.ID)
	}
	if err := resets.DeleteByUser(ctx, user.ID); err != nil {
		t.Fatalf("delete password resets: %v", err)
	}
	if _, err := resets.GetByTokenHash(ctx, resetOne.TokenHash); !stderrors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("deleted password reset lookup error = %v, want ErrNotFound", err)
	}
}

func TestIntegrationUniqueConstraints(t *testing.T) {
	database, ctx := integrationDatabase(t)
	users := NewUserRepository(database)
	firstUser := &domain.User{Email: "same@example.test", StatusSlug: integrationSlug(), GoogleSub: "same-google-sub"}
	if err := users.Create(ctx, firstUser); err != nil {
		t.Fatalf("create first user: %v", err)
	}

	duplicateEmail := &domain.User{Email: firstUser.Email, StatusSlug: integrationSlug()}
	if err := users.Create(ctx, duplicateEmail); !stderrors.Is(err, apperrors.ErrEmailTaken) {
		t.Fatalf("duplicate email error = %v, want ErrEmailTaken", err)
	}
	duplicateSlug := &domain.User{Email: "other@example.test", StatusSlug: firstUser.StatusSlug}
	if err := users.Create(ctx, duplicateSlug); !stderrors.Is(err, apperrors.ErrSlugTaken) {
		t.Fatalf("duplicate slug error = %v, want ErrSlugTaken", err)
	}
	duplicateGoogleSub := &domain.User{Email: "google@example.test", StatusSlug: integrationSlug(), GoogleSub: firstUser.GoogleSub}
	if err := users.Create(ctx, duplicateGoogleSub); !driver.IsDuplicateKeyError(err) {
		t.Fatalf("duplicate Google subject error = %v, want duplicate key", err)
	}

	sessionRepo := NewSessionRepository(database)
	session := &domain.Session{UserID: firstUser.ID, TokenHash: "unique-session-token", ExpiresAt: time.Now().Add(time.Hour)}
	if err := sessionRepo.Create(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	duplicateSession := &domain.Session{UserID: firstUser.ID, TokenHash: session.TokenHash, ExpiresAt: time.Now().Add(time.Hour)}
	if err := sessionRepo.Create(ctx, duplicateSession); !driver.IsDuplicateKeyError(err) {
		t.Fatalf("duplicate session token error = %v, want duplicate key", err)
	}

	resetRepo := NewPasswordResetRepository(database)
	resetOne := &domain.PasswordReset{UserID: firstUser.ID, TokenHash: "unique-reset-token", ExpiresAt: time.Now().Add(time.Hour)}
	resetTwo := &domain.PasswordReset{UserID: firstUser.ID, TokenHash: resetOne.TokenHash, ExpiresAt: time.Now().Add(time.Hour)}
	if err := resetRepo.Create(ctx, resetOne); err != nil {
		t.Fatalf("create first reset: %v", err)
	}
	if err := resetRepo.Create(ctx, resetTwo); !driver.IsDuplicateKeyError(err) {
		t.Fatalf("duplicate reset token error = %v, want duplicate key", err)
	}
}

func TestIntegrationRepositoryIndexes(t *testing.T) {
	database, ctx := integrationDatabase(t)
	for _, test := range []struct {
		collection string
		name       string
		unique     bool
		ttlSeconds int64
	}{
		{collection: "users", name: "users_email_unique", unique: true},
		{collection: "users", name: "users_status_slug_unique", unique: true},
		{collection: "sessions", name: "sessions_token_hash_unique", unique: true},
		{collection: "sessions", name: "sessions_expires_at_ttl", ttlSeconds: 0},
		{collection: "password_resets", name: "password_resets_token_hash_unique", unique: true},
		{collection: "password_resets", name: "password_resets_expires_at_ttl", ttlSeconds: 0},
		{collection: "monitors", name: "monitors_due_lease"},
		{collection: "checks", name: "checks_monitor_checked_at"},
		{collection: "checks", name: "checks_checked_at_ttl", ttlSeconds: int64(integrationCheckRetention / time.Second)},
		{collection: "incidents", name: "incidents_monitor_started_at"},
		{collection: "incidents", name: "incidents_one_open_per_monitor", unique: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			indexes := listIntegrationIndexes(t, ctx, database.Collection(test.collection))
			index, ok := indexes[test.name]
			if !ok {
				t.Fatalf("index %q missing from %s", test.name, test.collection)
			}
			if test.unique && indexBool(index, "unique") != true {
				t.Fatalf("index %q unique = %v, want true", test.name, index["unique"])
			}
			if test.name == "incidents_one_open_per_monitor" {
				if _, ok := index["partialFilterExpression"]; !ok {
					t.Fatalf("index %q is missing its partial filter", test.name)
				}
			}
			if !test.unique && test.name == "checks_checked_at_ttl" {
				if got := indexInt64(index, "expireAfterSeconds"); got != test.ttlSeconds {
					t.Fatalf("index %q TTL = %d, want %d", test.name, got, test.ttlSeconds)
				}
			}
			if test.ttlSeconds == 0 && (test.name == "sessions_expires_at_ttl" || test.name == "password_resets_expires_at_ttl") {
				if got := indexInt64(index, "expireAfterSeconds"); got != 0 {
					t.Fatalf("index %q TTL = %d, want 0", test.name, got)
				}
			}
		})
	}
}

func TestIntegrationMonitorCheckAndIncidentRepositories(t *testing.T) {
	database, ctx := integrationDatabase(t)
	ownerID := domain.ID(bson.NewObjectID().Hex())
	monitorRepo := NewMonitorRepository(database)
	checkRepo := NewCheckRepository(database)
	incidentRepo := NewIncidentRepository(database)
	now := time.Now().UTC().Truncate(time.Millisecond)

	monitorOne := integrationMonitor(ownerID, now, false)
	monitorTwo := integrationMonitor(ownerID, now, true)
	for _, monitor := range []*domain.Monitor{monitorOne, monitorTwo} {
		if err := monitorRepo.Create(ctx, monitor); err != nil {
			t.Fatalf("create monitor: %v", err)
		}
	}
	if got, err := monitorRepo.GetByID(ctx, monitorOne.ID); err != nil || got.Name != monitorOne.Name {
		t.Fatalf("get monitor = (%v, %v), want %q", got, err, monitorOne.Name)
	}
	if got, err := monitorRepo.CountByOwner(ctx, ownerID); err != nil || got != 2 {
		t.Fatalf("count monitors = (%d, %v), want 2", got, err)
	}
	if got, err := monitorRepo.ListByOwner(ctx, ownerID); err != nil || len(got) != 2 {
		t.Fatalf("list monitors = (%d, %v), want 2", len(got), err)
	}
	if got, err := monitorRepo.ListPublicByOwner(ctx, ownerID); err != nil || len(got) != 1 || got[0].ID != monitorTwo.ID {
		t.Fatalf("list public monitors = (%v, %v), want only %s", got, err, monitorTwo.ID)
	}
	monitorTwo.Name = "Updated monitor"
	monitorTwo.Public = false
	if err := monitorRepo.Update(ctx, monitorTwo); err != nil {
		t.Fatalf("update monitor: %v", err)
	}
	updatedMonitor, err := monitorRepo.GetByID(ctx, monitorTwo.ID)
	if err != nil || updatedMonitor.Name != monitorTwo.Name || updatedMonitor.Public {
		t.Fatalf("updated monitor = (%v, %v), want updated name and private status", updatedMonitor, err)
	}
	if err := monitorRepo.Delete(ctx, monitorTwo.ID); err != nil {
		t.Fatalf("delete monitor: %v", err)
	}
	if err := monitorRepo.Delete(ctx, monitorTwo.ID); !stderrors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("delete missing monitor error = %v, want ErrNotFound", err)
	}

	for _, check := range []domain.Check{
		{MonitorID: monitorOne.ID, CheckedAt: now.Add(-time.Hour), Up: true, Latency: 100 * time.Millisecond},
		{MonitorID: monitorOne.ID, CheckedAt: now.Add(-30 * time.Minute), Up: false, Latency: 200 * time.Millisecond, Detail: "HTTP 503"},
		{MonitorID: monitorOne.ID, CheckedAt: now.Add(-15 * time.Minute), Up: true, Latency: 150 * time.Millisecond},
	} {
		if err := checkRepo.Insert(ctx, &check); err != nil {
			t.Fatalf("insert check: %v", err)
		}
	}
	checks, err := checkRepo.ListByMonitor(ctx, monitorOne.ID, 2)
	if err != nil || len(checks) != 2 || !checks[0].CheckedAt.After(checks[1].CheckedAt) {
		t.Fatalf("list checks = (%v, %v), want newest two first", checks, err)
	}
	gotUptime, err := checkRepo.UptimeSince(ctx, monitorOne.ID, now.Add(-45*time.Minute))
	if err != nil || gotUptime != (domain.Uptime{Up: 1, Total: 2}) {
		t.Fatalf("uptime since = (%+v, %v), want {Up:1 Total:2}", gotUptime, err)
	}
	if err := checkRepo.DeleteByMonitor(ctx, monitorOne.ID); err != nil {
		t.Fatalf("delete checks: %v", err)
	}
	if checks, err := checkRepo.ListByMonitor(ctx, monitorOne.ID, 10); err != nil || len(checks) != 0 {
		t.Fatalf("checks after delete = (%v, %v), want empty", checks, err)
	}

	incident := &domain.Incident{MonitorID: monitorOne.ID, StartedAt: now.Add(-time.Hour), Cause: "HTTP 503"}
	if err := incidentRepo.Open(ctx, incident); err != nil {
		t.Fatalf("open incident: %v", err)
	}
	duplicate := &domain.Incident{MonitorID: monitorOne.ID, StartedAt: now, Cause: "still down"}
	if err := incidentRepo.Open(ctx, duplicate); !driver.IsDuplicateKeyError(err) {
		t.Fatalf("second open incident error = %v, want duplicate key", err)
	}
	if err := incidentRepo.Resolve(ctx, monitorOne.ID, now); err != nil {
		t.Fatalf("resolve incident: %v", err)
	}
	incidents, err := incidentRepo.ListByMonitor(ctx, monitorOne.ID, 10)
	if err != nil || len(incidents) != 1 || !incidents[0].ResolvedAt.Equal(now) {
		t.Fatalf("list incidents = (%v, %v), want resolved incident", incidents, err)
	}
	if err := incidentRepo.Open(ctx, duplicate); err != nil {
		t.Fatalf("open incident after resolution: %v", err)
	}
	if err := incidentRepo.DeleteByMonitor(ctx, monitorOne.ID); err != nil {
		t.Fatalf("delete incidents: %v", err)
	}
	if incidents, err := incidentRepo.ListByMonitor(ctx, monitorOne.ID, 10); err != nil || len(incidents) != 0 {
		t.Fatalf("incidents after delete = (%v, %v), want empty", incidents, err)
	}
}

func TestIntegrationConcurrentMonitorClaims(t *testing.T) {
	database, ctx := integrationDatabase(t)
	const count = 8
	ownerID := domain.ID(bson.NewObjectID().Hex())
	now := time.Now().UTC().Truncate(time.Millisecond)
	repository := NewMonitorRepository(database)
	for range count {
		if err := repository.Create(ctx, integrationMonitor(ownerID, now.Add(-time.Second), true)); err != nil {
			t.Fatalf("create due monitor: %v", err)
		}
	}

	type claimResult struct {
		claim *domain.MonitorClaim
		err   error
	}
	results := make(chan claimResult, count)
	var wait sync.WaitGroup
	for range count {
		wait.Add(1)
		go func() {
			defer wait.Done()
			claim, err := repository.ClaimDue(ctx, now)
			results <- claimResult{claim: claim, err: err}
		}()
	}
	wait.Wait()
	close(results)

	claimedIDs := make(map[domain.ID]bool, count)
	claimTokens := make(map[string]bool, count)
	for result := range results {
		if result.err != nil {
			t.Fatalf("claim due monitor: %v", result.err)
		}
		if claimedIDs[result.claim.Monitor.ID] {
			t.Fatalf("monitor %s was claimed more than once", result.claim.Monitor.ID)
		}
		if claimTokens[result.claim.ClaimToken] {
			t.Fatalf("claim token %q was reused", result.claim.ClaimToken)
		}
		claimedIDs[result.claim.Monitor.ID] = true
		claimTokens[result.claim.ClaimToken] = true
	}
	if len(claimedIDs) != count {
		t.Fatalf("unique claims = %d, want %d", len(claimedIDs), count)
	}
}

func integrationMonitor(owner domain.ID, nextCheckAt time.Time, enabled bool) *domain.Monitor {
	return &domain.Monitor{
		OwnerID:     owner,
		Name:        "Integration monitor " + bson.NewObjectID().Hex(),
		Type:        domain.MonitorHTTP,
		Target:      "https://example.test/health",
		Interval:    time.Minute,
		Timeout:     5 * time.Second,
		Public:      enabled,
		Enabled:     enabled,
		Status:      domain.StatusUnknown,
		NextCheckAt: nextCheckAt,
		CreatedAt:   time.Now().UTC(),
	}
}

func integrationSlug() string {
	return "it-" + bson.NewObjectID().Hex()
}

func listIntegrationIndexes(t *testing.T, ctx context.Context, collection *driver.Collection) map[string]bson.M {
	t.Helper()
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		t.Fatalf("list indexes for %s: %v", collection.Name(), err)
	}
	defer cursor.Close(ctx)
	var documents []bson.M
	if err := cursor.All(ctx, &documents); err != nil {
		t.Fatalf("decode indexes for %s: %v", collection.Name(), err)
	}
	indexes := make(map[string]bson.M, len(documents))
	for _, document := range documents {
		if name, ok := document["name"].(string); ok {
			indexes[name] = document
		}
	}
	return indexes
}

func indexBool(index bson.M, field string) bool {
	value, _ := index[field].(bool)
	return value
}

func indexInt64(index bson.M, field string) int64 {
	switch value := index[field].(type) {
	case int32:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return -1
	}
}
