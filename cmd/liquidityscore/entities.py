from enum import Enum

MIN_VALID_SCORE = 0.002

class DefaultScore(Enum):
    MIN_SCORE = 1
    MAX_SCORE = 2


class SortedSetType(Enum):
    WHITELIST_WHITELIST = "whitelist-whitelist"
    TOKEN_WHITELIST     = "token-whitelist"
    WHITELIST_TOKEN     = "whitelist-token"
    DIRECT              = "direct"

class LiquidityScoreCalcInput:
    def __init__(self, trade_data, liquidity):
        self.trade_data = trade_data
        self.liquidity = liquidity


class TradeDataGenerationFile:
    def __init__(self, pools, levels, invalid_pools):
        self.pools = pools
        self.levels = levels
        self.invalid_pools = invalid_pools


class LiquidityScoreOutput:
    def __init__(self, scores, direct_scores, whitelist_token_scores):
        self.scores = scores
        self.direct_scores = direct_scores
        self.whitelist_token_scores = whitelist_token_scores