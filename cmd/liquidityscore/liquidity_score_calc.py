import math
import numpy as np
from scipy.optimize import minimize

class Pool:
    def __init__(self, name, level, trades_data):
        self.name = name
        self.level = level
        self.data = []
        for i in range(len(trades_data)):
          self.data.append(compute_price_impacts_log_scale(trades_data[i]))
          
    def append_trade(self, trade_data):
        self.data.append(compute_price_impacts_log_scale(trade_data))
    
    def __repr__(self):
        return str(self.data).join(self.name)

# 1. Harmonic Mean
def harmonic_mean(nums):
    return len(nums) / sum(1/x for x in nums)

# 2. Geometric Mean
def geometric_mean(nums):
    product = 1
    for x in nums:
        product *= x
    return product ** (1/len(nums))

# 3. Arithmetic Mean
def arithmetic_mean(nums):
    return sum(nums) / len(nums)

def compute_price_impacts_log_scale(trades_data):
    price_impacts_original = [(trade, (1 - (received / trade)), token_in, token_out) for trade, received, token_in, token_out in trades_data]
    return [(math.log(trade, 10), impact, token_in, token_out) for trade, impact, token_in, token_out in price_impacts_original]

def distance(m, trades):
    x_values, y_values, _, _ = zip(*trades)
    return np.sum(np.abs(y_values - (10**np.array(x_values) - 1) / (m + 10**np.array(x_values) - 1)))

def find_best_m(trades):
    x_values, y_values, _, _ = zip(*trades)
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

def pool_score(pools) -> list:
    result = []
    for _, pool in enumerate(pools):
        tmp_trade_data=[]
        for i in range(len(pool.data)):
          tmp_score = find_best_m(pool.data[i])
          tmp_trade_data.append((tmp_score, pool.data[i][0][2], pool.data[i][0][3]))
        
        # clean up trade data which has tokenIn with 0 price impact
        price_impact_zero_tokens = set()
        valid_scores = []
        for score, token_in, token_out in tmp_trade_data:
            if score == 0:
                price_impact_zero_tokens.add(token_in)
            if token_in not in price_impact_zero_tokens:
                valid_scores.append(score)

        if len(valid_scores) < 2:
            result.append({
                'pool': pool.name,
                'harmonic': float(valid_scores[0]),
                'geometric': float(valid_scores[0]),
                'arithmetic': float(valid_scores[0]),
                'level': pool.level
            })
        else:
            result.append(
                {
                'pool': pool.name,
                'harmonic': float(harmonic_mean(valid_scores)),
                'geometric': float(geometric_mean(valid_scores)),
                'arithmetic': float(arithmetic_mean(valid_scores)),
                'level': pool.level
            })
    
    return result


