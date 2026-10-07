// The story event screen's pure half, under Node (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const { viewOf, noteOf, choiceForKey, commandFor } = require(path.join(here, '../../_datafiles/html/public/static/js/windows/window-event.js'));

const page = {
  active: true, id: 'gorge-descent', page: 'gorge-descent/start', title: 'The Gorge', picture: 'gorge',
  text: 'The road ends at air.\n\nA rope hangs from the ring.',
  result: ['The rope holds.'],
  choices: [
    { n: 1, label: 'Send a climber', open: true, who: 'Sera', risk: 'chancy' },
    { n: 2, label: 'Go down yourself', open: true },
    { n: 3, label: 'Cast a line', open: false, needs: 'a ranger' },
  ],
};

test('a page becomes paragraphs, a result and numbered choices with notes', () => {
  const v = viewOf(page);
  assert.equal(v.title, 'The Gorge');
  assert.deepEqual(v.paragraphs, ['The road ends at air.', 'A rope hangs from the ring.']);
  assert.deepEqual(v.result, ['The rope holds.']);
  assert.deepEqual(v.choices.map(c => c.note), ['Sera, chancy', '', 'closed: needs a ranger']);
  assert.equal(v.picture, 'gorge');
  assert.equal(v.ended, false);
});

test('a picture key that is not a plain key is dropped', () => {
  assert.equal(viewOf({ ...page, picture: '../x' }).picture, '');
});

test('a closed scene with no result shows nothing; with a result it ends on Continue', () => {
  assert.equal(viewOf({ active: false }), null);
  assert.equal(viewOf(null), null);
  const v = viewOf({ active: false, title: 'The Gorge', result: ['You leave.'] });
  assert.equal(v.ended, true);
  assert.deepEqual(v.result, ['You leave.']);
  assert.equal(v.choices.length, 0);
});

test('number keys answer open choices only', () => {
  const v = viewOf(page);
  assert.equal(choiceForKey(v, '1').n, 1);
  assert.equal(choiceForKey(v, '3'), null, 'closed');
  assert.equal(choiceForKey(v, '9'), null, 'no such choice');
  assert.equal(choiceForKey(v, 'a'), null);
  assert.equal(choiceForKey(viewOf({ active: false, result: ['x'] }), '1'), null);
});

test('a closed choice with no stated need still reads closed', () => {
  assert.equal(noteOf({ open: false }), 'closed');
});

test('an answer names the page it answers, so a double click cannot answer the next page', () => {
  assert.equal(commandFor(viewOf(page), 2), 'choose 2 gorge-descent/start');
  assert.equal(commandFor(viewOf({ ...page, page: 'bad token!' }), 2), 'choose 2', 'a malformed token is not sent');
  assert.equal(commandFor(null, 1), 'choose 1');
});
