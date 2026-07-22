// store.test.ts は TTLStore の TTL・単回使用・sweep を検証する（clock を注入して時間を制御する）。
import { test } from "node:test";
import assert from "node:assert/strict";
import { TTLStore } from "./store.ts";

test("TTLStore: take は単回使用で、2 回目は undefined を返す", () => {
  let now = 0;
  const store = new TTLStore<string>(1000, () => now);
  store.set("code", "value");
  assert.equal(store.take("code"), "value");
  assert.equal(store.take("code"), undefined);
  assert.equal(store.size, 0);
});

test("TTLStore: get は値を保持し、繰り返し取得できる", () => {
  let now = 0;
  const store = new TTLStore<string>(1000, () => now);
  store.set("session", "value");
  assert.equal(store.get("session"), "value");
  assert.equal(store.get("session"), "value");
  assert.equal(store.size, 1);
});

test("TTLStore: TTL 経過後は get が undefined を返し、エントリを失効させる", () => {
  let now = 0;
  const store = new TTLStore<string>(1000, () => now);
  store.set("code", "value");
  now = 999;
  assert.equal(store.get("code"), "value");
  now = 1000;
  assert.equal(store.get("code"), undefined);
  assert.equal(store.size, 0);
});

test("TTLStore: 期限切れ後の take は undefined を返す", () => {
  let now = 0;
  const store = new TTLStore<string>(1000, () => now);
  store.set("code", "value");
  now = 2000;
  assert.equal(store.take("code"), undefined);
});

test("TTLStore: sweep は期限切れのみを回収し、鮮度内は残す", () => {
  let now = 0;
  const store = new TTLStore<string>(1000, () => now);
  store.set("old", "1");
  now = 500;
  store.set("new", "2");
  now = 1000;
  store.sweep();
  assert.equal(store.get("old"), undefined);
  assert.equal(store.get("new"), "2");
  assert.equal(store.size, 1);
});

test("TTLStore: clear は全エントリを破棄する", () => {
  const store = new TTLStore<string>(1000, () => 0);
  store.set("a", "1");
  store.set("b", "2");
  store.clear();
  assert.equal(store.size, 0);
});

test("TTLStore: 未知の key は undefined を返す", () => {
  const store = new TTLStore<string>(1000, () => 0);
  assert.equal(store.get("missing"), undefined);
  assert.equal(store.take("missing"), undefined);
});
