import http from "k6/http";
import { check, sleep } from "k6";
import { collectData } from "./libs/collectData.js";
import { Trend } from "k6/metrics";

const diffRateTrend = new Trend("diff_rate");

const dataPoints = [
  1, 2, 5, 10, 11, 15, 20, 22, 25, 30, 33, 35, 40, 44, 45, 50, 55, 60, 66, 70,
  77, 80, 88, 90, 100, 500, 1000, 10000, 100000,
];
// const chains = ["avalanche", "bsc", "cronos", "ethereum", "fantom", "polygon"];
const chains = ["bsc"];
const CHAIN = chains[Math.floor(Math.random() * chains.length)];
// const BASE_URL = "https://aggregator-partners.kyberswap.com";
const BASE_URL = "https://test-dmm-aggregator-bsc.knstats.com";
const BASE_PROD_URL = "https://aggregator-api.kyberswap.com";
const MAX_AMOUNT_IN = 1000000;
const MIN_AMOUNT_IN = 0.1;

var _data = collectData();
var _tokens = _data[CHAIN];
if (_tokens.length == 0) {
  console.error(`token list on ${CHAIN} is empty`);
}

var _pair = [];
_pair.push(_tokens.splice(Math.random() * _tokens.length, 1)[0]);
_pair.push(_tokens.splice(Math.random() * _tokens.length, 1)[0]);

export const options = {
  scenarios: {
    cachePoints: {
      executor: "constant-vus",
      exec: "cachePoints",
      vus: 3,
      duration: "3m",
    },
    hottestPoint: {
      executor: "constant-vus",
      exec: "hottestPoint",
      vus: 3,
      duration: "3m",
    },
    rangePoints: {
      executor: "constant-vus",
      exec: "rangePoints",
      vus: 3,
      duration: "3m",
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<10000"], // 95% of requests must complete below 10s
    diff_rate: ["p(95)<0.5"], // 95% of diff rate must be below 0.5%
  },
};

export function cachePoints() {
  const requestID = Math.round(Math.random() * 10000);

  const tokenIn = _pair[0];
  const tokenOut = _pair[1];
  const tokenInAddress = tokenIn.address;
  const tokenOutAddress = tokenOut.address;
  // const amountIn = Math.floor(Math.floor((Math.random() * (MAX_AMOUNT_IN - MIN_AMOUNT_IN + 1) + MIN_AMOUNT_IN)*10)/10) * 10 ** Number(tokenIn.decimals);
  const amountIn =
    dataPoints[Math.floor(Math.random() * (dataPoints.length - 1)) + 1] +
    Number(10 ** Number(tokenIn.decimals))
      .toString()
      .slice(1);

  const URL = `${BASE_URL}/${CHAIN}/route?tokenIn=${tokenInAddress}&tokenOut=${tokenOutAddress}&amountIn=${amountIn}&saveGas=0&gasInclude=0`;
  console.log(`URL ${requestID} :>> `, URL);
  const res = http.get(URL);
  const PROD_URL = `${BASE_PROD_URL}/${CHAIN}/route?tokenIn=${tokenInAddress}&tokenOut=${tokenOutAddress}&amountIn=${amountIn}&saveGas=0&gasInclude=0`;
  console.log(`PROD_URL ${requestID} :>> `, PROD_URL);
  const res2 = http.get(PROD_URL);

  let amountOut1 = -1;
  let amountOut2 = -1;

  check(res, {
    "status was 200": (r) => r.status == 200,
    "correct outputAmount": (r) => {
      const body = r.body && JSON.parse(r.body);
      if (body == undefined) {
        console.error(URL);
        console.log(`RESPOND PROD ${requestID} :>> `, JSON.stringify(res));
        return false;
      }
      console.debug(`RESPOND ${requestID} :>> `, body.outputAmount);
      amountOut1 = body.outputAmount;
      return amountOut1 != 0;
    },
  });
  check(res2, {
    "status was 200": (r) => r.status == 200,
    "correct outputAmount": (r) => {
      const body = r.body && JSON.parse(r.body);
      if (body == undefined) {
        console.error(URL);
        console.log(`RESPOND PROD ${requestID}:>> `, JSON.stringify(res));
        return false;
      }
      console.debug(`RESPOND ${requestID}:>> `, body.outputAmount);
      amountOut2 = body.outputAmount;
      return amountOut2 != 0;
    },
  });

  if (amountOut1 > 0 && amountOut2 > 0) {
    let diffRate = (Math.abs(amountOut2 - amountOut1) / amountOut2) * 100;
    console.log(`amountOut1 ${requestID}:>> `, amountOut1);
    console.log(`amountOut2 ${requestID}:>> `, amountOut2);
    console.log(`diffRate ${requestID}:>> `, diffRate);
    diffRateTrend.add(diffRate);
  }

  sleep(1);
}

export default function () {

}

export function hottestPoint() {
  const requestID = Math.round(Math.random() * 10000);

  const tokenIn = _pair[0];
  const tokenOut = _pair[1];
  const tokenInAddress = tokenIn.address;
  const tokenOutAddress = tokenOut.address;
  const amountIn =
    1 +
    Number(10 ** Number(tokenIn.decimals))
      .toString()
      .slice(1);

  const URL = `${BASE_URL}/${CHAIN}/route?tokenIn=${tokenInAddress}&tokenOut=${tokenOutAddress}&amountIn=${amountIn}&saveGas=0&gasInclude=0`;
  console.log(`URL ${requestID} :>> `, URL);
  const res = http.get(URL);
  const PROD_URL = `${BASE_PROD_URL}/${CHAIN}/route?tokenIn=${tokenInAddress}&tokenOut=${tokenOutAddress}&amountIn=${amountIn}&saveGas=0&gasInclude=0`;
  console.log(`PROD_URL ${requestID} :>> `, PROD_URL);
  const res2 = http.get(PROD_URL);

  let amountOut1 = -1;
  let amountOut2 = -1;

  check(res, {
    "status was 200": (r) => r.status == 200,
    "correct outputAmount": (r) => {
      const body = r.body && JSON.parse(r.body);
      if (body == undefined) {
        console.error(URL);
        console.log(`RESPOND ${requestID} :>> `, JSON.stringify(res));
        return false;
      }
      console.debug(`RESPOND ${requestID} :>> `, body.outputAmount);
      amountOut1 = body.outputAmount;
      return amountOut1 != 0;
    },
  });
  check(res2, {
    "status was 200": (r) => r.status == 200,
    "correct outputAmount": (r) => {
      const body = r.body && JSON.parse(r.body);
      if (body == undefined) {
        console.error(URL);
        console.debug(`RESPOND PROD ${requestID}:>> `, JSON.stringify(res));
        return false;
      }
      console.debug(`RESPOND PROD ${requestID}:>> `, body.outputAmount);
      amountOut2 = body.outputAmount;
      return amountOut2 != 0;
    },
  });

  if (amountOut1 > 0 && amountOut2 > 0) {
    let diffRate = (Math.abs(amountOut2 - amountOut1) / amountOut2) * 100;
    console.log(`amountOut1 ${requestID}:>> `, amountOut1);
    console.log(`amountOut2 ${requestID}:>> `, amountOut2);
    console.log(`diffRate ${requestID}:>> `, diffRate);
    diffRateTrend.add(diffRate);
  }

  sleep(1);
}

export function rangePoints() {
  const requestID = Math.round(Math.random() * 10000);

  const tokenIn = _pair[0];
  const tokenOut = _pair[1];
  const tokenInAddress = tokenIn.address;
  const tokenOutAddress = tokenOut.address;
  const amountIn =
    Math.floor(
      (Math.random() * (MAX_AMOUNT_IN - MIN_AMOUNT_IN + 1) + MIN_AMOUNT_IN) *
        10 ** 6
    ) +
    Number(10 ** Number(tokenIn.decimals - 6))
      .toString()
      .slice(1);

  const URL = `${BASE_URL}/${CHAIN}/route?tokenIn=${tokenInAddress}&tokenOut=${tokenOutAddress}&amountIn=${amountIn}&saveGas=0&gasInclude=0`;
  console.log(`URL ${requestID} :>> `, URL);
  const res = http.get(URL);
  const PROD_URL = `${BASE_PROD_URL}/${CHAIN}/route?tokenIn=${tokenInAddress}&tokenOut=${tokenOutAddress}&amountIn=${amountIn}&saveGas=0&gasInclude=0`;
  console.log(`PROD_URL ${requestID} :>> `, PROD_URL);
  const res2 = http.get(PROD_URL);

  let amountOut1 = -1;
  let amountOut2 = -1;

  check(res, {
    "status was 200": (r) => r.status == 200,
    "correct outputAmount": (r) => {
      const body = r.body && JSON.parse(r.body);
      if (body == undefined) {
        console.error(URL);
        console.log(`RESPOND PROD ${requestID} :>> `, JSON.stringify(res));
        return false;
      }
      console.debug(`RESPOND ${requestID} :>> `, body.outputAmount);
      amountOut1 = body.outputAmount;
      return amountOut1 != 0;
    },
  });
  check(res2, {
    "status was 200": (r) => r.status == 200,
    "correct outputAmount": (r) => {
      const body = r.body && JSON.parse(r.body);
      if (body == undefined) {
        console.error(URL);
        console.log(`RESPOND PROD ${requestID}:>> `, JSON.stringify(res));
        return false;
      }
      console.debug(`RESPOND ${requestID}:>> `, body.outputAmount);
      amountOut2 = body.outputAmount;
      return amountOut2 != 0;
    },
  });

  if (amountOut1 > 0 && amountOut2 > 0) {
    let diffRate = (Math.abs(amountOut2 - amountOut1) / amountOut2) * 100;
    console.log(`amountOut1 ${requestID}:>> `, amountOut1);
    console.log(`amountOut2 ${requestID}:>> `, amountOut2);
    console.log(`diffRate ${requestID}:>> `, diffRate);
    diffRateTrend.add(diffRate);
  }

  sleep(1);
}
