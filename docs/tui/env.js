// What every part of the terminal reads: the colors it draws with, the time
// it is, and where it runs. createTerminal sets it, so a page has one
// terminal at a time.
export const env = {
  P: null,                    // the palette, palette.js
  now: () => new Date(),
  home: '/Users/you',         // ~ in the paths a restore prints
  cwd: '~/work',              // where gomi runs; what was deleted here is "here"
  previewDelay: 80,           // ms the cursor rests before a preview is read; 0 reads at once
  changed: () => {},          // something drawn later is ready, such as an image
};
