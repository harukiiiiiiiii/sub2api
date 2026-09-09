//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

type resetGroupRepo struct {
	groupRepoNoop
	group *Group
	err   error
}

func (r resetGroupRepo) GetByID(context.Context, int64) (*Group, error) { return r.group, r.err }

type resetGroupSubs struct {
	userSubRepoNoop
	subs   []UserSubscription
	calls  []int64
	failID int64
	pages  []int
}

func (r *resetGroupSubs) ListByGroupID(_ context.Context, groupID int64, p pagination.PaginationParams) ([]UserSubscription, *pagination.PaginationResult, error) {
	r.pages = append(r.pages, p.Page)
	var matches []UserSubscription
	for _, sub := range r.subs {
		if sub.GroupID == groupID {
			matches = append(matches, sub)
		}
	}
	start := p.Offset()
	end := min(start+p.Limit(), len(matches))
	return matches[start:end], &pagination.PaginationResult{Pages: (len(matches) + p.Limit() - 1) / p.Limit()}, nil
}

func (r *resetGroupSubs) ResetUsageWindows(_ context.Context, id int64, daily, weekly, monthly bool, dailyStart, periodicStart time.Time) error {
	if id == r.failID {
		return errors.New("reset failed")
	}
	r.calls = append(r.calls, id)
	for i := range r.subs {
		sub := &r.subs[i]
		if sub.ID != id {
			continue
		}
		if daily {
			sub.DailyUsageUSD = 0
			sub.DailyWindowStart = &dailyStart
		}
		if weekly {
			sub.WeeklyUsageUSD = 0
			sub.WeeklyWindowStart = &periodicStart
		}
		if monthly {
			sub.MonthlyUsageUSD = 0
			sub.MonthlyWindowStart = &periodicStart
		}
	}
	return nil
}

func TestAdminResetGroupQuota_AllPagesSelectedWindowsAndDates(t *testing.T) {
	now := time.Date(2026, 9, 9, 15, 0, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)
	repo := &resetGroupSubs{}
	for i := int64(1); i <= 501; i++ {
		repo.subs = append(repo.subs, UserSubscription{ID: i, UserID: i, GroupID: 7, DailyUsageUSD: 12, WeeklyUsageUSD: 25, MonthlyUsageUSD: 40, StartsAt: now.Add(-time.Hour), ExpiresAt: expires, Status: SubscriptionStatusActive})
	}
	repo.subs = append(repo.subs, UserSubscription{ID: 999, GroupID: 8, WeeklyUsageUSD: 100})
	svc := &SubscriptionService{groupRepo: resetGroupRepo{group: &Group{ID: 7, SubscriptionType: "subscription"}}, userSubRepo: repo, now: func() time.Time { return now }}
	result, err := svc.AdminResetGroupQuota(context.Background(), 7, true, true, false)
	require.NoError(t, err)
	require.Equal(t, 501, result.ResetCount)
	require.Equal(t, []int{1, 2}, repo.pages)
	for _, sub := range repo.subs[:501] {
		require.Zero(t, sub.DailyUsageUSD)
		require.Zero(t, sub.WeeklyUsageUSD)
		require.Equal(t, 40.0, sub.MonthlyUsageUSD)
		require.Equal(t, timezone.StartOfDay(now), *sub.DailyWindowStart)
		require.Equal(t, now, *sub.WeeklyWindowStart)
		require.Equal(t, now.Add(-time.Hour), sub.StartsAt)
		require.Equal(t, expires, sub.ExpiresAt)
		require.Equal(t, SubscriptionStatusActive, sub.Status)
	}
	require.Equal(t, 100.0, repo.subs[501].WeeklyUsageUSD)
}

func TestAdminResetGroupQuota_ValidationAndEmptyGroup(t *testing.T) {
	repo := &resetGroupSubs{}
	svc := &SubscriptionService{groupRepo: resetGroupRepo{group: &Group{ID: 7, SubscriptionType: "subscription"}}, userSubRepo: repo, now: time.Now}
	_, err := svc.AdminResetGroupQuota(context.Background(), 7, false, false, false)
	require.ErrorIs(t, err, ErrInvalidInput)
	result, err := svc.AdminResetGroupQuota(context.Background(), 7, false, false, true)
	require.NoError(t, err)
	require.Zero(t, result.ResetCount)
	svc.groupRepo = resetGroupRepo{group: &Group{SubscriptionType: "standard"}}
	_, err = svc.AdminResetGroupQuota(context.Background(), 7, true, true, true)
	require.ErrorIs(t, err, ErrGroupNotSubscriptionType)
	svc.groupRepo = resetGroupRepo{err: ErrGroupNotFound}
	_, err = svc.AdminResetGroupQuota(context.Background(), 7, true, true, true)
	require.ErrorIs(t, err, ErrGroupNotFound)
}
