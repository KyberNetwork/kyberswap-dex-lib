package alphafee

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/stretchr/testify/assert"
)

const routeSummaryStr = `
{
	"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	"amountIn": 427029674778,
	"amountInUsd": 426703.04383881437,
	"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
	"amountOut": 442308321,
	"amountOutUsd": 426105.2430285946,
	"gas": 5884562,
	"gasPrice": "0",
	"gasUsd": 4.245012132379891,
	"l1FeeUsd": 0,
	"extraFee": {
		"feeAmount": [0],
		"chargeFeeBy": "",
		"isInBps": false,
		"feeReceiver": [""]
	},
	"route": [
		[
			{
				"pool": "0x8aa4e11cbdf30eedc92100f4c8a31ff748e201d44712cc8c90d189edaa8e4e47",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 38432670746,
				"amountOut": 38424009569,
				"exchange": "uniswap-v4",
				"poolType": "uniswap-v4",
				"poolExtra": {
					"router": "0x66a9893cc07d91d95644aedd05d03f95e1dba8af",
					"permit2Addr": "0x000000000022d473030f116ddee9f6b43ac78ba3",
					"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"fee": 10,
					"tickSpacing": 1,
					"hookAddress": "0x0000000000000000000000000000000000000000",
					"hookData": ""
				},
				"extra": {
					"nSqrtRx96": "79219131371359678640007362854"
				}
			},
			{
				"pool": "0x56534741cd8b152df6d48adf7ac51f75169a83b2",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 38424009569,
				"amountOut": 39823510,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 500,
					"priceLimit": "1461300573427867316570072651998408279850435624080"
				},
				"extra": {
					"nSqrtRx96": "2461166421285286696339980168298"
				}
			}
		],
		[
			{
				"pool": "0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"swapAmount": 25621780486,
				"amountOut": 14101736921852430894,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 500,
					"priceLimit": "4295558253"
				},
				"extra": {
					"nSqrtRx96": "1859068346748804331934278405978284"
				}
			},
			{
				"pool": "0x54c72c46df32f2cc455e84e41e191b26ed73a29452cdd3d82f511097af9f427e",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 14101736921852430894,
				"amountOut": 26551761,
				"exchange": "uniswap-v4",
				"poolType": "uniswap-v4",
				"poolExtra": {
					"router": "0x66a9893cc07d91d95644aedd05d03f95e1dba8af",
					"permit2Addr": "0x000000000022d473030f116ddee9f6b43ac78ba3",
					"tokenIn": "0x0000000000000000000000000000000000000000",
					"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
					"fee": 3000,
					"tickSpacing": 60,
					"hookAddress": "0x0000000000000000000000000000000000000000",
					"hookData": ""
				},
				"extra": {
					"nSqrtRx96": "108838551630492074087159"
				}
			}
		],
		[
			{
				"pool": "0x9a772018fbd77fcd2d25657e5c547baff3fd7d16",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 55513857718,
				"amountOut": 57513465,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 500,
					"priceLimit": "1461300573427867316570072651998408279850435624080"
				},
				"extra": {
					"nSqrtRx96": "2461514782898424644611082479702"
				}
			}
		],
		[
			{
				"pool": "0x667701e51b4d1ca244f17c78f7ab8744b4c99f9b",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 38432670729,
				"amountOut": 38424280015,
				"exchange": "fluid-dex-t1",
				"poolType": "fluid-dex-t1",
				"poolExtra": {
					"blockNumber": 22394840
				},
				"extra": {
					"hasNative": false
				}
			},
			{
				"pool": "0x048f0e7ea2cfd522a4a058d1b1bdd574a0486c46",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"swapAmount": 38424280015,
				"amountOut": 21150789652787714531,
				"exchange": "integral",
				"poolType": "integral",
				"poolExtra": null,
				"extra": {
					"relayerAddress": "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E"
				}
			},
			{
				"pool": "0x37f6df71b40c50b2038329cabf5fda3682df1ebf",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 21150789652787714531,
				"amountOut": 39823248,
				"exchange": "integral",
				"poolType": "integral",
				"poolExtra": null,
				"extra": {
					"relayerAddress": "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E"
				}
			}
		],
		[
			{
				"pool": "0x8aa4e11cbdf30eedc92100f4c8a31ff748e201d44712cc8c90d189edaa8e4e47",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 17081186990,
				"amountOut": 17077273293,
				"exchange": "uniswap-v4",
				"poolType": "uniswap-v4",
				"poolExtra": {
					"router": "0x66a9893cc07d91d95644aedd05d03f95e1dba8af",
					"permit2Addr": "0x000000000022d473030f116ddee9f6b43ac78ba3",
					"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"fee": 10,
					"tickSpacing": 1,
					"hookAddress": "0x0000000000000000000000000000000000000000",
					"hookData": ""
				},
				"extra": {
					"nSqrtRx96": "79219039616014287339041279931"
				}
			},
			{
				"pool": "0x048f0e7ea2cfd522a4a058d1b1bdd574a0486c46",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"swapAmount": 17077273293,
				"amountOut": 9400249402953438148,
				"exchange": "integral",
				"poolType": "integral",
				"poolExtra": null,
				"extra": {
					"relayerAddress": "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E"
				}
			},
			{
				"pool": "0x37f6df71b40c50b2038329cabf5fda3682df1ebf",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 9400249402953438148,
				"amountOut": 17699029,
				"exchange": "integral",
				"poolType": "integral",
				"poolExtra": null,
				"extra": {
					"relayerAddress": "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E"
				}
			}
		],
		[
			{
				"pool": "0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"swapAmount": 12810890243,
				"amountOut": 7049782058819715903,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 500,
					"priceLimit": "4295558253"
				},
				"extra": {
					"nSqrtRx96": "1858972864645315963343317125204777"
				}
			},
			{
				"pool": "0x43e818a9e1e07434629babf873a4f717aff93754",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf",
				"swapAmount": 7049782058819715903,
				"amountOut": 13273108,
				"exchange": "ringswap",
				"poolType": "ringswap",
				"poolExtra": {
					"fee": 3,
					"feePrecision": 1000,
					"blockNumber": 22394840
				},
				"extra": {
					"wTokenIn": "0xa250cc729bb3323e7933022a67b52200fe354767",
					"wTokenOut": "0xdbf1703e5d29afefbf1bd958ce7a6023c67f3e5d",
					"isToken0To1": true,
					"isWrapIn": true,
					"isUnwrapOut": true
				}
			},
			{
				"pool": "0xe8f7c89c5efa061e340f2d2f206ec78fd8f7e124",
				"tokenIn": "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 13273108,
				"amountOut": 13279356,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 100,
					"priceLimit": "1461446703485210103287273052203988822378723970341"
				},
				"extra": {
					"nSqrtRx96": "79205972481581453423152891841"
				}
			}
		],
		[
			{
				"pool": "0x8aa4e11cbdf30eedc92100f4c8a31ff748e201d44712cc8c90d189edaa8e4e47",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 34162373981,
				"amountOut": 34154427909,
				"exchange": "uniswap-v4",
				"poolType": "uniswap-v4",
				"poolExtra": {
					"router": "0x66a9893cc07d91d95644aedd05d03f95e1dba8af",
					"permit2Addr": "0x000000000022d473030f116ddee9f6b43ac78ba3",
					"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"fee": 10,
					"tickSpacing": 1,
					"hookAddress": "0x0000000000000000000000000000000000000000",
					"hookData": ""
				},
				"extra": {
					"nSqrtRx96": "79218856105961150177198331108"
				}
			},
			{
				"pool": "0x048f0e7ea2cfd522a4a058d1b1bdd574a0486c46",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"swapAmount": 34154427909,
				"amountOut": 18800433479555250943,
				"exchange": "integral",
				"poolType": "integral",
				"poolExtra": null,
				"extra": {
					"relayerAddress": "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E"
				}
			},
			{
				"pool": "0x4585fe77225b41b697c938b018e2ac67ac5a20c0",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 18800433479555250943,
				"amountOut": 35390316,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 500,
					"priceLimit": "1461300573427867316570072651998408279850435624080"
				},
				"extra": {
					"nSqrtRx96": "57737983475573041793749089677135592"
				}
			}
		],
		[
			{
				"pool": "0x4f493b7de8aac7d55f71853688b1f7c8f0243c85",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 21351483738,
				"amountOut": 21346561000,
				"exchange": "curve-stable-ng",
				"poolType": "curve-stable-ng",
				"poolExtra": {
					"tokenInIndex": 0,
					"tokenOutIndex": 1,
					"underlying": false,
					"TokenInIsNative": false,
					"TokenOutIsNative": false
				},
				"extra": null
			},
			{
				"pool": "0x048f0e7ea2cfd522a4a058d1b1bdd574a0486c46",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"swapAmount": 21346561000,
				"amountOut": 11750294900950380525,
				"exchange": "integral",
				"poolType": "integral",
				"poolExtra": null,
				"extra": {
					"relayerAddress": "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E"
				}
			},
			{
				"pool": "0xcbcdf9626bc03e24f779434178a73a0b4bad62ed",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 11750294900950380525,
				"amountOut": 22117020,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 3000,
					"priceLimit": "1457652066949847389969617340386294118487833376467"
				},
				"extra": {
					"nSqrtRx96": "57669471774101402986372889698639512"
				}
			}
		],
		[
			{
				"pool": "0x8aa4e11cbdf30eedc92100f4c8a31ff748e201d44712cc8c90d189edaa8e4e47",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 12810890243,
				"amountOut": 12807869670,
				"exchange": "uniswap-v4",
				"poolType": "uniswap-v4",
				"poolExtra": {
					"router": "0x66a9893cc07d91d95644aedd05d03f95e1dba8af",
					"permit2Addr": "0x000000000022d473030f116ddee9f6b43ac78ba3",
					"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"fee": 10,
					"tickSpacing": 1,
					"hookAddress": "0x0000000000000000000000000000000000000000",
					"hookData": ""
				},
				"extra": {
					"nSqrtRx96": "79218787289910414584576949798"
				}
			},
			{
				"pool": "0x048f0e7ea2cfd522a4a058d1b1bdd574a0486c46",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"swapAmount": 12807869670,
				"amountOut": 7050140098693314133,
				"exchange": "integral",
				"poolType": "integral",
				"poolExtra": null,
				"extra": {
					"relayerAddress": "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E"
				}
			},
			{
				"pool": "0x00b06862de00a7e67a2d6d3fbeea592a32460de0",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 7050140098693314133,
				"amountOut": 13277030,
				"exchange": "ringswap",
				"poolType": "ringswap",
				"poolExtra": {
					"fee": 3,
					"feePrecision": 1000,
					"blockNumber": 22394840
				},
				"extra": {
					"wTokenIn": "0xa250cc729bb3323e7933022a67b52200fe354767",
					"wTokenOut": "0x2078f336fdd260f708bec4a20c82b063274e1b23",
					"isToken0To1": false,
					"isWrapIn": true,
					"isUnwrapOut": true
				}
			}
		],
		[
			{
				"pool": "0x4f493b7de8aac7d55f71853688b1f7c8f0243c85",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 21351483738,
				"amountOut": 21346442448,
				"exchange": "curve-stable-ng",
				"poolType": "curve-stable-ng",
				"poolExtra": {
					"tokenInIndex": 0,
					"tokenOutIndex": 1,
					"underlying": false,
					"TokenInIsNative": false,
					"TokenOutIsNative": false
				},
				"extra": null
			},
			{
				"pool": "0x11b815efb8f581194ae79006d24e0d814b7697f6",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"swapAmount": 21346442448,
				"amountOut": 11749944696453640049,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 500,
					"priceLimit": "1461300573427867316570072651998408279850435624080"
				},
				"extra": {
					"nSqrtRx96": "3376699687610020413522784"
				}
			},
			{
				"pool": "0x4585fe77225b41b697c938b018e2ac67ac5a20c0",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 11749944696453640049,
				"amountOut": 22110597,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 500,
					"priceLimit": "1461300573427867316570072651998408279850435624080"
				},
				"extra": {
					"nSqrtRx96": "57745751981822148905609262241903366"
				}
			}
		],
		[
			{
				"pool": "0x8aa4e11cbdf30eedc92100f4c8a31ff748e201d44712cc8c90d189edaa8e4e47",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 85405934952,
				"amountOut": 85385229148,
				"exchange": "uniswap-v4",
				"poolType": "uniswap-v4",
				"poolExtra": {
					"router": "0x66a9893cc07d91d95644aedd05d03f95e1dba8af",
					"permit2Addr": "0x000000000022d473030f116ddee9f6b43ac78ba3",
					"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"fee": 10,
					"tickSpacing": 1,
					"hookAddress": "0x0000000000000000000000000000000000000000",
					"hookData": ""
				},
				"extra": {
					"nSqrtRx96": "79218328519294220572413293995"
				}
			},
			{
				"pool": "kyber_pmm_0x2260fac5e5542a773aa44fbcfedf7c193bc2c599_0x6b175474e89094c44da98b954eedeac495271d0f_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 85385229148,
				"amountOut": 88415669,
				"exchange": "kyber-pmm",
				"poolType": "kyber-pmm",
				"poolExtra": {
					"timestamp": 1746173284
				},
				"extra": {
					"takerAsset": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"takingAmount": "85385229148",
					"makerAsset": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
					"makingAmount": "88435270"
				}
			}
		],
		[
			{
				"pool": "0x85b2b559bc2d21104c4defdd6efca8a20343361d",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 64054451214,
				"amountOut": 64039469045,
				"exchange": "balancer-v3-stable",
				"poolType": "balancer-v3-stable",
				"poolExtra": {
					"buffIn": "0xd4fa2d31b7968e448877f69a96de69f5de8cd23e",
					"buffOut": "0x7bc3485026ac48b6cf9baf0a377477fff5703af8"
				},
				"extra": {
					"Buffers": null,
					"AggregateFee": 1420096
				}
			},
			{
				"pool": "pmm_2_0x2260fac5e5542a773aa44fbcfedf7c193bc2c599_0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"swapAmount": 64039469045,
				"amountOut": 66307320,
				"exchange": "pmm-2",
				"poolType": "pmm-2",
				"poolExtra": {
					"timestamp": 1746173284
				},
				"extra": {
					"b": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"bAmt": "64039469045",
					"q": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
					"qAmt": "66327142"
				}
			}
		]
	],
	"alphaFee": {
		"ammAmount": "442243792",
		"swapReductions": [
			{
				"ExecutedId": 27,
				"Pool": "kyber_pmm_0x2260fac5e5542a773aa44fbcfedf7c193bc2c599_0x6b175474e89094c44da98b954eedeac495271d0f_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
				"Token": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"ReduceAmount": "20485",
				"ReduceAmountUsd": 19.734573122445873
			},
			{
				"ExecutedId": 29,
				"Pool": "pmm_2_0x2260fac5e5542a773aa44fbcfedf7c193bc2c599_0xdac17f958d2ee523a2206206994597c13d831ec7",
				"Token": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				"ReduceAmount": "20485",
				"ReduceAmountUsd": 19.734573122445873
			}
		]
	},
	"routeID": "my-custom-route-id:1746173289130",
	"checksum": "7509029929040348999",
	"timestamp": 1746173288
}
`

func TestCalculateDefaultAlphaFeeNonMergeRoute(t *testing.T) {
	ctx := context.Background()

	var routeSummary valueobject.RouteSummary
	if err := json.Unmarshal([]byte(routeSummaryStr), &routeSummary); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	param := DefaultAlphaFeeParams{
		RouteSummary: routeSummary,
	}

	calc := AlphaFeeV2Calculation{
		reductionFactorInBps: big.NewInt(5000),
		config: valueobject.AlphaFeeConfig{
			ReductionConfig: valueobject.AlphaFeeReductionConfig{
				DefaultAlphaFeePercentageBps: 8000, // charge too much fee
			},
		},
	}

	// In case of charging too much fee, the alpha fee must still smaller
	// than the possible amountOut in swap.
	alphaFeeV2, err := calc.CalculateDefaultAlphaFee(ctx, param)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, reduction := range alphaFeeV2.SwapReductions {
		executedId := 0
		for _, path := range routeSummary.Route {
			for _, swap := range path {
				if executedId == reduction.ExecutedId {
					assert.Equal(t, reduction.ReduceAmount.Cmp(swap.AmountOut), -1,
						"alpha fee must be smaller than the possible amountOut in swap")
				}
				executedId++
			}
		}
	}
}

const mergeRouteSummaryStr = `
{
	"tokenIn": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	"amountIn": 50000000000000000000,
	"amountInUsd": 90219.38534732959,
	"tokenOut": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
	"amountOut": 115120865977379351237993,
	"amountOutUsd": 90099.30050184815,
	"gas": 1357000,
	"gasPrice": "359550148",
	"gasUsd": 0.8803779956303116,
	"l1FeeUsd": 0,
	"extraFee": {
		"feeAmount": [0],
		"chargeFeeBy": "",
		"isInBps": false,
		"feeReceiver": [""]
	},
	"route": [
		[
			{
				"pool": "0xa3f558aebaecaf0e11ca4b2199cc5ed341edfd74",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
				"swapAmount": 34000000000000000000,
				"amountOut": 78307603482183558609312,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 3000,
					"priceLimit": "1457652066949847389969617340386294118487833376467"
				},
				"extra": {
					"nSqrtRx96": "1649216578716471285217992649"
				}
			}
		],
		[
			{
				"pool": "0x8626be23f128f5985a5d76359b10bf3db84a7306",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
				"swapAmount": 4500000000000000000,
				"amountOut": 1642647526655218546985,
				"exchange": "ringswap",
				"poolType": "ringswap",
				"poolExtra": {
					"fee": 3,
					"feePrecision": 1000,
					"blockNumber": 22422214
				},
				"extra": {
					"wTokenIn": "0xa250cc729bb3323e7933022a67b52200fe354767",
					"wTokenOut": "0xe8e1f50392bd61d0f8f48e8e7af51d3b8a52090a",
					"isToken0To1": true,
					"isWrapIn": true,
					"isUnwrapOut": true
				}
			}
		],
		[
			{
				"pool": "pmm_2_0x5a98fcbea516cf06857215779fd812ca3bef1b32_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
				"swapAmount": 6500000000000000000,
				"amountOut": 14954709148645699039149,
				"exchange": "pmm-2",
				"poolType": "pmm-2",
				"poolExtra": {
					"timestamp": 1746504814
				},
				"extra": {
					"b": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					"bAmt": "6500000000000000000",
					"q": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
					"qAmt": "14960532037708923535360"
				}
			}
		],
		[
			{
				"pool": "0xe0554a476a092703abdb3ef35c80e0d76d32939f",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"swapAmount": 2500000000000000000,
				"amountOut": 4512252777,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 100,
					"priceLimit": "1461446703485210103287273052203988822378723970341"
				},
				"extra": {
					"nSqrtRx96": "1865002396564034920628940868223229"
				}
			}
		],
		[
			{
				"pool": "0xc7bbec68d12a0d1830360f8ec58fa599ba1b0e9b",
				"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 2500000000000000000,
				"amountOut": 4512231613,
				"exchange": "uniswapv3",
				"poolType": "uniswapv3",
				"poolExtra": {
					"swapFee": 100,
					"priceLimit": "4295128740"
				},
				"extra": {
					"nSqrtRx96": "3365763793342467160202785"
				}
			}
		],
		[
			{
				"pool": "kyber_pmm_0x1f9840a85d5af5bf1d1762f925bdaddc4201f984_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenIn": "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 1642647526655218546985,
				"amountOut": 8121091860,
				"exchange": "kyber-pmm",
				"poolType": "kyber-pmm",
				"poolExtra": {
					"timestamp": 1746504814
				},
				"extra": {
					"takerAsset": "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
					"takingAmount": "1642647526655218546985",
					"makerAsset": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"makingAmount": "8122632125"
				}
			}
		],
		[
			{
				"pool": "0x4f493b7de8aac7d55f71853688b1f7c8f0243c85",
				"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"swapAmount": 4512252777,
				"amountOut": 4512661753,
				"exchange": "curve-stable-ng",
				"poolType": "curve-stable-ng",
				"poolExtra": {
					"tokenInIndex": 0,
					"tokenOutIndex": 1,
					"underlying": false,
					"TokenInIsNative": false,
					"TokenOutIsNative": false
				},
				"extra": null
			}
		],
		[
			{
				"pool": "pmm_2_0x5a98fcbea516cf06857215779fd812ca3bef1b32_0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
				"swapAmount": 12633753613,
				"amountOut": 16107140115897753703955,
				"exchange": "pmm-2",
				"poolType": "pmm-2",
				"poolExtra": {
					"timestamp": 1746504814
				},
				"extra": {
					"b": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"bAmt": "12633753613",
					"q": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
					"qAmt": "16111343588419645210624"
				}
			}
		],
		[
			{
				"pool": "kyber_pmm_0x5a98fcbea516cf06857215779fd812ca3bef1b32_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"tokenIn": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"tokenOut": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
				"swapAmount": 4512231613,
				"amountOut": 5751413230652339885577,
				"exchange": "kyber-pmm",
				"poolType": "kyber-pmm",
				"poolExtra": {
					"timestamp": 1746504814
				},
				"extra": {
					"takerAsset": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"takingAmount": "4512231613",
					"makerAsset": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
					"makingAmount": "5753652625430601007104"
				}
			}
		]
	],
	"alphaFee": {
		"ammAmount": "115104631910875352897788",
		"swapReductions": [
			{
				"ExecutedId": 2,
				"Pool": "pmm_2_0x5a98fcbea516cf06857215779fd812ca3bef1b32_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"Token": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
				"ReduceAmount": "5972434659254992896",
				"ReduceAmountUsd": 4.684044316743018
			},
			{
				"ExecutedId": 5,
				"Pool": "kyber_pmm_0x1f9840a85d5af5bf1d1762f925bdaddc4201f984_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"Token": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"ReduceAmount": "1621475",
				"ReduceAmountUsd": 1.6209276257082188
			},
			{
				"ExecutedId": 7,
				"Pool": "pmm_2_0x5a98fcbea516cf06857215779fd812ca3bef1b32_0xdac17f958d2ee523a2206206994597c13d831ec7",
				"Token": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
				"ReduceAmount": "4364542312352579584",
				"ReduceAmountUsd": 3.422514295863155
			},
			{
				"ExecutedId": 8,
				"Pool": "kyber_pmm_0x5a98fcbea516cf06857215779fd812ca3bef1b32_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"Token": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
				"ReduceAmount": "2296908335432073216",
				"ReduceAmountUsd": 1.8014333024467566
			}
		]
	}
}
`

func TestCalculateDefaultAlphaFeeMergeRoute(t *testing.T) {
	ctx := context.Background()

	var routeSummary valueobject.RouteSummary
	if err := json.Unmarshal([]byte(mergeRouteSummaryStr), &routeSummary); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	param := DefaultAlphaFeeParams{
		RouteSummary: routeSummary,
	}

	calc := AlphaFeeV2Calculation{
		reductionFactorInBps: big.NewInt(5000),
		config: valueobject.AlphaFeeConfig{
			ReductionConfig: valueobject.AlphaFeeReductionConfig{
				DefaultAlphaFeePercentageBps: 1000,
			},
		},
	}

	alphaFeeV2, err := calc.CalculateDefaultAlphaFee(ctx, param)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, reduction := range alphaFeeV2.SwapReductions {
		executedId := 0
		for _, path := range routeSummary.Route {
			for _, swap := range path {
				if executedId == reduction.ExecutedId {
					assert.Equal(t, reduction.ReduceAmount.Cmp(swap.AmountOut), -1,
						"alpha fee must be smaller than the possible amountOut in swap")
				}
				executedId++
			}
		}
	}
}
