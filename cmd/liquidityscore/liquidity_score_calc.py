import csv
import math
import numpy as np
from scipy.optimize import minimize
import entities

class Pool:
    def __init__(self, address, key, level, trades_data):
        self.address = address
        self.key = key
        self.level = level
        self.data = []
        self.data = compute_price_impacts_log_scale(trades_data)
    
    def __repr__(self):
        return str(self.data).join(self.address)

# 1. Harmonic Mean
def harmonic_mean(nums):
    return len(nums) / sum(1/x for x in nums)

# 2. Geometric Mean
def geometric_mean(nums):
    product = 1
    for x in nums:
        product *= x
    return product ** (1/len(nums))

# 2. Geometric Mean which doesn't return inf value
def geometric_mean_safe(nums):
    if not nums:
        return 0.0
    log_sum = 0.0
    for x in nums:
        if x < 0:
            raise ValueError('input data must contain non-negative numbers.')
        if x == 0:
            return 0.0
        log_sum += math.log(x)

    return math.exp(log_sum / len(nums))

# 3. Arithmetic Mean
def arithmetic_mean(nums):
    return sum(nums) / len(nums)

def compute_price_impacts_log_scale(trades_data):
    price_impacts_original = [(trade, (1 - (received / trade)), token_in, token_out, set_type) for trade, received, token_in, token_out, set_type in trades_data]
    return [(math.log(trade, 10), impact, token_in, token_out, set_type) for trade, impact, token_in, token_out, set_type in price_impacts_original]

def distance(m, trades):
    x_values, y_values, _, _, _ = zip(*trades)
    return np.sum(np.abs(y_values - (10**np.array(x_values) - 1) / (m + 10**np.array(x_values) - 1)))

def find_best_m(trades):
    x_values, y_values, _, _, _ = zip(*trades)
    x_values = np.array(x_values)
    y_values = np.array(y_values)

    # Compute initial m estimates from the data
    m_values = ((1 - y_values) / y_values) * (10**x_values - 1)

    # Filter out any non-positive m_values
    m_values = m_values[m_values > 0]

    if len(m_values) == 0:
        initial_m = 0.001  # Default to 0.001 if no positive m_values
    else:
        initial_m = np.mean(m_values)

    # Ensure initial_m is within bounds
    max_score = 10**max(trades, key=lambda x: x[0])[0]
    initial_m = np.clip(initial_m, 0.001, max_score)
    result = minimize(
        lambda m: distance(m[0], trades),
        [initial_m],
        method='Nelder-Mead',
        bounds=[(0.001, max_score)]
    )
    return result.x[0]

def calculate_scores(pools, default_scores) -> dict:
    for _, pool in enumerate(pools):
        # add try catch exeption here to make sure if find_best_m failed for 1 pool, others still get success
        try:
            tmp_score = find_best_m(pool.data)
            map_key = pool.key + pool.address
            if map_key not in default_scores:
                default_scores[map_key] = []
            # score, token_in, token_out, level, pool_address
            default_scores[map_key].append((tmp_score, pool.data[0][2], pool.data[0][3], pool.level, pool.address, pool.data[0][4]))
        except Exception as e:
            print(f'Exception while find_best_m {e} pool {pool.address} tradedata {pool.data}')
    
    return default_scores

def calculate_mean_scores(score_map, min_valid_score) -> entities.LiquidityScoreOutput:
    # dictionary with key key+pool
    direct_scores = {}
    extra_scores = {}
    result = []
    
    for key, scores in score_map.items():
        # clean up trade data which has tokenIn with 0 price impact
        price_impact_zero_tokens = set()
        valid_scores = []
        invalid_scores = []
        # we need to filter out zero score from mean calculation input
        # with every whitelist-whitelist pair score, we have to store the pair scores in to 3 different keys
        # example: we have a pair ETH-USDC, we have to store mean value score as whitelist-whitelis key
        # ETH-whitelist score for the original value, and whitelist-USDC for the original value as well
        for score, token_in, token_out, level, pool_address, key_set_type in scores:
            score_key = key[:-len(pool_address)]
            if score <= min_valid_score:
                price_impact_zero_tokens.add(token_in)
            if token_in not in price_impact_zero_tokens:
                valid_scores.append(score)
            else:
                invalid_scores.append(score)

            if key_set_type == entities.SortedSetType.WHITELIST_WHITELIST:
                token_wl_key = extract_prefix(score_key, key_set_type, len(token_in)) + token_in + ':whitelist' + pool_address
                if token_wl_key not in extra_scores:
                    extra_scores[token_wl_key] = []
                wl_token_key = extract_prefix(score_key, key_set_type, len(token_out)) + 'whitelist:' + token_out + pool_address
                if wl_token_key not in extra_scores:
                    extra_scores[wl_token_key] = []
                                
                extra_scores[token_wl_key].append((score, token_in, token_out, level, pool_address, entities.SortedSetType.TOKEN_WHITELIST))
                extra_scores[wl_token_key].append((score, token_in, token_out, level, pool_address, entities.SortedSetType.WHITELIST_TOKEN))
            if key_set_type == entities.SortedSetType.TOKEN_WHITELIST or key_set_type == entities.SortedSetType.WHITELIST_TOKEN:
                direct_key = extract_prefix(score_key, key_set_type, len(token_in)) + token_in + '-' + token_out
                direct_scores[direct_key+pool_address] = {
                    'key': direct_key,
                    'pool': pool_address,
                    'harmonic': score,
                    'geometric': score,
                    'arithmetic': score,
                    'level': level
                }
        
        sorted_set_key = key[:-len(pool_address)]
        if len(valid_scores) == 0:
            valid_scores = invalid_scores
            
        if len(valid_scores) < 2:
            result.append({
                'key': sorted_set_key,
                'pool': pool_address,
                'harmonic': float(valid_scores[0]),
                'geometric': float(valid_scores[0]),
                'arithmetic': float(valid_scores[0]),
                'level': level
            })
        else:
            try:
                harmonic = harmonic_mean(valid_scores)
                geometric = geometric_mean(valid_scores)
                if math.isinf(geometric):
                    geometric = geometric_mean_safe(valid_scores)
                    print(f'Geometric mean is inf with pool {pool_address} scores {valid_scores} approximate geometric mean {geometric}')

                arithmetic = arithmetic_mean(valid_scores)
                result.append({
                    'key': sorted_set_key,
                    'pool': pool_address,
                    'harmonic': float(harmonic),
                    'geometric': float(geometric),
                    'arithmetic': float(arithmetic),
                    'level': level
                })
            except Exception as e:
                print(f'Exception while calculate mean values {e} pool {pool_address} scores {valid_scores}')

    return entities.LiquidityScoreOutput(result, direct_scores, extra_scores)

def extract_prefix(score_key, type, token_len):
    match type:
        case entities.SortedSetType.WHITELIST_WHITELIST:
            return score_key[:-len('whitelist')]
        # "ethereum:liquidityScoreTvl:whitelist:0xe6300a5d7c5bf23af11f8d85b0372a7b54a7256f"
        case entities.SortedSetType.WHITELIST_TOKEN:
            return score_key[:-(len('whitelist:') + token_len)]
        # "ethereum:liquidityScoreTvl:0xe6300a5d7c5bf23af11f8d85b0372a7b54a7256f:whitelist"
        case entities.SortedSetType.TOKEN_WHITELIST:
            return score_key[:-(token_len + len(':whitelist'))]
    
    return ''


