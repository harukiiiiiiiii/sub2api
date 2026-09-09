package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

type GroupQuotaResetResult struct {
	ResetCount    int `json:"reset_count"`
	CacheWarnings int `json:"cache_warnings"`
}

// AdminResetGroupQuota resets a snapshot of all non-revoked subscriptions in
// one transaction. Neither subscription terms nor historical usage logs change.
func (s *SubscriptionService) AdminResetGroupQuota(ctx context.Context, groupID int64, daily, weekly, monthly bool) (*GroupQuotaResetResult, error) {
	if !daily && !weekly && !monthly {
		return nil, ErrInvalidInput
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !group.IsSubscriptionType() {
		return nil, ErrGroupNotSubscriptionType
	}

	txCtx := ctx
	var tx *dbent.Tx
	if s.entClient != nil {
		// Stable pagination even if subscriptions are created concurrently.
		tx, err = s.entClient.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		if err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback() }() // No-op after a successful commit.
		txCtx = dbent.NewTxContext(ctx, tx)
	}
	now := s.now()
	var users []int64
	for page := 1; ; page++ {
		subs, info, err := s.userSubRepo.ListByGroupID(txCtx, groupID, pagination.PaginationParams{Page: page, PageSize: 500})
		if err != nil {
			return nil, err
		}
		for _, sub := range subs {
			if err := s.userSubRepo.ResetUsageWindows(txCtx, sub.ID, daily, weekly, monthly, timezone.StartOfDay(now), now); err != nil {
				return nil, fmt.Errorf("reset subscription %d: %w", sub.ID, err)
			}
			users = append(users, sub.UserID)
		}
		if len(subs) == 0 || info == nil || page >= info.Pages {
			break
		}
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}
	result := &GroupQuotaResetResult{ResetCount: len(users)}
	// Invalidate only after commit, including the other instances' L1 caches.
	// Bound the whole cache refresh phase even when Redis is unavailable.
	cacheCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, userID := range users {
		s.InvalidateSubCacheSync(userID, groupID)
		if s.billingCacheService == nil {
			continue
		}
		invalidateErr := s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		publishErr := s.billingCacheService.PublishSubscriptionCacheInvalidation(cacheCtx, subCacheKey(userID, groupID))
		if invalidateErr != nil || publishErr != nil {
			result.CacheWarnings++
			log.Printf("group quota reset cache invalidation: user=%d group=%d: invalidate=%v publish=%v", userID, groupID, invalidateErr, publishErr)
		}
	}
	return result, nil
}
