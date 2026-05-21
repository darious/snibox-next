/* Components: Nav rail, list, filters, row, modals, toast */

const { useState, useEffect, useMemo, useRef, useCallback } = React;

/* ---------- Relative time formatter ---------- */
function fmtRel(iso) {
  const d = new Date(iso);
  const now = new Date("2026-05-21T12:00:00Z");
  const diff = (now - d) / 1000;
  if (diff < 60) return "now";
  if (diff < 3600) return `${Math.floor(diff/60)}m`;
  if (diff < 86400) return `${Math.floor(diff/3600)}h`;
  if (diff < 86400 * 7) return `${Math.floor(diff/86400)}d`;
  if (diff < 86400 * 30) return `${Math.floor(diff/(86400*7))}w`;
  if (diff < 86400 * 365) return `${Math.floor(diff/(86400*30))}mo`;
  return `${Math.floor(diff/(86400*365))}y`;
}

function typeIcon(t) {
  if (t === "snippet") return <I.Code size={14} />;
  if (t === "note") return <I.Note size={14} />;
  if (t === "link") return <I.Link size={14} />;
  return <I.Inbox size={14} />;
}

/* ---------- Nav rail ---------- */
function NavRail({
  view, setView, counts, allTags, activeTags, toggleTag,
  collapsed, setCollapsed,
  openNew, openImport, openExport, navOpen, setNavOpen,
}) {
  const items = [
    { id: "all",      label: "All",      icon: <I.Inbox size={16} />, count: counts.all },
    { id: "snippets", label: "Snippets", icon: <I.Code size={16} />,  count: counts.snippet },
    { id: "notes",    label: "Notes",    icon: <I.Note size={16} />,  count: counts.note },
    { id: "links",    label: "Links",    icon: <I.Link size={16} />,  count: counts.link },
    { id: "pinned",   label: "Pinned",   icon: <I.PinFilled size={16} />, count: counts.pinned },
    { id: "archive",  label: "Archive",  icon: <I.Archive size={16} />,   count: counts.archived },
  ];

  return (
    <aside className={`nav ${navOpen ? "open" : ""}`}>
      <div className="nav-head">
        <div className="nav-logo">s</div>
        {!collapsed && (
          <div className="nav-title">snibox<span className="next">·next</span></div>
        )}
        <button
          className="nav-collapse"
          onClick={() => setCollapsed(!collapsed)}
          title={collapsed ? "Expand nav" : "Collapse nav"}
          aria-label="Toggle navigation width"
        >
          <I.PanelLeft size={15} />
        </button>
      </div>

      <button className="nav-new" onClick={openNew}>
        <I.Plus size={14} />
        {!collapsed && <span>New item</span>}
        {!collapsed && <span className="kbd">N</span>}
      </button>

      <div className="nav-scroll">
        {items.map(it => (
          <button
            key={it.id}
            className={`nav-item ${view.kind === it.id ? "active" : ""}`}
            onClick={() => { setView({ kind: it.id }); setNavOpen(false); }}
            title={collapsed ? it.label : ""}
          >
            <span className="ico">{it.icon}</span>
            <span className="label">{it.label}</span>
            <span className="count">{it.count}</span>
          </button>
        ))}

        {!collapsed && (
          <>
            <div className="nav-section">Tags</div>
            {allTags.length === 0 && (
              <div style={{padding: "4px 10px", fontSize: 12, color: "var(--text-mute)"}}>No tags yet</div>
            )}
            {allTags.map(({ tag, count }) => (
              <button
                key={tag}
                className={`nav-item tag-item ${activeTags.includes(tag) ? "active" : ""}`}
                onClick={() => toggleTag(tag)}
              >
                <span className="ico">#</span>
                <span className="label">{tag}</span>
                <span className="count">{count}</span>
              </button>
            ))}
          </>
        )}
      </div>

      <div className="nav-foot">
        {!collapsed && <span style={{fontFamily: "var(--font-mono)"}}>v0.1</span>}
        <button className="iconbtn" onClick={openImport} title="Import JSON"><I.Upload size={14}/></button>
        <button className="iconbtn" onClick={openExport} title="Export JSON"><I.Download size={14}/></button>
      </div>
    </aside>
  );
}

/* ---------- Search + filters + sort header ---------- */
function ListHeader({
  query, setQuery, view, activeTags, removeTag,
  typeFilter, setTypeFilter, langFilter, setLangFilter,
  availableLangs, total, sort, setSort, searchRef,
  setNavOpen,
}) {
  const titles = {
    all: "All items", snippets: "Snippets", notes: "Notes",
    links: "Links", pinned: "Pinned", archive: "Archive",
    search: "Search results", tag: `#${activeTags[0] || ""}`,
  };
  const showTypeFilter = view.kind === "all" || view.kind === "pinned" || view.kind === "search";
  const showLangFilter = view.kind === "all" || view.kind === "snippets" || view.kind === "pinned";

  return (
    <div className="list-head">
      <div style={{display: "flex", alignItems: "center", gap: 8, marginBottom: 8}}>
        <button
          className="iconbtn mobile-toggle"
          style={{display: "none"}}
          onClick={() => setNavOpen(true)}
          aria-label="Open navigation"
        ><I.Menu size={16}/></button>
        <div style={{fontSize: 13, fontWeight: 600, color: "var(--text)", letterSpacing: "-0.01em"}}>
          {titles[view.kind] || "Items"}
        </div>
        <div style={{marginLeft: "auto", fontSize: 11, color: "var(--text-mute)", fontFamily: "var(--font-mono)"}}>
          {total}
        </div>
      </div>

      <div className="search">
        <span className="ico"><I.Search size={14}/></span>
        <input
          ref={searchRef}
          type="text"
          placeholder="Search title, body, tags, url…"
          value={query}
          onChange={e => setQuery(e.target.value)}
        />
        {!query && <span className="kbd-hint"><span className="kbd">/</span></span>}
      </div>

      {(activeTags.length > 0 || showTypeFilter || showLangFilter) && (
        <div className="filters">
          {activeTags.map(t => (
            <button key={t} className="chip active" onClick={() => removeTag(t)} title="Remove tag filter">
              #{t} <span className="x">×</span>
            </button>
          ))}
          {showTypeFilter && ["snippet", "note", "link"].map(t => (
            <button
              key={t}
              className={`chip ${typeFilter === t ? "active" : ""}`}
              onClick={() => setTypeFilter(typeFilter === t ? null : t)}
            >
              {typeIcon(t)} {t}
            </button>
          ))}
          {showLangFilter && availableLangs.length > 1 && (
            <select
              className="chip"
              value={langFilter || ""}
              onChange={e => setLangFilter(e.target.value || null)}
              style={{padding: "3px 6px", cursor: "pointer"}}
            >
              <option value="">all langs</option>
              {availableLangs.map(l => <option key={l} value={l}>{l}</option>)}
            </select>
          )}
        </div>
      )}
    </div>
  );
}

/* ---------- List meta (sort) ---------- */
function ListMeta({ sort, setSort }) {
  return (
    <div className="list-meta">
      <I.Sort size={11} style={{marginRight: 5}}/>
      <select value={sort} onChange={e => setSort(e.target.value)}>
        <option value="updated">recently updated</option>
        <option value="created">recently created</option>
        <option value="title">title a→z</option>
      </select>
    </div>
  );
}

/* ---------- Item row ---------- */
function ItemRow({ item, active, onClick }) {
  const previewText = useMemo(() => {
    if (item.type === "link") return item.url || "";
    // Strip markdown noise
    return (item.body || "")
      .replace(/```[\s\S]*?```/g, " ")
      .replace(/[#>*_`\-]/g, "")
      .replace(/\s+/g, " ")
      .trim()
      .slice(0, 140);
  }, [item.body, item.type, item.url]);

  return (
    <div
      className={`row ${item.type} ${active ? "active" : ""} ${item.archived ? "archived" : ""}`}
      onClick={onClick}
    >
      <div className="row-head">
        <span className={`row-type ${item.type}`}>{typeIcon(item.type)}</span>
        <span className="row-title">{item.title || "Untitled"}</span>
        {item.pinned && <span className="row-pin"><I.PinFilled size={11}/></span>}
      </div>
      {previewText && <div className="row-preview">{previewText}</div>}
      <div className="row-foot">
        <span className="row-tags">
          {item.tags.slice(0, 3).map(t => (
            <span key={t} className="row-tag">{t}</span>
          ))}
          {item.tags.length > 3 && <span style={{color: "var(--text-mute)"}}>+{item.tags.length - 3}</span>}
        </span>
        <span className="row-date">{fmtRel(item.updated_at)}</span>
      </div>
    </div>
  );
}

/* ---------- Empty main ---------- */
function EmptyMain({ onNew }) {
  return (
    <div className="empty-main">
      <div>
        <h2>No item selected</h2>
        <p>Pick something from the list, or start a new snippet, note, or link.</p>
        <div className="shortcuts">
          <div className="row-s"><span className="kbd">N</span><span>new item</span></div>
          <div className="row-s"><span className="kbd">/</span><span>focus search</span></div>
          <div className="row-s"><span className="kbd">⌘</span><span>+</span><span className="kbd">K</span><span>command bar (soon)</span></div>
        </div>
        <button
          onClick={onNew}
          className="btn primary"
          style={{marginTop: 22}}
        ><I.Plus size={14}/> New item</button>
      </div>
    </div>
  );
}

/* ---------- New-item modal ---------- */
function NewModal({ onCreate, onClose }) {
  const [type, setType] = useState("snippet");
  const [title, setTitle] = useState("");
  const [language, setLanguage] = useState("bash");
  const [url, setUrl] = useState("");
  const [tags, setTags] = useState("");
  const titleRef = useRef(null);

  useEffect(() => { titleRef.current?.focus(); }, []);

  const onSubmit = (e) => {
    e.preventDefault();
    if (!title.trim()) return;
    const item = {
      id: "i" + Math.random().toString(36).slice(2, 9),
      title: title.trim(),
      body: type === "link" ? "" : (type === "snippet" ? "// new snippet\n" : "# " + title.trim() + "\n\n"),
      type,
      language: type === "snippet" ? (language || null) : null,
      url: type === "link" ? (url.trim() || null) : null,
      tags: tags.split(",").map(t => t.trim().replace(/^#/, "")).filter(Boolean),
      pinned: false,
      archived: false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    onCreate(item);
  };

  return (
    <div className="modal-backdrop" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <form className="modal" onSubmit={onSubmit}>
        <div className="modal-head">
          New item
          <button type="button" className="x" onClick={onClose}><I.X size={16}/></button>
        </div>
        <div className="modal-body">
          <div className="field">
            <label>Type</label>
            <div className="type-pick">
              <button type="button" className={type === "snippet" ? "on" : ""} onClick={() => setType("snippet")}>
                <I.Code size={18}/> snippet
              </button>
              <button type="button" className={type === "note" ? "on" : ""} onClick={() => setType("note")}>
                <I.Note size={18}/> note
              </button>
              <button type="button" className={type === "link" ? "on" : ""} onClick={() => setType("link")}>
                <I.Link size={18}/> link
              </button>
            </div>
          </div>
          <div className="field">
            <label>Title</label>
            <input
              ref={titleRef}
              className="input"
              value={title}
              onChange={e => setTitle(e.target.value)}
              placeholder={type === "link" ? "Tailscale admin console" : type === "snippet" ? "find files larger than 1G" : "Self-hosting reading list"}
            />
          </div>
          {type === "snippet" && (
            <div className="field">
              <label>Language</label>
              <select className="select" value={language} onChange={e => setLanguage(e.target.value)}>
                {["bash","yaml","go","javascript","typescript","python","sql","nginx","caddyfile","nix","ini","json","html","css","rust","dockerfile"].map(l => (
                  <option key={l} value={l}>{l}</option>
                ))}
              </select>
            </div>
          )}
          {type === "link" && (
            <div className="field">
              <label>URL</label>
              <input className="input" value={url} onChange={e => setUrl(e.target.value)} placeholder="https://…" />
            </div>
          )}
          <div className="field">
            <label>Tags (comma separated)</label>
            <input className="input" value={tags} onChange={e => setTags(e.target.value)} placeholder="homelab, docker" />
          </div>
        </div>
        <div className="modal-foot">
          <button type="button" className="btn ghost-border" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn primary"><I.Plus size={13}/> Create</button>
        </div>
      </form>
    </div>
  );
}

/* ---------- Import modal ---------- */
function ImportModal({ onImport, onClose }) {
  const [text, setText] = useState("");
  const [mode, setMode] = useState("merge");
  const [err, setErr] = useState(null);

  const onFile = (e) => {
    const f = e.target.files?.[0];
    if (!f) return;
    const r = new FileReader();
    r.onload = () => setText(String(r.result || ""));
    r.readAsText(f);
  };

  const onSubmit = (e) => {
    e.preventDefault();
    try {
      const parsed = JSON.parse(text);
      if (!Array.isArray(parsed)) throw new Error("Expected an array of items");
      onImport(parsed, mode);
    } catch (ex) {
      setErr(ex.message);
    }
  };

  return (
    <div className="modal-backdrop" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <form className="modal" onSubmit={onSubmit}>
        <div className="modal-head">
          Import JSON
          <button type="button" className="x" onClick={onClose}><I.X size={16}/></button>
        </div>
        <div className="modal-body">
          <div className="field">
            <label>JSON file or paste</label>
            <input type="file" accept="application/json,.json" onChange={onFile} style={{color: "var(--text-dim)"}}/>
            <textarea
              className="textarea"
              value={text}
              onChange={e => { setText(e.target.value); setErr(null); }}
              rows="6"
              placeholder='[ { "id": "...", "title": "...", "type": "note", ... } ]'
              style={{fontFamily: "var(--font-mono)", fontSize: 12}}
            />
          </div>
          <div className="field">
            <label>Mode</label>
            <div style={{display: "flex", gap: 6}}>
              <button type="button" className={`chip ${mode === "merge" ? "active" : ""}`} onClick={() => setMode("merge")}>Merge by ID</button>
              <button type="button" className={`chip ${mode === "append" ? "active" : ""}`} onClick={() => setMode("append")}>Append (new IDs)</button>
              <button type="button" className={`chip ${mode === "replace" ? "active" : ""}`} onClick={() => setMode("replace")}>Replace all</button>
            </div>
          </div>
          {err && <div style={{color: "var(--danger)", fontSize: 12, fontFamily: "var(--font-mono)"}}>{err}</div>}
        </div>
        <div className="modal-foot">
          <button type="button" className="btn ghost-border" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn primary" disabled={!text.trim()}><I.Upload size={13}/> Import</button>
        </div>
      </form>
    </div>
  );
}

/* ---------- Toast ---------- */
function ToastStack({ toasts }) {
  return (
    <div className="toast-stack">
      {toasts.map(t => (
        <div key={t.id} className="toast">
          <span className="ico"><I.Check size={13}/></span>
          {t.msg}
        </div>
      ))}
    </div>
  );
}

Object.assign(window, {
  NavRail, ListHeader, ListMeta, ItemRow, EmptyMain,
  NewModal, ImportModal, ToastStack,
  fmtRel, typeIcon,
});
