//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestCarpoolDisablesCommercialSettingsBeforeRepositoryAccess(t *testing.T) {
	// A nil repository deliberately makes any persisted-setting lookup fail.
	svc := &SettingService{cfg: &config.Config{RunMode: config.RunModeCarpool}}
	ctx := context.Background()
	if !svc.IsCarpoolMode() || svc.IsPromoCodeEnabled(ctx) ||
		svc.IsRegistrationEnabled(ctx) || svc.GetModelPlazaRuntime(ctx).Enabled ||
		svc.IsAffiliateEnabled(ctx) || svc.IsAffiliateAdminRechargeEnabled(ctx) {
		t.Fatal("carpool must disable commercial rewards regardless of stored settings")
	}
	for _, mode := range []string{config.RunModeStandard, config.RunModeSimple} {
		if (&SettingService{cfg: &config.Config{RunMode: mode}}).IsCarpoolMode() {
			t.Fatalf("%s incorrectly treated as carpool", mode)
		}
	}
	var absent *SettingService
	if absent.IsCarpoolMode() {
		t.Fatal("nil settings must not enable carpool")
	}
}

func TestCarpoolDisablesSignupGiftsAndOAuthRegistrationBypass(t *testing.T) {
	cfg := &config.Config{RunMode: config.RunModeCarpool}
	cfg.Default.UserBalance = 100
	cfg.Default.UserConcurrency = 2
	svc := &AuthService{cfg: cfg, settingService: &SettingService{cfg: cfg}}
	plan := svc.resolveSignupGrantPlan(context.Background(), "dingtalk")
	if plan.Balance != 0 || len(plan.Subscriptions) != 0 || len(plan.PlatformQuotas) != 0 {
		t.Fatal("carpool must not grant signup balance, subscriptions or quotas")
	}
	if svc.canBypassRegistrationDisabledForOAuth(context.Background(), "dingtalk") {
		t.Fatal("OAuth must not bypass registration disabled in carpool")
	}
	if svc.settingService.GetCustomMenuItemsRaw(context.Background()) != "[]" {
		t.Fatal("custom pages must not be available in carpool")
	}
}
