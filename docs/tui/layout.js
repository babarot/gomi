// How the list view splits its rows and columns (layout.go): the preview
// goes right of the list when the terminal is at least 100 columns wide,
// as ui.preview.position auto does.

const AUTO_RIGHT_WIDTH = 100;
const MIN_RIGHT_PANE = 40, MAX_RIGHT_PANE = 64;
const MIN_LIST_WIDTH = 40;
const PANE_INFO_ROWS = 5;

// layout is { side, listW, rows, paneW, paneH } for a TUI W by H
export function layout(m, W, H) {
  let chrome = 1 + 2 + 1;                       // header; count and status; help
  if (m.filtering || m.filter !== '') chrome++;
  const avail = Math.max(1, H - chrome);
  const l = { side: null, listW: W, rows: avail };
  if (W >= AUTO_RIGHT_WIDTH) {
    const paneW = Math.min(Math.max(Math.floor(W * 42 / 100), MIN_RIGHT_PANE), MAX_RIGHT_PANE);
    if (W - paneW - 1 >= MIN_LIST_WIDTH && avail >= PANE_INFO_ROWS + 3) {
      Object.assign(l, { side: 'right', paneW, paneH: avail, listW: W - paneW - 1 });
    }
  }
  return l;
}
