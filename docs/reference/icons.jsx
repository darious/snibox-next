/* Lucide-style icon set — stroke 1.75, 16px default */
const Icon = ({ d, size = 16, stroke = 1.75, fill = "none", paths, ...rest }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={size} height={size}
    viewBox="0 0 24 24"
    fill={fill}
    stroke="currentColor"
    strokeWidth={stroke}
    strokeLinecap="round"
    strokeLinejoin="round"
    {...rest}
  >
    {paths || <path d={d} />}
  </svg>
);

const I = {
  Search:    (p) => <Icon {...p} paths={<><circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/></>} />,
  Plus:      (p) => <Icon {...p} d="M12 5v14M5 12h14" />,
  X:         (p) => <Icon {...p} d="M18 6 6 18M6 6l12 12" />,
  Check:     (p) => <Icon {...p} d="M20 6 9 17l-5-5" />,
  Pin:       (p) => <Icon {...p} paths={<><path d="M12 17v5"/><path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V7a1 1 0 0 1 1-1 2 2 0 0 0 0-4H8a2 2 0 0 0 0 4 1 1 0 0 1 1 1z"/></>} />,
  PinFilled: (p) => <Icon {...p} fill="currentColor" stroke="none" paths={<path d="M12 2a3 3 0 0 0-3 3v3.5L6.6 12 5 13v2h6v6l1 1 1-1v-6h6v-2l-1.6-1L15 8.5V5a3 3 0 0 0-3-3z"/>} />,
  Archive:   (p) => <Icon {...p} paths={<><rect x="3" y="4" width="18" height="4" rx="1"/><path d="M5 8v11a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8"/><path d="M10 13h4"/></>} />,
  Tag:       (p) => <Icon {...p} paths={<><path d="M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z"/><circle cx="7.5" cy="7.5" r="1"/></>} />,
  Code:      (p) => <Icon {...p} d="m16 18 6-6-6-6M8 6l-6 6 6 6" />,
  Note:      (p) => <Icon {...p} paths={<><path d="M15 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6"/></>} />,
  Link:      (p) => <Icon {...p} paths={<><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></>} />,
  Inbox:     (p) => <Icon {...p} paths={<><path d="M22 12h-6l-2 3h-4l-2-3H2"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></>} />,
  Copy:      (p) => <Icon {...p} paths={<><rect x="8" y="8" width="14" height="14" rx="2"/><path d="M4 16V4a2 2 0 0 1 2-2h12"/></>} />,
  Trash:     (p) => <Icon {...p} paths={<><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></>} />,
  Edit:      (p) => <Icon {...p} d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4z" />,
  Eye:       (p) => <Icon {...p} paths={<><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7z"/><circle cx="12" cy="12" r="3"/></>} />,
  External:  (p) => <Icon {...p} paths={<><path d="M15 3h6v6"/><path d="m10 14 11-11"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/></>} />,
  Menu:      (p) => <Icon {...p} d="M4 6h16M4 12h16M4 18h16" />,
  Settings:  (p) => <Icon {...p} paths={<><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></>} />,
  Download:  (p) => <Icon {...p} paths={<><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="M7 10l5 5 5-5"/><path d="M12 15V3"/></>} />,
  Upload:    (p) => <Icon {...p} paths={<><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="M17 8l-5-5-5 5"/><path d="M12 3v12"/></>} />,
  Star:      (p) => <Icon {...p} d="m12 2 3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01z" />,
  ChevronLeft:(p) => <Icon {...p} d="m15 18-6-6 6-6" />,
  ChevronRight:(p)=> <Icon {...p} d="m9 18 6-6-6-6" />,
  ChevronDown:(p) => <Icon {...p} d="m6 9 6 6 6-6" />,
  Sort:      (p) => <Icon {...p} d="M3 6h18M6 12h12M10 18h4" />,
  Hash:      (p) => <Icon {...p} d="M4 9h16M4 15h16M10 3 8 21M16 3l-2 18" />,
  Folder:    (p) => <Icon {...p} d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />,
  Filter:    (p) => <Icon {...p} d="M22 3H2l8 9.46V19l4 2v-8.54z" />,
  PanelLeft: (p) => <Icon {...p} paths={<><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 3v18"/></>} />,
  ArrowDown: (p) => <Icon {...p} d="M12 5v14M19 12l-7 7-7-7" />,
};

window.I = I;
