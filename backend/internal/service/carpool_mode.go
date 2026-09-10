package service

import "github.com/Wei-Shaw/sub2api/internal/config"

// IsCarpoolMode retains subscription billing and quota enforcement while
// disabling the retail and referral surfaces of a shared subscription service.
func (s *SettingService) IsCarpoolMode() bool {
	return s != nil && s.cfg != nil && s.cfg.RunMode == config.RunModeCarpool
}
