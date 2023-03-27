import http from "k6/http";
import { check, sleep } from "k6";
import { collectData } from "./libs/collectData.js";

const CHAIN = "polygon";
const GAS_PRICE = 50;
var _data = collectData();
var _tokens = _data[CHAIN];
if (_tokens.length == 0) {
  console.error(`token list on ${CHAIN} is empty`);
}

var _pair = [];
_pair.push(_tokens.splice(Math.random() * _tokens.length, 1)[0]);
_pair.push(_tokens.splice(Math.random() * _tokens.length, 1)[0]);

export const options = {
  stages: [
    { duration: "20s", target: 100 },
    { duration: "20s", target: 100 },
    { duration: "1m", target: 3000 },
    { duration: "1m30s", target: 3000 },
    { duration: "3m", target: 4000 },
    { duration: "5m", target: 4000 },
    { duration: "20s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(99)<2000"], // 99% of requests must complete below 2s
  },
};

export default function () {
  const requestID = Math.round(Math.random() * 10000)
  console.debug("REQUEST ID :>> ", requestID);
  
  const BASE_URL = "https://dev-kyberswap-api.knstats.com/v1";
  const chain = CHAIN;
  const tokenIn = _pair[0];
  const tokenOut = _pair[1];
  const tokenInAddress = tokenIn.address;
  const tokenOutAddress = tokenOut.address;
  const amountIn = 2 * 10 ** Number(tokenIn.decimals);
  const gasPrice = GAS_PRICE * 1e9;
  
  const URL = `${BASE_URL}/${chain}/route?tokenIn=${tokenInAddress}&tokenOut=${tokenOutAddress}&amountIn=${amountIn}&saveGas=0&gasInclude=0&gasPrice=${gasPrice}`;
  console.debug(`URL ${requestID} :>> `, URL);
  const res = http.get(URL);
  
  check(res, {
    "status was 200": (r) => r.status == 200,
    "get outputAmount success": (r) => {
      const body = r.body && JSON.parse(r.body);
      console.debug(`RESPOND ${requestID}:>> `, body.outputAmount);
      if (body.outputAmount == 0) console.error(URL)
      return body.outputAmount != 0;
    },
  });

  sleep(1);
}
