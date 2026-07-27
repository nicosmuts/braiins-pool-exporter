package braiins

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Decimal preserves API numeric precision by retaining the original JSON
// token text. Braiins documents BTC values as strings in several responses and
// high-precision hashrates as JSON numbers.
type Decimal string

// UnmarshalJSON accepts JSON strings or numbers and rejects null or composite
// values.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		return fmt.Errorf("decimal value is required")
	}
	if strings.HasPrefix(text, `"`) {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("decimal string is empty")
		}
		*d = Decimal(value)
		return nil
	}
	if !isJSONNumber(text) {
		return fmt.Errorf("decimal must be encoded as a JSON string or number")
	}
	*d = Decimal(text)
	return nil
}

func (d Decimal) String() string { return string(d) }

type CoinEnvelope[T any] map[string]T

type PoolStats struct {
	HashRateUnit      string               `json:"hash_rate_unit"`
	PoolActiveWorkers *int64               `json:"pool_active_workers"`
	PoolHashRate5m    Decimal              `json:"pool_5m_hash_rate"`
	PoolHashRate60m   Decimal              `json:"pool_60m_hash_rate"`
	PoolHashRate24h   Decimal              `json:"pool_24h_hash_rate"`
	UpdateTS          int64                `json:"update_ts"`
	Blocks            map[string]PoolBlock `json:"blocks"`
	FPPSRate          *Decimal             `json:"fpps_rate"`
}

type PoolBlock struct {
	DateFound           int64   `json:"date_found"`
	MiningDuration      int64   `json:"mining_duration"`
	TotalShares         Decimal `json:"total_shares"`
	State               string  `json:"state"`
	ConfirmationsLeft   int64   `json:"confirmations_left"`
	Value               Decimal `json:"value"`
	UserReward          Decimal `json:"user_reward"`
	PoolScoringHashRate Decimal `json:"pool_scoring_hash_rate"`
}

type ProfileResponse struct {
	Username string             `json:"username"`
	Coins    map[string]Profile `json:"-"`
}

func (p *ProfileResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.Coins = make(map[string]Profile)
	for key, value := range raw {
		if key == "username" {
			if err := json.Unmarshal(value, &p.Username); err != nil {
				return err
			}
			continue
		}
		var profile Profile
		if err := json.Unmarshal(value, &profile); err != nil {
			return err
		}
		p.Coins[key] = profile
	}
	return nil
}

type Profile struct {
	AllTimeReward     Decimal `json:"all_time_reward"`
	HashRateUnit      string  `json:"hash_rate_unit"`
	HashRate5m        Decimal `json:"hash_rate_5m"`
	HashRate60m       Decimal `json:"hash_rate_60m"`
	HashRate24h       Decimal `json:"hash_rate_24h"`
	HashRateYesterday Decimal `json:"hash_rate_yesterday"`
	LowWorkers        int64   `json:"low_workers"`
	OffWorkers        int64   `json:"off_workers"`
	OKWorkers         int64   `json:"ok_workers"`
	DisabledWorkers   int64   `json:"dis_workers"`
	CurrentBalance    Decimal `json:"current_balance"`
	TodayReward       Decimal `json:"today_reward"`
	EstimatedReward   Decimal `json:"estimated_reward"`
	Shares5m          Decimal `json:"shares_5m"`
	Shares60m         Decimal `json:"shares_60m"`
	Shares24h         Decimal `json:"shares_24h"`
	SharesYesterday   Decimal `json:"shares_yesterday"`
}

type RewardsResponse struct {
	DailyRewards []DailyReward `json:"daily_rewards"`
}

type DailyReward struct {
	Date            int64        `json:"date"`
	TotalReward     Decimal      `json:"total_reward"`
	MiningReward    Decimal      `json:"mining_reward"`
	BOSPlusReward   Decimal      `json:"bos_plus_reward"`
	ReferralBonus   Decimal      `json:"referral_bonus"`
	ReferralReward  Decimal      `json:"referral_reward"`
	Shares          Decimal      `json:"shares"`
	SharePrices     []SharePrice `json:"share_prices"`
	CalculationDate int64        `json:"calculation_date"`
}

type SharePrice struct {
	FromTS     int64   `json:"from_ts"`
	ToTS       int64   `json:"to_ts"`
	SharePrice Decimal `json:"share_price"`
}

func (p *SharePrice) UnmarshalJSON(data []byte) error {
	var scalar Decimal
	if err := json.Unmarshal(data, &scalar); err == nil {
		p.SharePrice = scalar
		return nil
	}
	type sharePrice SharePrice
	var object sharePrice
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	*p = SharePrice(object)
	return nil
}

type DailyHashrate struct {
	Date         int64   `json:"date"`
	HashRateUnit string  `json:"hash_rate_unit"`
	HashRate24h  Decimal `json:"hash_rate_24h"`
	TotalShares  Decimal `json:"total_shares"`
}

type BlockRewardsResponse struct {
	BlockRewards []BlockReward `json:"block_rewards"`
	HashRateUnit string        `json:"hash_rate_unit"`
}

type BlockReward struct {
	BlockFoundAt             int64   `json:"block_found_at"`
	PoolScoringHashRate      Decimal `json:"pool_scoring_hash_rate"`
	UserScoringHashRate      Decimal `json:"user_scoring_hash_rate"`
	BlockValue               Decimal `json:"block_value"`
	UserReward               Decimal `json:"user_reward"`
	BlockHeight              int64   `json:"block_height"`
	MiningReward             Decimal `json:"mining_reward"`
	BraiinsOSPlusMiningBonus Decimal `json:"braiinsos_plus_mining_bonus"`
	ReferralReward           Decimal `json:"referral_reward"`
	ReferralBonus            Decimal `json:"referral_bonus"`
	ConfirmationsLeft        int64   `json:"confirmations_left"`
}

type WorkersResponse struct {
	Workers map[string]Worker `json:"workers"`
}

type Worker struct {
	State           string   `json:"state"`
	LastShare       *int64   `json:"last_share"`
	HashRateUnit    string   `json:"hash_rate_unit"`
	HashRateScoring *Decimal `json:"hash_rate_scoring"`
	HashRate5m      Decimal  `json:"hash_rate_5m"`
	HashRate60m     Decimal  `json:"hash_rate_60m"`
	HashRate24h     Decimal  `json:"hash_rate_24h"`
	Shares5m        Decimal  `json:"shares_5m"`
	Shares60m       Decimal  `json:"shares_60m"`
	Shares24h       Decimal  `json:"shares_24h"`
}

type PayoutsResponse struct {
	Onchain   []Payout `json:"onchain"`
	Lightning []Payout `json:"lightning"`
}

type Payout struct {
	FinancialAccountName string  `json:"financial_account_name"`
	RequestedAtTS        int64   `json:"requested_at_ts"`
	ResolvedAtTS         int64   `json:"resolved_at_ts"`
	Status               string  `json:"status"`
	AmountSats           int64   `json:"amount_sats"`
	FeeSats              int64   `json:"fee_sats"`
	Destination          string  `json:"destination"`
	TxID                 *string `json:"tx_id"`
	Invoice              *string `json:"invoice"`
	Preimage             *string `json:"preimage"`
	TriggerType          string  `json:"trigger_type"`
}

func isJSONNumber(text string) bool {
	var number json.Number
	return json.Unmarshal([]byte(text), &number) == nil
}
