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
  swap0For1?:boolean;
  a?: BigNumber;
  gamma?: BigNumber;
  invariantLast?:Bignumber;
  futureParamsTime?: Bignumber;
  priceScale?:Bignumber;
  totalSupply?:Bignumber;
  virtualPrice?:Bignumber;
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
  const d = computeDFromAdjustedBalances(a, adjustedReserveIn, adjustedReserveOut, checkOverflow);
  
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const x = adjustedReserveIn.add(feeDeductedAmountIn.mul(params.tokenInPrecisionMultiplier!));
  const y = getY(a, x, d);
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

function getAmountOutAqua(params: GetAmountParams, checkOverflow: boolean, shouldEstimateTweakPrice: boolean): BigNumber {
  if (params.reserveIn.isZero() || params.amount.isZero()) {
      return ZERO;
  }
  if (!params.a || !params.gamma || !params.invariantLast || !params.priceScale || !params.futureParamsTime || !params.totalSupply || !params.virtualPrice) {
      console.warn('getAmountOutAqua: incomplete params', params);
      return ZERO;
  }
  if (params.a?.isZero() || params.gamma?.isZero() || params.invariantLast?.isZero()) {
      console.warn('getAmountOutAqua: invalid params', params);
      return ZERO
  }

  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const tokenInPrecisionMultiplier = params.tokenInPrecisionMultiplier!;
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const tokenOutPrecisionMultiplier = params.tokenOutPrecisionMultiplier!;

  // Get XPs.
  const amountIn = params.amount;
  const balanceIn = params.reserveIn.add(amountIn);
  const balanceOut = params.reserveOut;
  const priceScale = params.priceScale;

  let xpIn;
  let xpOut;
  if (params.swap0For1) {
      xpIn = balanceIn.mul(tokenInPrecisionMultiplier);
      xpOut = balanceOut.mul(priceScale).mul(tokenOutPrecisionMultiplier).div(ETHER);
  } else {
      xpIn = balanceIn.mul(priceScale).mul(tokenInPrecisionMultiplier).div(ETHER);
      xpOut = balanceOut.mul(tokenOutPrecisionMultiplier);
  }

  let invariant;
  const timestamp = blockTimestamp();
  if (params.futureParamsTime.gt(timestamp)) {
      // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
      // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
      let oldXpIn;
      if (params.swap0For1) {
          oldXpIn = params.reserveIn.mul(tokenInPrecisionMultiplier);
          invariant = cryptoMathComputeD(params.a, params.gamma, oldXpIn, xpOut);
      } else {
          oldXpIn = params.reserveIn.mul(priceScale).mul(tokenInPrecisionMultiplier).div(ETHER);
          invariant = cryptoMathComputeD(params.a, params.gamma, xpOut, oldXpIn);
      }
  } else {
      invariant = params.invariantLast;
  }

  if (checkOverflow && balanceIn.gt(UINT128_MAX)) {
      throw Error('overflow');
  }

  let amountOutAfterFee;
  let newXpOut; // used for estimateTweakPriceAqua

  if (params.swap0For1) {
      const y = cryptoMathGetY(params.a, params.gamma, xpIn, xpOut, invariant, 1);
      const yOut = xpOut.sub(y);
      if (yOut.lte(ONE)) {
          return ZERO;
      }

      // modify xp out
      newXpOut = xpOut.sub(yOut);

      const amountOut = yOut.sub(ONE).mul(ETHER).div(priceScale.mul(tokenOutPrecisionMultiplier));
      const swapFee = getCryptoFee(params.swapFee, xpIn, newXpOut);
      const amountFee = amountOut.mul(swapFee).div(DECIMALS_5);
      amountOutAfterFee = amountOut.sub(amountFee);

      if (shouldEstimateTweakPrice) {
          // update xp out
          newXpOut = params.reserveOut.sub(amountOutAfterFee).mul(priceScale.mul(tokenOutPrecisionMultiplier)).div(ETHER);
      }
  } else {
      const y = cryptoMathGetY(params.a, params.gamma, xpOut, xpIn, invariant, 0);
      const yOut = xpOut.sub(y);
      if (yOut.lte(ONE)) {
          return ZERO;
      }

      // modify xp out
      newXpOut = xpOut.sub(yOut);

      const amountOut = yOut.sub(ONE).div(tokenOutPrecisionMultiplier);
      const swapFee = getCryptoFee(params.swapFee, newXpOut, xpIn);
      const amountFee = amountOut.mul(swapFee).div(DECIMALS_5);
      amountOutAfterFee = amountOut.sub(amountFee);

      if (shouldEstimateTweakPrice) {
          // update xp out
          newXpOut = params.reserveOut.sub(amountOutAfterFee).mul(tokenOutPrecisionMultiplier);
      }
  }

  // Calculate price
  // Do not use, the last price algorithm has been updated.
  /*
  let price;
  if (amountIn.gt(DECIMALS_5) && amountOutAfterFee.gt(DECIMALS_5)) {
      const _amountIn = amountIn.mul(tokenInPrecisionMultiplier);
      const _amountOut = amountOutAfterFee.mul(tokenOutPrecisionMultiplier);
      if (params.swap0For1) {
          price = ETHER.mul(_amountIn).div(_amountOut);
      } else {
          price = ETHER.mul(_amountOut).div(_amountIn);
      }
  }
  */

  if (shouldEstimateTweakPrice) {
      if (params.swap0For1) {
          estimateTweakPriceAqua(
              params.a, params.gamma, xpIn, newXpOut, ZERO, params.virtualPrice, priceScale, params.totalSupply, params.futureParamsTime
          );
      } else {
          estimateTweakPriceAqua(
              params.a, params.gamma, newXpOut, xpIn, ZERO, params.virtualPrice, priceScale, params.totalSupply, params.futureParamsTime
          );
      }
  }

  if (amountOutAfterFee.lte(ZERO)) {
      return ZERO;
  } else {
      return amountOutAfterFee;
  }
}

function getCryptoFee(feeData: FeeData, xp0: BigNumber, xp1: BigNumber): BigNumber {
  const gamma: BigNumber = feeData.gamma;
  const minFee: BigNumber = BigNumber.from(feeData.minFee);
  const maxFee: BigNumber = BigNumber.from(feeData.maxFee);

  //console.log('getCryptoFee', gamma.toString(), minFee.toString(), maxFee.toString());
  //console.log('getCryptoFee xp0', xp0.toString(), 'xp1', xp1.toString());
  let f = xp0.add(xp1);
  f = gamma.mul(ETHER).div(
      gamma.add(ETHER).sub(ETHER.mul(4).mul(xp0).div(f).mul(xp1).div(f))
  );

  const fee = minFee.mul(f).add(maxFee.mul(ETHER.sub(f))).div(ETHER);
  //console.log('getCryptoFee fee', fee.toString());
  return fee;
}

function blockTimestamp(): BigNumber {
  return BigNumber.from(Math.floor(Date.now() / 1000));
}

const A_MULTIPLIER = BigNumber.from(10000);
const N_COINS = BigNumber.from(2);
function cryptoMathGetY(ANN: BigNumber, gamma: BigNumber, x0: BigNumber, x1: BigNumber, D: BigNumber, i: number): BigNumber {
  //invariant(D.gte(DECIMALS_17) && D.lte(DECIMALS_33), "unsafe values D");
  if (!(D.gte(DECIMALS_17) && D.lte(DECIMALS_33))) {
      //console.warn("cryptoMathGetY: unsafe values D", D.toString());
      return ZERO;
  }

  let x_j = x0;
  if (i === 0) {
      x_j = x1;
  }

  let y = D.pow(TWO).div(x_j.mul(N_COINS.pow(TWO)));
  const K0_i = ETHER.mul(N_COINS).mul(x_j).div(D);

  //invariant(K0_i.gte(DECIMALS_16.mul(N_COINS)) && K0_i.lte(DECIMALS_20.mul(N_COINS)), "cryptoMathGetY: unsafe values x[i]");
  if (!(K0_i.gte(DECIMALS_16.mul(N_COINS)) && K0_i.lte(DECIMALS_20.mul(N_COINS)))) {
      //console.warn("cryptoMathGetY: unsafe values x[i]", K0_i.toString());
      return ZERO;
  }

  const convergence_limit = max(max(x_j.div(DECIMALS_14), D.div(DECIMALS_14)), DECIMALS_2);
  const __g1k0 = gamma.add(ETHER);

  for (let j = 0; j < 255; j++) {
      const y_prev = y;

      const K0 = K0_i.mul(y).mul(N_COINS).div(D);
      const S = x_j.add(y);

      let _g1k0 = __g1k0;
      if (_g1k0.gt(K0)) {
          _g1k0 = _g1k0.sub(K0).add(1);
      } else {
          _g1k0 = K0.sub(_g1k0).add(1);
      }

      const mul1 = (DECIMALS_18.mul(D).div(gamma).mul(_g1k0).div(gamma).mul(_g1k0).mul(A_MULTIPLIER)).div(ANN);
      const mul2 = DECIMALS_18.add((TWO.mul(DECIMALS_18)).mul(K0)).div(_g1k0);

      let yfprime = DECIMALS_18.mul(y).add(S.mul(mul2)).add(mul1);
      const _dyfprime = D.mul(mul2);

      if (yfprime.lt(_dyfprime)) {
          y = y_prev.div(TWO);
          continue;
      } else {
          yfprime = yfprime.sub(_dyfprime);
      }

      const fprime = yfprime.div(y);

      const y_minus_temp = mul1.div(fprime);
      const y_plus = (yfprime.add(ETHER.mul(D))).div(fprime).add(y_minus_temp.mul(ETHER).div(K0));
      const y_minus = y_minus_temp.add(DECIMALS_18.mul(S).div(fprime));

      if (y_plus.lt(y_minus)) {
          y = y_prev.div(2);
      } else {
          y = y_plus.sub(y_minus);
      }

      let diff;
      if (y.gt(y_prev)) {
          diff = y.sub(y_prev);
      } else {
          diff = y_prev.sub(y);
      }

      if (diff.lt(max(convergence_limit, y.div(DECIMALS_14)))) {
          const frac = y.mul(DECIMALS_18).div(D);
          //invariant(frac.gte(DECIMALS_16) && frac.lte(DECIMALS_20), "cryptoMathGetY: unsafe value for y");
          if (!(frac.gte(DECIMALS_16) && frac.lte(DECIMALS_20))) {
              //console.warn("cryptoMathGetY: unsafe values for y", frac.toString());
              return ZERO;
          }

          return y;
      }
  }

  //throw new Error("Did not converge");
  console.warn("Did not converge");
  return ZERO;
}

function max(x: BigNumber, y: BigNumber): BigNumber {
  return x.gt(y) ? x : y;
}

console.log(
  getAmountOutAqua(
    {
      reserveOut: BigNumber.from("193408158540"),
      reserveIn: BigNumber.from("8466391136317679557"),
      amount: BigNumber.from("1000000000000000000"),
      swapFee: {
        maxFee: 1000,
        gamma: BigNumber.from("230000000000000"),
        minFee: 800,
      },
      tokenInPrecisionMultiplier: BigNumber.from(1),
      tokenOutPrecisionMultiplier: BigNumber.from(1000000000),
    
      swap0For1: true,
      a:BigNumber.from(4000000),
      gamma: BigNumber.from("1450000000000000"),
      invariantLast: BigNumber.from("18973521177677971086"),
      priceScale: BigNumber.from("54451990779514461"),
      futureParamsTime: BigNumber.from("1709616182"), 
      totalSupply:    BigNumber.from("37758794556622160853"),
      virtualPrice: BigNumber.from("1076695600534779561"),
    },
    true, false,
  ).toString()
)
