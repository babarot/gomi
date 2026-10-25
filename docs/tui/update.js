// What a key does in the TUI (update.go). update changes the state and
// returns what has to happen outside it, as a tea.Cmd would: restore files,
// or quit with a message; nothing when the TUI goes on.
import { VIEWS, SORTS, rows, keyOf, fixCursor, find, currentRow, currentFile, setView, move, moveFile, foldOf, select, unselect } from './list.js';
import { env } from './env.js';
import { layout } from './layout.js';

// chosen is what enter restores: the selected files, or the row's
const chosen = (m, r) => (m.sel.size ? m.files.filter(f => m.sel.has(f.id)) : !r ? [] : r.kind === 'fold' ? r.files : [r.file]);

// update handles key k (a KeyboardEvent key, ' ' for space, BackTab for
// shift+tab, ctrl+c) in a TUI W by H
export function update(m, k, W, H) {
  m.status = '';
  if (m.overlay === 'help') { if (['?', 'Escape', 'q'].includes(k)) m.overlay = null; return null; }
  if (m.overlay === 'sort') return sortKey(m, k);
  if (k === 'ctrl+c') return { quit: 'bye!' };
  if (m.filtering) return filterKey(m, k);
  if (m.detail) return detailKey(m, k);

  const rs = rows(m), r = rs[m.cursor], page = layout(m, W, H).rows;
  const folds = m.view === 'places' || m.view === 'batches';
  switch (k) {
    case 'j': case 'ArrowDown': move(m, 1); break;
    case 'k': case 'ArrowUp': move(m, -1); break;
    case 'g': case 'Home': m.cursor = 0; fixCursor(m, rs); break;
    case 'G': case 'End': m.cursor = rs.length - 1; fixCursor(m, rs); break;
    case 'l': case 'ArrowRight': case 'PageDown':
      if (folds && k !== 'PageDown') { if (r && r.kind === 'fold') m.open[m.view + ':' + r.key] = true; }
      else move(m, page);
      break;
    case 'h': case 'ArrowLeft': case 'PageUp':
      if (folds && k !== 'PageUp') {
        if (r && r.kind === 'fold') m.open[m.view + ':' + r.key] = false;
        else if (r) { const key = foldOf(m, r.file); m.open[m.view + ':' + key] = false; find(m, 'fold:' + key); }
      } else move(m, -page);
      break;
    case ' ':
      if (r && r.kind === 'fold') m.open[m.view + ':' + r.key] = !r.open;
      else if (r) { m.detail = true; m.vy = 0; }
      break;
    case 'Tab': select(m); break;
    case 'BackTab': unselect(m); break;
    case 'Enter': { const files = chosen(m, r); return files.length ? { restore: files } : null; }
    case '/': m.filtering = true; break;
    case 'v': setView(m, VIEWS[(VIEWS.indexOf(m.view) + 1) % VIEWS.length]); break;
    case '.': {
      const key = r ? keyOf(r) : '';
      m.scoped = !m.scoped;
      find(m, key);
      m.status = m.scoped ? 'Files deleted in ' + env.cwd : 'Every file in the trash';
      break;
    }
    case 's': if (m.view === 'picker') { m.overlay = 'sort'; m.sortSel = m.sort; } break;
    case '?': m.overlay = 'help'; break;
    case 'Escape': if (m.filter) m.filter = ''; else m.sel.clear(); break;
    case 'q': return { quit: 'bye!' };
  }
  return null;
}

function sortKey(m, k) {
  if (k === 'Escape' || k === 's') m.overlay = null;
  else if (k === 'j' || k === 'ArrowDown') m.sortSel = Math.min(SORTS.length - 1, m.sortSel + 1);
  else if (k === 'k' || k === 'ArrowUp') m.sortSel = Math.max(0, m.sortSel - 1);
  else if (/^[1-5]$/.test(k)) { m.sort = +k - 1; m.overlay = null; m.cursor = 0; }
  else if (k === 'Enter') { m.sort = m.sortSel; m.overlay = null; m.cursor = 0; }
  return null;
}

// filterKey types into the filter; in the picker esc hands the keys to the
// list, elsewhere it clears the filter
function filterKey(m, k) {
  if (k === 'Escape') {
    if (m.view !== 'picker') m.filter = '';
    m.filtering = false;
  } else if (k === 'Enter') {
    if (m.view === 'picker') { const files = chosen(m, currentRow(m)); if (files.length) return { restore: files }; }
    m.filtering = false;
  } else if (k === 'Backspace') { m.filter = m.filter.slice(0, -1); m.cursor = 0; }
  else if (k === 'ArrowDown') move(m, 1);
  else if (k === 'ArrowUp') move(m, -1);
  else if (k === 'Tab') select(m);
  else if (k.length === 1) { m.filter += k; m.cursor = 0; fixCursor(m, rows(m)); }
  return null;
}

// detailKey moves through the preview opened over the list
function detailKey(m, k) {
  switch (k) {
    case 'j': case 'ArrowDown': m.vy++; break;
    case 'k': case 'ArrowUp': m.vy = Math.max(0, m.vy - 1); break;
    case 'd': m.vy += 10; break;
    case 'u': m.vy = Math.max(0, m.vy - 10); break;
    case 'n': moveFile(m, 1); break;
    case 'p': moveFile(m, -1); break;
    case ' ': case 'Escape': m.detail = false; m.vy = 0; break;
    case 'Tab': select(m); break;
    case 'Enter': { const f = currentFile(m); if (f) return { restore: chosen(m, { kind: 'file', file: f }) }; break; }
    case '?': m.overlay = 'help'; break;
    case 'q': return { quit: 'bye!' };
  }
  return null;
}
