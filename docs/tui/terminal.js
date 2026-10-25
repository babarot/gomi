// A terminal running a shell, where rm -b opens gomi's TUI under the prompt
// and restoring prints what gomi prints (internal/cli/restore.go).
import { env } from './env.js';
import { PALETTE } from './palette.js';
import { Grid } from './grid.js';
import { fg } from './text.js';
import { newList, fullPath, currentFile, setView as listSetView } from './list.js';
import { view } from './view.js';
import { update } from './update.js';

// createTerminal makes the page's terminal. Options:
//   cwd, home      where gomi runs, and what ~ stands for
//   now            the clock
//   scope, view    ui.scope and ui.view
//   previewDelay   ms the cursor rests before a preview is read; 0 at once
//   shell          { ls(): names, rm(name): whether there was such a file }
//                  for the commands typed at the prompt
export function createTerminal(opts = {}) {
  const o = Object.assign({ cwd: '~/work', home: '/Users/you', now: () => new Date(), scope: 'auto', view: 'recent', previewDelay: 80, shell: {} }, opts);
  Object.assign(env, { cwd: o.cwd, home: o.home, now: o.now, previewDelay: o.previewDelay, P: PALETTE.dark });

  const files = [];          // the trash, newest first
  const history = [];        // the lines the shell printed, as spans
  const listeners = { change: [], restore: [] };
  const s = {
    W: 80, rows: 32, tuiRows: 28,
    dark: true, cells: false,
    mode: 'shell',           // shell | tui | ask
    line: '',                // what is typed at the prompt
    tui: null,               // the TUI's state while it runs
    ask: null,               // the files being restored, while gomi asks a name
  };
  let seq = 0, pvTimer = null;

  const emit = (ev, ...a) => listeners[ev].forEach(fn => fn(...a));
  env.changed = () => emit('change');

  const PS1 = () => [['~/work', fg(env.P.DirName)], [' ❯ ', { fg: env.P.Cursor, b: true }]];
  const print = (...segs) => history.push(segs);
  const echo = cmd => print(...PS1(), [cmd, {}]);

  function open() {
    echo('rm -b');
    if (!files.length) { print(["The trash is empty. Let's try deleting a file first", {}]); s.mode = 'shell'; return; }
    s.tui = newList(files, { scope: o.scope, view: o.view });
    s.mode = 'tui';
    readPreview();
  }
  function quit(msg) {
    s.tui = null; s.mode = 'shell';
    if (msg) print([msg, {}]);
  }

  // restore restores files one by one, asking a new name for one whose
  // name is taken where it goes back to
  function restore(list) {
    s.tui = null;
    s.ask = { queue: list.slice(), value: '', error: '' };
    next();
  }
  function next() {
    const a = s.ask;
    while (a.queue.length) {
      const f = a.queue[0];
      if (f.origin === 'taken') { s.mode = 'ask'; a.value = ''; a.error = ''; return; }
      a.queue.shift();
      done(f, f.name);
    }
    s.ask = null; s.mode = 'shell';
  }
  function done(f, name) {
    files.splice(files.indexOf(f), 1);
    const back = Object.assign({}, f, { name });
    print(["Restored '" + f.name + "' to " + fullPath(back), {}]);
    emit('restore', back);
  }

  const askLine = (...segs) => print(['? ', fg(env.P.ask)], ['New name to avoid to overwrite: ', {}], ...segs);
  function askKey(k) {
    const a = s.ask, f = a.queue[0];
    if (k === 'Escape' || k === 'ctrl+c') {
      askLine();
      print(['Canceled! No new filename input.', {}]);
      a.queue.shift(); return next();
    }
    if (k === 'Enter') {
      if (!a.value) { a.error = 'min: 1 characters'; return; }
      if (/[\s/]/.test(a.value)) { a.error = 'not valid chars are included'; return; }
      askLine([a.value, {}]);
      a.queue.shift();
      done(f, a.value);
      return next();
    }
    if (k === 'Backspace') a.value = a.value.slice(0, -1);
    else if (k.length === 1) a.value += k;
    a.error = '';
  }

  // the commands the prompt knows: rm -b, ls, rm of a file on the desk, clear
  function shellKey(k) {
    if (k === 'Tab' && !s.line) { s.line = 'rm -b'; return; }
    if (k === 'Backspace') { s.line = s.line.slice(0, -1); return; }
    if (k.length === 1) { s.line += k; return; }
    if (k !== 'Enter') return;
    const cmd = s.line.trim() || 'rm -b';
    s.line = '';
    if (/^(rm|gomi) (-b|--restore)$/.test(cmd)) return open();
    echo(cmd);
    if (cmd === 'clear') { history.length = 0; return; }
    if (cmd === 'ls') { const names = o.shell.ls ? o.shell.ls() : []; if (names.length) print([names.join('  '), {}]); return; }
    const rm = cmd.match(/^(?:rm|gomi)(?: -[rf]+)? (.+)$/);
    if (rm) {
      for (const n of rm[1].split(/\s+/)) if (!(o.shell.rm && o.shell.rm(n))) print(['gomi: ' + n + ': no such file or directory', {}]);
      return;
    }
    print(['zsh: command not found: ' + cmd.split(' ')[0], {}]);
  }

  // the preview is read after the cursor rests on a file (previewDelay)
  function readPreview() {
    clearTimeout(pvTimer);
    const m = s.tui; if (!m || env.previewDelay === 0) return;
    const f = currentFile(m); if (!f || m.previewed.has(f.id)) return;
    pvTimer = setTimeout(() => { if (s.tui === m) { m.previewed.add(f.id); emit('change'); } }, env.previewDelay + Math.random() * 60);
  }

  function key(k) {
    if (s.mode === 'tui') {
      const cmd = update(s.tui, k, s.W, s.tuiRows);
      if (cmd && cmd.restore) restore(cmd.restore);
      else if (cmd && cmd.quit) quit(cmd.quit);
      else readPreview();
    } else if (s.mode === 'ask') askKey(k);
    else shellKey(k);
    emit('change');
  }

  // grid is the terminal: the shell's last lines, and the TUI under the
  // prompt that ran it, or the prompt
  function grid() {
    env.P = PALETTE[s.dark ? 'dark' : 'light'];
    const P = env.P, g = new Grid(s.W, s.rows);
    if (s.mode === 'tui') {
      const shellRows = s.rows - s.tuiRows;
      const hist = shellRows > 0 ? history.slice(-shellRows) : [];
      hist.forEach((l, y) => g.put(0, y, l));
      const t = view(s.tui, s.W, s.tuiRows);
      t.c.forEach((row, y) => { if (hist.length + y < s.rows) g.c[hist.length + y] = row; });
      return g;
    }
    const lines = history.slice();
    if (s.mode === 'ask') {
      const a = s.ask, name = a.queue[0].name;
      lines.push([['? ', fg(P.ask)], ['New name to avoid to overwrite: ', {}],
        ...(a.value ? [[a.value, {}], [' ', { r: true }]] : [[name[0], { r: true }], [name.slice(1), fg(P.c240)]])]);
      if (a.error) lines.push([['✘ ', fg(P.err)], [a.error, fg(P.c240)]]);
    } else {
      lines.push([...PS1(), [s.line, {}], ...(s.line ? [[' ', { r: true }]] : [['r', { r: true, fg: P.c240 }], ['m -b', fg(P.c240)]])]);
    }
    lines.slice(-s.rows).forEach((l, y) => g.put(0, y, l));
    return g;
  }

  return {
    get files() { return files; },
    get isOpen() { return s.mode === 'tui'; },
    get view() { return s.tui && s.tui.view; },
    palette: () => PALETTE[s.dark ? 'dark' : 'light'],

    // add puts a file in the trash: { name, dir, at, size, isDir, run,
    // origin, and text, lang, img, ls, listing or mime for its preview }
    add(f) {
      files.push(Object.assign({ id: 'f' + seq++, origin: '', isDir: false }, f));
      files.sort((a, b) => b.at - a.at);
    },
    // size is the terminal's columns and rows, and how many of the rows
    // the TUI takes under the prompt
    size(cols, rows, tuiRows = rows - 4) { Object.assign(s, { W: cols, rows, tuiRows }); },
    setDark(dark) { s.dark = dark; },
    // setCells draws every letter in a cell of its own, for a font whose
    // letters differ in width
    setCells(cells) { s.cells = cells; },
    on(ev, fn) { listeners[ev].push(fn); },

    key,
    run: open,                                   // rm -b, as if typed
    quit: () => { if (s.mode === 'tui') quit('bye!'); },
    echo,                                        // a command at the prompt, not run
    print: text => print([text, {}]),            // a line of output
    setView(v) { if (s.mode !== 'tui') open(); if (s.tui) { listSetView(s.tui, v); readPreview(); } },

    // restoreNewest is a quick undo: rm -b, then enter on the row the
    // cursor starts on. It returns what the terminal is left doing:
    // 'shell', or 'ask' when the name is taken there.
    restoreNewest() {
      if (s.mode === 'ask') return 'ask';
      if (s.mode === 'tui') quit('bye!');
      open();
      if (s.mode === 'tui') key('Enter');
      return s.mode;
    },
    // reset empties the trash and the shell
    reset() {
      clearTimeout(pvTimer);
      files.length = 0; history.length = 0;
      Object.assign(s, { mode: 'shell', line: '', tui: null, ask: null });
    },

    html: () => grid().html(env.P, s.cells),
    text: () => grid().text(),
  };
}

