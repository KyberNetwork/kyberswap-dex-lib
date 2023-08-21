## Reproduce steps

### Install foundry

```
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

### Build

```
git clone https://github.com/traderjoe-xyz/joe-v2
git checkout v2.1.1
forge install
forge build