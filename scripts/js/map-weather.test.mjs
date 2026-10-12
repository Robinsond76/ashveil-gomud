// LM4: weather on the map, under Node (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const W = require(path.join(here, '../../_datafiles/html/public/static/js/map-weather.js'));

const texes = p => p.layers.map(l => l.tex);

test('clear weather, no weather, or a clear sky draws nothing', () => {
  assert.equal(W.plan(null), null);
  assert.equal(W.plan({}), null);
  assert.equal(W.plan({ name: 'clear', cloud_cover: 0 }), null);
  assert.equal(W.key(null), '');
});

test('overcast drifts cloud shadows and dims a little', () => {
  const p = W.plan({ name: 'overcast', cloud_cover: 3 });
  assert.deepEqual(texes(p), ['clouds']);
  assert.ok(p.dim > 0 && p.dim < 0.15);
  assert.ok(p.layers[0].vx > 0, 'the clouds drift');
});

test('rain falls diagonally, down and to the left, with splashes', () => {
  const p = W.plan({ name: 'rain', cloud_cover: 3 });
  assert.deepEqual(texes(p), ['clouds', 'rain']);
  const r = p.layers[1];
  assert.ok(r.vy > 0 && r.vx < 0, 'down and to the left');
  const angle = Math.atan2(-r.vx, r.vy) * 180 / Math.PI;
  assert.ok(angle > 15 && angle < 25, 'about 20 degrees off vertical: ' + angle);
  assert.ok(p.splash > 0 && !p.lightning);
  assert.equal(r.size, 2, 'a rain texture spans two tiles so its streaks read');
});

test('a storm is heavier, faster rain with lightning and a darker sky', () => {
  const rain = W.plan({ name: 'rain' }), storm = W.plan({ name: 'storm' });
  assert.deepEqual(texes(storm), ['clouds', 'rain-heavy']);
  assert.ok(storm.layers[1].vy > rain.layers[1].vy);
  assert.ok(storm.lightning && storm.dim > rain.dim && storm.splash > rain.splash);
});

test('mist, fog and thick fog drift ever thicker, thinner over the sight veil', () => {
  const a = n => W.plan({ name: n }).layers[0].alpha;
  assert.ok(a('mist') < a('fog') && a('fog') < a('thick-fog'));
  assert.deepEqual(texes(W.plan({ name: 'fog' })), ['fog']);
  assert.ok(Math.abs(W.plan({ name: 'fog' }, { fogVeil: true }).layers[0].alpha - a('fog') * 0.35) < 1e-9, 'LM2 already veils the fog');
});

test('snow falls in two layers at two speeds; a blizzard is faster and slanted', () => {
  const s = W.plan({ name: 'snow' }), b = W.plan({ name: 'Blizzard' });
  assert.deepEqual(texes(s), ['snow-far', 'snow']);
  assert.ok(s.layers[0].vy < s.layers[1].vy, 'the far layer falls slower');
  assert.ok(b.layers[1].vy > s.layers[1].vy && Math.abs(b.layers[1].vx) > Math.abs(s.layers[1].vx));
});

test('reduced motion keeps the weather but stops it', () => {
  const p = W.plan({ name: 'storm' }, { reduced: true });
  assert.ok(p.layers.every(l => l.vx === 0 && l.vy === 0));
  assert.equal(p.splash, 0);
  assert.ok(p.lightning, 'the bolt still shows; the flash is the caller\'s to skip');
});

test('lightning strikes 4 to 9 seconds apart; the flash is short, the bolt has 3 frames', () => {
  assert.equal(W.nextStrike(1000, 0), 5000);
  assert.equal(W.nextStrike(1000, 1), 10000);
  assert.equal(W.strike(0, -1), null);
  const s0 = W.strike(0, 0);
  assert.deepEqual(s0, { flash: 1, frame: 0 });
  assert.ok(W.strike(0, W.FLASH_MS / 2).flash > 0 && W.strike(0, W.FLASH_MS / 2).flash < 1);
  assert.equal(W.strike(0, W.FLASH_MS).flash, 0, 'the flash is over within 140 ms');
  assert.equal(W.strike(0, W.BOLT_MS - 1).frame, 2);
  assert.equal(W.strike(0, W.BOLT_MS), null);
  assert.ok(W.FLASH_MS < 150);
});

test('the key changes when the drawing would', () => {
  assert.notEqual(W.key({ name: 'rain' }), W.key({ name: 'storm' }));
  assert.notEqual(W.key({ name: 'rain', indoor: true }), W.key({ name: 'rain' }));
});
