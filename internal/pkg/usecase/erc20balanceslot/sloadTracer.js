var tracer = {
  sloads: [],
  step: function (log, db) {
    if (log.op.toNumber() == 0x54 /*SLOAD*/) {
      // In geth older version, debug_traceCall doesn't support JavaScript `let` keyword.
      var addr = log.contract.getAddress()
      var slot = toWord(log.stack.peek(0).toString(16))
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
