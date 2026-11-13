package repositories

import (
	"context"
	"time"

	"github.com/marketdome/otp-api/pkg/types"
	"github.com/redis/go-redis/v9"
)

// consumeIfMatch deletes the stored code only when it matches, so a code can be used once.
var consumeIfMatch = redis.NewScript(`
	local stored = redis.call('GET', KEYS[1])
	if stored == ARGV[1] then
		redis.call('DEL', KEYS[1])
		return 1
	end
	return 0
`)

type otpRepository struct {
	store *redis.Client
}

func (repository *otpRepository) Save(ctx context.Context, phoneNumber, otp string, ttl time.Duration) error {
	return repository.store.Set(ctx, otpKey(phoneNumber), otp, ttl).Err()
}

func (repository *otpRepository) ConsumeIfMatch(ctx context.Context, phoneNumber, otp string) (bool, error) {
	result, err := consumeIfMatch.Run(ctx, repository.store, []string{otpKey(phoneNumber)}, otp).Int()
	return result == 1, err
}

func otpKey(phoneNumber string) string {
	return "otp:" + phoneNumber
}

func NewOTPRepository(store *redis.Client) types.OTPRepository {
	return &otpRepository{store: store}
}
