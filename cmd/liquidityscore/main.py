#!/usr/bin/env python3

import csv
import json
import math
import os
import sys

import numpy

import entropy_calc as entropy
import liquidity_score_calc as liq
import entities

DEV_ENV = "dev"

def main():
    # Get configs from environments
    trade_data_filename = os.environ.get('TRADE__DATA__FILE')
    mean_type = os.environ.get('MEAN__TYPE')
    target_factor_entropy = os.environ.get('TARGET__FACTOR__ENTROPY')
    min_threshold_amount_out_percentage = float(os.environ.get('MIN__THRESHOLD__PERCENTAGE'))
    min_filtered_pools_len = int(os.environ.get('MIN__FILTERED__POOL__LEN'))
    filter_score_filename = os.environ.get('USECASE__UPDATELIQUIDITYSCORE__INPUTFILENAME')
    invalid_scores_filename = os.environ.get('INVALID__SCORES__FILE')
    min_threshold_tvl_in_usd = float(os.environ.get('USECASE__TRADEDATAGENERATOR__MINTHRESHOLDTVL'))

    # Get configs from params
    args = sys.argv
    if len(args) == 5:
        target_factor_entropy = args[1]
        trade_data_filename = args[2]
        filter_score_filename = args[3]
        invalid_scores_filename = args[4]
    
    trade_data_file = read_trade_data(trade_data_filename, min_threshold_amount_out_percentage)
    if len(trade_data_file.pools) == 0:
        return
    
    print(f'invalid pools after read trade data {trade_data_file.invalid_pools}')
    pool_scores = calculate_liquidity_scores(trade_data_file, target_factor_entropy, min_filtered_pools_len, mean_type, invalid_scores_filename, min_threshold_tvl_in_usd)

    save_scores(filter_score_filename, pool_scores)


def filter_scores(pool_scores, mean_type, min_score, min_len):
    pool_scores = sorted(pool_scores, key=lambda pool_score: pool_score[mean_type], reverse=True)
    index = 0
    for i in range(len(pool_scores) - 1, -1, -1):
        if pool_scores[i][mean_type] >= min_score or i < min_len:
            index = i
            break
    
    return pool_scores, index


def save_scores(filename: str, scores: list):
    field_names = ['key', 'pool', 'harmonic', 'geometric', 'arithmetic', 'level']
    with open(filename, 'w', newline='') as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=field_names)
        writer.writeheader()
        for value in scores:
            # format number to avoid rounding
            value_str = {
                'key': value['key'],
                'pool': value['pool'],
                'harmonic': f"{value['harmonic']:.2f}",
                'geometric': f"{value['geometric']:.2f}",
                'arithmetic': f"{value['arithmetic']:.2f}",
                'level': value['level']
            }
            writer.writerow(value_str)

    csvfile.close()
    print(f'Save scores successfully with filename {filename}')

# output is a map from pool -> key -> trade data list and a map level
def read_trade_data(filename, min_threshold_amount_out_percentage):
    f = open(filename, 'r')
    out = f.readlines()
    if len(out) == 0:
        print('file is empty')
        return {}, {}

    pools = {}
    levels = {}
    # a set of pair key, pool_address
    invalid_pools = {entities.DefaultScore.MIN_SCORE: set(), entities.DefaultScore.MAX_SCORE: set()}

    for row in out:
        item = row.split(',', 2)
        # key will be extractly the key of redis sorted set
        key_set_type = item[0]

        pool_addr = item[1]
        try:
            input = json.loads(item[2])
        except Exception as e:
            print(f'parse json trade data failed {item[2]} exception {e}\n')
        else:
            input_object = entities.LiquidityScoreCalcInput(**input)
            if pool_addr not in pools:
                pools[pool_addr] = {}
                levels[pool_addr] = {}
            
            trades = input_object.trade_data
            level_counter = 0
            # count invalid trade 'pool + key + token' -> counter
            invalid_trades_count = {entities.DefaultScore.MIN_SCORE: {}, entities.DefaultScore.MAX_SCORE:{}}
            # count invalid trade 'pool + key + token' -> price_impact
            last_price_impact = {}

            for trade in trades:
                # token will be in format tokenIn - tokenOut
                key = trade['key']
                if key not in pools[pool_addr]:
                    pools[pool_addr][key] = {}
                if key not in levels[pool_addr]:
                    levels[pool_addr][key] = {}

                token = trade['token_in'] + '-' + trade['token_out']
                invalid_trade_key = (pool_addr, key, token)

                amount_in_usd = round(trade['amount_in_usd'])
                log_value = math.log10(amount_in_usd)
                price_impact, ok = calculate_price_impact(trade)

                if token not in pools[pool_addr][key]:
                    pools[pool_addr][key][token] = []
                
                if token not in levels[pool_addr][key]:
                    level_counter = math.floor(log_value) - 1 # level starts from -1
                    levels[pool_addr][key][token] = 0

                if invalid_trade_key not in invalid_trades_count[entities.DefaultScore.MAX_SCORE]:
                    invalid_trades_count[entities.DefaultScore.MAX_SCORE][invalid_trade_key] = 0
                    last_price_impact[invalid_trade_key] = last_price_impact
                if invalid_trade_key not in invalid_trades_count[entities.DefaultScore.MIN_SCORE]:
                    invalid_trades_count[entities.DefaultScore.MIN_SCORE][invalid_trade_key] = 0

                # if both token in and token out have no price, then do not need to calculate level
                if not ok:
                    level_counter = 0
                    invalid_trades_count[entities.DefaultScore.MIN_SCORE][invalid_trade_key] += 1
                elif abs(log_value - round(log_value)) < 1e-9: # check if a number is an integer power of 10
                    # amount out is valid if amount out of trade data 10^x is not the same as amount out of trade data 10^x-1
                    # or amount out of this trade data 10^x > threshold (example, swap 100usd returns at least 80usd can considered valid case)
                    # it means the pools can really serve requests with 10^x successfully, increase level
                    try:
                        amountOut = float(trade['amount_out_usd'])
                        if amountOut == 0:
                            amountOut = 0.8 * amount_in_usd
                        if len(pools[pool_addr][key][token]) == 0 or not numpy.isclose(amountOut, pools[pool_addr][key][token][-1][1],
                                                rtol=0.01) and price_impact > min_threshold_amount_out_percentage:
                            level_counter += 1
                    except Exception as e:
                        print(f'convert value error amount out usd {type(amountOut)} trade data {trade} error {e}')
                    
                    if price_impact >= 1:
                        invalid_trades_count[entities.DefaultScore.MAX_SCORE][invalid_trade_key] += 1
                    elif price_impact == last_price_impact[invalid_trade_key]:
                        invalid_trades_count[entities.DefaultScore.MIN_SCORE][invalid_trade_key] += 1

                if trade['amount_out_usd'] == 0.0:
                    invalid_pools[entities.DefaultScore.MIN_SCORE].add(key)
                    
                pools[pool_addr][key][token].append(
                    (amount_in_usd, trade['amount_out_usd'], trade['token_in'], trade['token_out'], entities.SortedSetType(key_set_type), input_object.liquidity))
                levels[pool_addr][key][token] = max(level_counter, levels[pool_addr][key][token])

            for default_score_key, dict in invalid_trades_count.items():
                for key, count in dict.items():
                    if len(pools[key[0]][key[1]][key[2]]) == count:
                        invalid_pools[default_score_key].add(key)

    return entities.TradeDataGenerationFile(pools, levels, invalid_pools)

def calculate_price_impact(trade):
    # (asumes meme tokens are tokens have no price)
    # pools that swap from meme tokens to meme tokens directly always belongs direct list, just set score = 0 and we always save them into the sorted set.
    if trade['amount_out_usd'] == 0 and trade['amount_in_usd'] == 0:
        return 0, False

    amountIn = trade['amount_in_usd']
    amountOut = trade['amount_out_usd']
    # if amountInUsd = 0, then this trade belongs to non-whitelist - whitelist token set (wl tokens always have their prices)
    # in this case, consider amount out = 70% amount in
    # because swap from meme token to whitelist tokens always returns higher impact from the opposite side
    if amountIn == 0:
        amountIn = amountOut / 0.7

    # if amountOutUsd = 0, then this trade belongs to whitelist non-whitelist token set (wl tokens always have their prices)
    # in this case, consider amount out = 80% amount in
    if amountOut == 0:
        amountOut = 0.8 * amountIn

    price_impact = amountOut / amountIn

    return price_impact, True


# trade_data_file.pools is a map from pool -> key -> token -> trade data list
# invalid_pools is a set of tuple (pool, key, token)
def calculate_liquidity_scores(trade_data_file: entities.TradeDataGenerationFile, target_factor_entropy: float, min_filtered_pools_len, mean_type, invalid_scores_filename, min_threshold_tvl_in_usd) -> list:
    pools = []
    result = []
    # a map with key+pool_address -> list scores
    default_scores = {}
    if len(trade_data_file.pools) == 0:
        print('Read pools from trade data file results empty list')
        return result

    # result is a list
    for pool_addr, value in trade_data_file.pools.items():
        for key, data in value.items():
            for token, trades in data.items():
                score_map_key = key + pool_addr
                level = max(trade_data_file.levels[pool_addr][key].values())
                if (pool_addr, key, token) in trade_data_file.invalid_pools[entities.DefaultScore.MIN_SCORE]:
                    if score_map_key not in default_scores:
                        default_scores[score_map_key] = []
                    tokens = token.split('-')
                    default_scores[score_map_key].append((0.0, tokens[0], tokens[1], level, pool_addr, trades[0][4], trades[0][5]))
                elif (pool_addr, key, token) in trade_data_file.invalid_pools[entities.DefaultScore.MAX_SCORE]:
                    if score_map_key not in default_scores:
                        default_scores[score_map_key] = []
                    max_score = math.pow(10, level-1) - 1
                    tokens = token.split('-')
                    default_scores[score_map_key].append((max_score, tokens[0], tokens[1], level, pool_addr, trades[0][4], trades[0][5]))
                else:
                    try:
                        pools.append(liq.Pool(pool_addr, key, level, trades))
                    except Exception as e:
                        print(f'exception when calculate log values {pool_addr} {key} {trades} {trade_data_file.levels[pool_addr][key].values()} exeption {e}')

    liquidity_scores = liq.calculate_scores(pools, default_scores)
    liquidity_scores_output = liq.calculate_mean_scores(liquidity_scores, entities.MIN_VALID_SCORE)
    
    if len(liquidity_scores_output.whitelist_token_scores) != 0:
        extra_mean_scores = liq.calculate_mean_scores(liquidity_scores_output.whitelist_token_scores, entities.MIN_VALID_SCORE)
        result.extend(extra_mean_scores.scores)
        result.extend(extra_mean_scores.direct_scores.values())

    if float(target_factor_entropy) == 1.0:
        result.extend(liquidity_scores_output.scores)
        result.extend(liquidity_scores_output.direct_scores.values())
        return result

    try:
        # run filter score by calculating entropy
        # only apply for whitelist - whitelist set
        min_score = entropy.get_top_pools(liquidity_scores_output.scores, mean_type, float(target_factor_entropy))
        pool_scores, index = filter_scores(liquidity_scores_output.scores, mean_type, min_score, min_filtered_pools_len)
        final_scores = pool_scores[:index]
        print(f'Length of final scores after filtering: {len(final_scores)} index {index}')
        result.extend(final_scores)
        # save invalid pool scores in separate file to remove them on sorted set later
        if invalid_scores_filename != '' and index != len(pool_scores):
            save_invalid_pool_scores(pool_scores[index:], invalid_scores_filename, min_threshold_tvl_in_usd)
    except Exception as e:
        print(f'exception while calculate entropy values {e}, back to save all scores {liquidity_scores_output.scores}')
        # when exception occurs here, we don't need to filter score
        result.extend(liquidity_scores_output.scores)

    result.extend(liquidity_scores_output.direct_scores.values())
        
    return result

# safe to remove some pools with low liquidity score and low tvl
def save_invalid_pool_scores(pool_scores, invalid_scores_filename, min_threshold_tvl_in_usd):
    invalid_scores = []
    
    for score in pool_scores:
        if score['tvl_in_usd'] <= min_threshold_tvl_in_usd:
            invalid_scores.append(score)
    
    save_scores(invalid_scores_filename, invalid_scores)

main()
