import http from "k6/http";
import { check, sleep } from "k6";
import { collectData } from "./libs/collectData.js";

const CHAIN = "polygon";
const GAS_PRICE = 50;
const MAX_AMOUNT_IN = 350;
const MIN_AMOUNT_IN = 0.1;

var _data = collectData();
var _tokens = _data[CHAIN];
if (_tokens.length == 0) {
  console.error(`token list on ${CHAIN} is empty`);
}

var _pair = [];
_pair.push(_tokens.splice(Math.random() * _tokens.length, 1)[0]);
_pair.push(_tokens.splice(Math.random() * _tokens.length, 1)[0]);

// export const options = {
//   stages: [
//     { duration: "20s", target: 100 },
//     { duration: "30s", target: 100 },
//     { duration: "1m", target: 200 },
//     { duration: "1m30s", target: 200 }, // stay at 1000 users for 1 minutes 30 sec
//     { duration: "3m", target: 500 }, // ramp-up to 2000 users over 3 minutes
//     { duration: "2m", target: 500 }, // stay at 2000 users
//     // { duration: "4m", target: 1000 }, // ramp-up to 4000 users over 3 minutes (peak hour starts)
//     // { duration: "4m", target: 1000 }, // stay at 4000 users (peak hour)
//     { duration: "20s", target: 0 },
//   ],
//   thresholds: {
//     http_req_duration: ["p(99)<2000"], // 99% of requests must complete below 2s
//   },
// };
export const options = {
  stages: [
    { duration: "20s", target: 300 },
    { duration: "30s", target: 300 },
    { duration: "1m", target: 1000 },
    { duration: "1m30s", target: 1000 }, // stay at 1000 users for 1 minutes 30 sec
    { duration: "3m", target: 2000 }, // ramp-up to 2000 users over 3 minutes
    { duration: "2m", target: 2000 }, // stay at 2000 users
    { duration: "4m", target: 4000 }, // ramp-up to 4000 users over 3 minutes (peak hour starts)
    { duration: "4m", target: 4000 }, // stay at 4000 users (peak hour)
    { duration: "20s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<10000"], // 95% of requests must complete below 10s
  },
};

export default function () {
  const requestID = Math.round(Math.random() * 10000);
  console.debug("REQUEST ID :>> ", requestID);

  const BASE_URL = "https://dev-kyberswap-api.knstats.com/v1";
  // const BASE_URL = "http://localhost:8080";
  const chain = CHAIN;
  const tokenIn = _pair[0];
  const tokenOut = _pair[1];
  const tokenInAddress = tokenIn.address;
  const tokenOutAddress = tokenOut.address;
  // const amountIn = Math.floor(Math.floor((Math.random() * (MAX_AMOUNT_IN - MIN_AMOUNT_IN + 1) + MIN_AMOUNT_IN)*10)/10) * 10 ** Number(tokenIn.decimals);
  const amountIn = 1 * 10 ** Number(tokenIn.decimals);
  const gasPrice = GAS_PRICE * 1e9;

  const URL = `${BASE_URL}/${chain}/route?tokenIn=${tokenInAddress}&tokenOut=${tokenOutAddress}&amountIn=${amountIn}&saveGas=0&gasInclude=0&gasPrice=${gasPrice}`;
  console.debug(`URL ${requestID} :>> `, URL);
  const res = http.get(URL);

  if (res.body && !res.body.startsWith("<!DOCTYPE html>")) {
    check(res, {
      "status was 200": (r) => r.status == 200,
      "get outputAmount success": (r) => {
        const body = r.body && JSON.parse(r.body);
        if (body == undefined) {
          console.error(URL);
          console.debug(`RESPOND ${requestID}:>> `, JSON.stringify(res));
          return false;
        }
        console.debug(`RESPOND ${requestID}:>> `, body.outputAmount);
        return body.outputAmount != 0;
      },
    });

    sleep(1);
  } else {
    check(res, {
      "cloudflare limit": (r) => false,
    });
    console.debug("res :>> ", JSON.stringify(res));
  }
}
