// The colors, by role. The roles are those of internal/ui/styles/palette.go;
// the colors are the site's, so that the terminal sits in the page: the
// can's purple for the cursor, the desk's peach for what matches the filter.
// syn colors the previews, as chroma does with ui.preview.colorscheme.
export const PALETTE = {
  dark: {
    bg: '#120d22', fg: '#d9d0f5',
    Name: '#ece6ff', DirName: '#9db7ff', CursorRow: '#261d45', Heading: '#c9bff0',
    Cursor: '#a48cf0', Selected: '#7fdca8', FilterMatch: '#ff9e7a', FilterPrompt: '#9db7ff',
    Description: '#8a80b0', Border: '#5d4f96', PreviewBorder: '#3a2f63', PreviewError: '#8a80b0',
    Dialog: '#ff6b9a', HelpKey: '#8a80b0', HelpDesc: '#5d5384', HelpSeparator: '#3a2f63', HelpEllipsis: '#3a2f63',
    lsDir: '#9db7ff', lsExec: '#7fdca8', c240: '#6f6698', ask: '#7fdca8', err: '#ff6b9a',
    syn: { text: '#d9d0f5', kw: '#b9a3ff', fn: '#8fd3ff', cls: '#ffc6a8', str: '#9fe0b8', esc: '#ffd27a', num: '#ffb08a', com: '#6f6698', punct: '#c9bff0' },
  },
  light: {
    bg: '#fbf9ff', fg: '#2a1d4d',
    Name: '#231740', DirName: '#3557c9', CursorRow: '#ece5fb', Heading: '#4b3a86',
    Cursor: '#6a4fc4', Selected: '#2f8f5b', FilterMatch: '#d9582b', FilterPrompt: '#3557c9',
    Description: '#7a6ea3', Border: '#b7a9e6', PreviewBorder: '#ddd3f5', PreviewError: '#7a6ea3',
    Dialog: '#d6336c', HelpKey: '#7a6ea3', HelpDesc: '#a89ccc', HelpSeparator: '#d3c9f0', HelpEllipsis: '#d3c9f0',
    lsDir: '#3557c9', lsExec: '#2f8f5b', c240: '#a89ccc', ask: '#2f8f5b', err: '#d6336c',
    syn: { text: '#2a1d4d', kw: '#6a4fc4', fn: '#1f74a8', cls: '#b5532a', str: '#2f8f5b', esc: '#a86a00', num: '#c2410c', com: '#9a8fc0', punct: '#4b3a86' },
  },
};
