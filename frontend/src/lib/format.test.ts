import assert from "node:assert/strict";
import test from "node:test";

import { validContentID } from "./format.ts";

test("detail query IDs accept only positive safe decimal integers", () => {
  assert.equal(validContentID("1"), 1);
  assert.equal(validContentID("9007199254740991"), Number.MAX_SAFE_INTEGER);
  for (const value of [null, "", "0", "-1", "+1", "01", "1.5", "1e2", "abc", "9007199254740992"]) {
    assert.equal(validContentID(value), null, String(value));
  }
});
