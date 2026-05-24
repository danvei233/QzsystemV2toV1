package repository

import "xiaoheiproxy/internal/domain"

func normalizeRetention(policy domain.LogRetentionPolicy) domain.LogRetentionPolicy {
	if policy.RetentionDays <= 0 {
		policy.RetentionDays = 7
	}
	if policy.MaxSizeBytes <= 0 {
		policy.MaxSizeBytes = 100 * 1024 * 1024
	}
	return policy
}
