#!/usr/bin/env python3

import ast
import csv
import math
import os

import numpy

import entropy_calc as entropy
import liquidity_score_calc as liq

DEV_ENV = "dev"


def main():
    env = os.environ.get('ENV')
    trade_data_filename = os.environ.get('TRADE__DATA__FILE')
    score_filename = os.environ.get('FULL__SCORE__FILE')
    mean_type = os.environ.get('MEAN__TYPE')
    target_factor_entropy = os.environ.get('TARGET__FACTOR__ENTROPY')
    min_threshold_amount_out_percentage = os.environ.get('MIN__THRESHOLD__PERCENTAGE')

    pools = read_trade_data(trade_data_filename, float(min_threshold_amount_out_percentage))
    if len(pools) == 0:
        print('Read pools from trade data file results empty list')
        return

    pool_scores = liq.pool_score(pools)
    # using score.txt for debug only purpose
    if env == DEV_ENV:
        save_scores(score_filename, pool_scores)

    # run filter score by calculating entropy
    min_score = entropy.get_top_pools(pool_scores, mean_type, float(target_factor_entropy))

    filter_score_filename = os.environ.get('USECASE__UPDATELIQUIDITYSCORE__INPUTFILENAME')
    filter_scores = [v for v in pool_scores if v[mean_type] > min_score]
    save_scores(filter_score_filename, filter_scores)


def save_scores(filename: str, scores: list):
    field_names = ['pool', 'harmonic', 'geometric', 'arithmetic', 'level']
    with open(filename, 'w', newline='') as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=field_names)
        writer.writeheader()
        for value in scores:
            writer.writerow(value)

    csvfile.close()


def read_trade_data(filename, min_threshold_amount_out_percentage) -> list:
    f = open(filename, 'r')
    out = f.readlines()
    if len(out) == 0:
        print('file is empty')
        return []

    pools = {}
    levels = {}

    for row in out:
        item = row.split(":", 2)
        # token will be in format tokenIn-tokenOut
        pool_addr = item[0]
        token = item[1]
        level_counter = 0
        trades = ast.literal_eval(item[2])
        if pool_addr not in pools:
            pools[pool_addr] = {}
            levels[pool_addr] = []

        for trade in trades:
            if token not in pools[pool_addr]:
                pools[pool_addr][token] = [
                    (trade['AmountInUsd'], trade['AmountOutUsd'], trade['TokenIn'], trade['TokenOut'])]
                level_counter = round(math.log10(trade['AmountInUsd']))
            else:
                price_impact = trade['AmountOutUsd'] / trade['AmountInUsd']
                if not numpy.isclose(trade['AmountOutUsd'], pools[pool_addr][token][-1][1],
                                     rtol=0.01) and price_impact > min_threshold_amount_out_percentage:
                    level_counter += 1

                pools[pool_addr][token].append(
                    (trade['AmountInUsd'], trade['AmountOutUsd'], trade['TokenIn'], trade['TokenOut']))
        levels[pool_addr].append(level_counter)

    result = []
    for p, data in pools.items():
        result.append(liq.Pool(p, max(levels[p]), list(data.values())))

    return result


main()
