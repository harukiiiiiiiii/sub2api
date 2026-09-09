//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type failingGroupResetRepository struct {
	service.UserSubscriptionRepository
	calls int
}

func (r *failingGroupResetRepository) ResetUsageWindows(ctx context.Context, id int64, daily, weekly, monthly bool, dailyStart, periodicStart time.Time) error {
	r.calls++
	if r.calls == 2 {
		return errors.New("injected reset failure")
	}
	return r.UserSubscriptionRepository.ResetUsageWindows(ctx, id, daily, weekly, monthly, dailyStart, periodicStart)
}

func TestSubscriptionGroupQuota_AtomicAndScoped(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	group, err := client.Group.Create().SetName(fmt.Sprintf("reset-group-%d", time.Now().UnixNano())).SetSubscriptionType("subscription").Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Group.DeleteOneID(group.ID).Exec(mixins.SkipSoftDelete(ctx))) })
	other, err := client.Group.Create().SetName(fmt.Sprintf("other-reset-group-%d", time.Now().UnixNano())).SetSubscriptionType("subscription").Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Group.DeleteOneID(other.ID).Exec(mixins.SkipSoftDelete(ctx))) })
	repo := NewUserSubscriptionRepository(client)
	var subs []*service.UserSubscription
	for i := 0; i < 4; i++ {
		userID := mustCreateUserForQuota(t, client)
		t.Cleanup(func() { require.NoError(t, client.User.DeleteOneID(userID).Exec(mixins.SkipSoftDelete(ctx))) })
		groupID := group.ID
		if i == 2 {
			groupID = other.ID
		}
		sub := &service.UserSubscription{UserID: userID, GroupID: groupID, StartsAt: time.Now().Add(-time.Hour), ExpiresAt: time.Now().Add(24 * time.Hour), Status: service.SubscriptionStatusActive, DailyUsageUSD: 10, WeeklyUsageUSD: 20, MonthlyUsageUSD: 30}
		require.NoError(t, repo.Create(ctx, sub))
		t.Cleanup(func() {
			require.NoError(t, client.UserSubscription.DeleteOneID(sub.ID).Exec(mixins.SkipSoftDelete(ctx)))
		})
		stored, err := repo.GetByID(ctx, sub.ID)
		require.NoError(t, err)
		subs = append(subs, stored)
	}
	require.NoError(t, repo.Delete(ctx, subs[3].ID)) // Revoked subscriptions are excluded.
	groupRepo := NewGroupRepository(client, integrationDB)
	failing := &failingGroupResetRepository{UserSubscriptionRepository: repo}
	svc := service.NewSubscriptionService(groupRepo, failing, nil, client, nil)
	_, err = svc.AdminResetGroupQuota(ctx, group.ID, true, true, true)
	require.ErrorContains(t, err, "injected reset failure")
	for _, sub := range subs[:3] {
		stored, err := repo.GetByID(ctx, sub.ID)
		require.NoError(t, err)
		require.Equal(t, 20.0, stored.WeeklyUsageUSD, "failed batch must roll back earlier updates")
	}
	svc = service.NewSubscriptionService(groupRepo, repo, nil, client, nil)
	result, err := svc.AdminResetGroupQuota(ctx, group.ID, false, true, false)
	require.NoError(t, err)
	require.Equal(t, 2, result.ResetCount)
	for i, sub := range subs {
		stored, err := repo.GetByIDIncludeDeleted(ctx, sub.ID)
		require.NoError(t, err)
		if i < 2 {
			require.Zero(t, stored.WeeklyUsageUSD)
		} else {
			require.Equal(t, 20.0, stored.WeeklyUsageUSD)
		}
		require.Equal(t, 10.0, stored.DailyUsageUSD)
		require.Equal(t, 30.0, stored.MonthlyUsageUSD)
		require.True(t, stored.StartsAt.Equal(sub.StartsAt))
		require.True(t, stored.ExpiresAt.Equal(sub.ExpiresAt))
		require.Equal(t, sub.Status, stored.Status)
	}
}
