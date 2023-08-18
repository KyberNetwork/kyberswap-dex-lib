var tracer = {
  sloads: [],
  step: function (log, db) {
    if (log.op.toNumber() == 0x54 /*SLOAD*/) {
      let addr = log.contract.getAddress()
      let slot = toWord(log.stack.peek(0).toString(16))
      this.sloads.push({
        addr: toHex(addr),
        slot: toHex(slot),
        value: toHex(db.getState(addr, slot))
      })
    }
  },
  result: function (ctx) {
    return {
      sloads: this.sloads,
      output: toHex(ctx.output)
    }
  },
  fault: function () { }
}