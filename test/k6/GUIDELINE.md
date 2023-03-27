# Load testing with K6

## Installation

Following this https://k6.io/docs/getting-started/installation/

## Config testing

Configure stages for each scenarios

```js
// load-testing.js
// Load testing configuration

export const options = {
  stages: [
    { duration: "20s", target: 100 },
    { duration: "30s", target: 100 },
    { duration: "1m", target: 1000 },
    { duration: "1m30s", target: 1000 }, // stay at 1000 users for 1 minutes 30 sec
    { duration: "3m", target: 3000 }, // ramp-up to 3000 users over 3 minutes (peak hour starts)
    { duration: "5m", target: 3000 }, // stay at 3000 users (peak hour)
    { duration: "20s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(99)<2000"], // 99% of requests must complete below 2s
  },
};
```

## Run test

### Normal mode

```sh
# Load testing to determine a system's behavior under both normal and peak conditions.
k6 run test/k6/load-testing.js

# Stress testing to determine the limits of the system.
k6 run test/k6/stress-testing.js

# Spike testing immediately overwhelms the system with an extreme surge of load.
k6 run test/k6/spike-testing.js
```

### Debug mode

To see called URL and more informations.

```sh
k6 run test/k6/load-testing.js -v
```

## Description

Random a pair for each virtual user who will request repeatedly to get outAmount for swapping 1 tokenIn each 1 second.