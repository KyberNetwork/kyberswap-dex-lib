import { BigNumber } from "ethers";

export const ZERO: BigNumber = BigNumber.from(0);
export const ONE = BigNumber.from(1);
export const TWO = BigNumber.from(2);
export const THREE = BigNumber.from(3);
export const FOUR = BigNumber.from(4);
export const TEN = BigNumber.from(10);

// Maximum value of uint256 type
export const UINT256_MAX = BigNumber.from(2).pow(256).sub(1);
export const UINT128_MAX = BigNumber.from(2).pow(128).sub(1);

export const ETHER = TEN.pow(18);
export const DECIMALS_2 = TEN.pow(2);
export const DECIMALS_3 = TEN.pow(3);
export const DECIMALS_4 = TEN.pow(4);
export const DECIMALS_5 = TEN.pow(5);
export const DECIMALS_6 = TEN.pow(6);
export const DECIMALS_8 = TEN.pow(8);
export const DECIMALS_9 = TEN.pow(9);
export const DECIMALS_10 = TEN.pow(10);
export const DECIMALS_12 = TEN.pow(12);
export const DECIMALS_14 = TEN.pow(14);
export const DECIMALS_15 = TEN.pow(15);
export const DECIMALS_16 = TEN.pow(16);
export const DECIMALS_17 = TEN.pow(17);
export const DECIMALS_18 = TEN.pow(18);
export const DECIMALS_20 = TEN.pow(20);
export const DECIMALS_24 = TEN.pow(24);
export const DECIMALS_26 = TEN.pow(26);
export const DECIMALS_33 = TEN.pow(33);
export const DECIMALS_36 = TEN.pow(36);


const MAX_LOOP_LIMIT = 256;
const MAX_FEE = BigNumber.from(100000); // 1e5
const TWO_MAX_FEE = MAX_FEE.mul(2); // 1e5
const DEFAULT_STABLE_POOL_A = BigNumber.from(1000);
const MINIMUM_LIQUIDITY = BigNumber.from(1000);
const MAX_XP = UINT128_MAX;//BigNumber.from('3802571709128108338056982581425910818');

function computeDFromAdjustedBalances(
  A: BigNumber,
  xp0: BigNumber,
  xp1: BigNumber,
  checkOverflow: boolean,
): BigNumber {
  const s = xp0.add(xp1);

  if (s.isZero()) {
      return ZERO;
  } else {
      let prevD;
      let d = s;
      const nA = A.mul(TWO);

      for (let i = 0; i < MAX_LOOP_LIMIT; i++) {
          const dSq = d.mul(d);

          if (checkOverflow && dSq.gt(UINT256_MAX)) {
              throw Error('overflow');
          }

          const d2 = dSq.div(xp0).mul(d);
          if (checkOverflow && d2.gt(UINT256_MAX)) {
              throw Error('overflow');
          }

          const dP = d2.div(xp1).div(FOUR);
          prevD = d;

          const d0 = nA.mul(s).add(dP.mul(TWO)).mul(d);
          if (checkOverflow && d0.gt(UINT256_MAX)) {
              throw Error('overflow');
          }

          d = d0.div(
              nA.sub(ONE).mul(d).add(dP.mul(THREE))
          );

          if (d.sub(prevD).abs().lte(ONE)) {
              return d;
          }
      }

      return d;
  }
}

export interface FeeData {
  gamma: BigNumber;
  minFee: number;
  maxFee: number;
}

export interface GetAmountParams {

  amount: BigNumber,
  reserveIn: BigNumber;
  reserveOut: BigNumber;
  swapFee: FeeData;
  tokenInPrecisionMultiplier?: BigNumber;
  tokenOutPrecisionMultiplier?: BigNumber;

  // Aqua pool parameters
  a?: BigNumber;
}


function getAmountOutStable(params: GetAmountParams, checkOverflow: boolean): BigNumber {
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const adjustedReserveIn = params.reserveIn.mul(params.tokenInPrecisionMultiplier!);
  if (checkOverflow && adjustedReserveIn.gt(MAX_XP)) {
      throw Error('overflow');
  }
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const adjustedReserveOut = params.reserveOut.mul(params.tokenOutPrecisionMultiplier!);
  if (checkOverflow && adjustedReserveOut.gt(MAX_XP)) {
      throw Error('overflow');
  }

  const amountIn = params.amount;
  const swapFee = params.swapFee.maxFee;
  const feeDeductedAmountIn = amountIn.sub(amountIn.mul(swapFee).div(MAX_FEE));
  const a = params.a ?? DEFAULT_STABLE_POOL_A;
  console.log("d input",adjustedReserveIn.toString(), adjustedReserveOut.toString());
  const d = computeDFromAdjustedBalances(a, adjustedReserveIn, adjustedReserveOut, checkOverflow);
  console.log("d", d.toString());
  
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const x = adjustedReserveIn.add(feeDeductedAmountIn.mul(params.tokenInPrecisionMultiplier!));
  const y = getY(a, x, d);
  console.log("y", y.toString());
  const dy = adjustedReserveOut.sub(y).sub(1);

  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const amountOut = dy.div(params.tokenOutPrecisionMultiplier!);
  console.log("amountOut",amountOut.toString());
  return amountOut;
}

function getY(A: BigNumber, x: BigNumber, d: BigNumber): BigNumber {
  const nA = A.mul(TWO);

  const c = d.mul(d).div(x.mul(TWO)).mul(d).div(nA.mul(TWO));

  const b = d.div(nA).add(x);

  let yPrev;
  let y = d;

  for (let i = 0; i < MAX_LOOP_LIMIT; i++) {
      yPrev = y;
      y = y.mul(y).add(c).div(y.mul(TWO).add(b).sub(d));

      if (y.sub(yPrev).abs().lte(ONE)) {
          break;
      }
  }

  return y;
}

console.log(
  getAmountOutStable(
    {
      reserveOut: BigNumber.from("1771167531"),
      reserveIn: BigNumber.from("8079308863505801735196"),
      amount: BigNumber.from("100000000000000000000"),
      swapFee: {
        maxFee: 100,
        gamma: BigNumber.from(1),
        minFee: 100,
      },
      tokenInPrecisionMultiplier: BigNumber.from(1),
      tokenOutPrecisionMultiplier: BigNumber.from(1000000000000),
    
      a:BigNumber.from(80)
    },
    true
  ).toString()
)
