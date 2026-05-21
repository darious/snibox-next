/* Snibox Next — main app */
const { useState: uS, useEffect: uE, useMemo: uM, useRef: uR, useCallback: uCB } = React;

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "lineNumbers": true,
  "wrap": false
}/*EDITMODE-END*/;

function App() {
  const t = useTweaks(TWEAK_DEFAULTS);
  const setTweak = t.setTweak;

  // ---- Persistent items ----
  const [items, setItems] = uS(() => {
    try {
      const stored = localStorage.getItem("snibox.items");
      if (stored) return JSON.parse(stored);
    } catch (_) {}
    return window.SEED_ITEMS;
  });
  uE(() => {
    try { localStorage.setItem("snibox.items", JSON.stringify(items)); } catch (_) {}
  }, [items]);

  // ---- Persistent selection + view ----
  const [view, setView] = uS(() => {
    try { return JSON.parse(localStorage.getItem("snibox.view")) || { kind: "all" }; } catch (_) { return { kind: "all" }; }
  });
  uE(() => { localStorage.setItem("snibox.view", JSON.stringify(view)); }, [view]);

  const [selectedId, setSelectedId] = uS(() => localStorage.getItem("snibox.selectedId") || "i01");
  uE(() => { localStorage.setItem("snibox.selectedId", selectedId || ""); }, [selectedId]);

  const [query, setQuery] = uS("");
  const [activeTags, setActiveTags] = uS([]);
  const [typeFilter, setTypeFilter] = uS(null);
  const [langFilter, setLangFilter] = uS(null);
  const [sort, setSort] = uS("updated");
  const [navCollapsed, setNavCollapsed] = uS(false);
  const [navOpen, setNavOpen] = uS(false);
  const [mobileShowPreview, setMobileShowPreview] = uS(false);

  const [showNew, setShowNew] = uS(false);
  const [showImport, setShowImport] = uS(false);
  const [toasts, setToasts] = uS([]);
  const searchRef = uR(null);

  const toast = uCB((msg) => {
    const id = Math.random().toString(36).slice(2);
    setToasts(ts => [...ts, { id, msg }]);
    setTimeout(() => setToasts(ts => ts.filter(t => t.id !== id)), 2200);
  }, []);

  // ---- Derived: counts for nav ----
  const counts = uM(() => {
    const c = { all: 0, snippet: 0, note: 0, link: 0, pinned: 0, archived: 0 };
    for (const it of items) {
      if (it.archived) { c.archived++; continue; }
      c.all++;
      c[it.type]++;
      if (it.pinned) c.pinned++;
    }
    return c;
  }, [items]);

  // ---- Derived: tag list ----
  const allTags = uM(() => {
    const m = new Map();
    for (const it of items) {
      if (it.archived) continue;
      for (const tg of it.tags) m.set(tg, (m.get(tg) || 0) + 1);
    }
    return [...m.entries()]
      .map(([tag, count]) => ({ tag, count }))
      .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));
  }, [items]);

  // ---- Filtered list ----
  const filtered = uM(() => {
    let list = items.slice();

    // Scope by view
    if (view.kind === "snippets") list = list.filter(i => i.type === "snippet" && !i.archived);
    else if (view.kind === "notes") list = list.filter(i => i.type === "note" && !i.archived);
    else if (view.kind === "links") list = list.filter(i => i.type === "link" && !i.archived);
    else if (view.kind === "pinned") list = list.filter(i => i.pinned && !i.archived);
    else if (view.kind === "archive") list = list.filter(i => i.archived);
    else list = list.filter(i => !i.archived);

    // Tag filter (AND)
    if (activeTags.length) {
      list = list.filter(i => activeTags.every(t => i.tags.includes(t)));
    }
    // Type filter
    if (typeFilter && view.kind !== "snippets" && view.kind !== "notes" && view.kind !== "links") {
      list = list.filter(i => i.type === typeFilter);
    }
    // Language filter (only relevant to snippets)
    if (langFilter) {
      list = list.filter(i => i.type === "snippet" && i.language === langFilter);
    }
    // Search
    const q = query.trim().toLowerCase();
    if (q) {
      list = list.filter(i => {
        const hay = [
          i.title, i.body, i.url || "", i.language || "",
          ...i.tags,
        ].join(" ").toLowerCase();
        return hay.includes(q);
      });
    }
    // Sort: pinned first (except in archive), then user sort
    list.sort((a, b) => {
      if (view.kind !== "archive") {
        if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
      }
      if (sort === "title") return a.title.localeCompare(b.title);
      if (sort === "created") return new Date(b.created_at) - new Date(a.created_at);
      return new Date(b.updated_at) - new Date(a.updated_at);
    });
    return list;
  }, [items, view, activeTags, typeFilter, langFilter, query, sort]);

  // Available languages for filter dropdown
  const availableLangs = uM(() => {
    const set = new Set();
    items.filter(i => i.type === "snippet" && !i.archived).forEach(i => i.language && set.add(i.language));
    return [...set].sort();
  }, [items]);

  // Ensure selected item is in list; else fallback to first
  uE(() => {
    if (!filtered.find(i => i.id === selectedId)) {
      setSelectedId(filtered[0]?.id || null);
    }
  }, [filtered, selectedId]);

  const selected = uM(() => items.find(i => i.id === selectedId), [items, selectedId]);

  // ---- Mutators ----
  const patch = (id, p) => {
    setItems(list => list.map(i => i.id === id ? { ...i, ...p, updated_at: new Date().toISOString() } : i));
  };
  const create = (item) => {
    setItems(list => [item, ...list]);
    setSelectedId(item.id);
    setShowNew(false);
    toast(`Created ${item.type}`);
  };
  const del = (id) => {
    setItems(list => list.filter(i => i.id !== id));
    toast("Deleted");
  };
  const togglePin = (id) => {
    const it = items.find(i => i.id === id);
    patch(id, { pinned: !it.pinned });
    toast(it.pinned ? "Unpinned" : "Pinned");
  };
  const toggleArchive = (id) => {
    const it = items.find(i => i.id === id);
    patch(id, { archived: !it.archived });
    toast(it.archived ? "Unarchived" : "Archived");
  };
  const copyBody = (id) => {
    const it = items.find(i => i.id === id);
    if (!it) return;
    const txt = it.type === "link" ? (it.url || "") : (it.body || "");
    navigator.clipboard?.writeText(txt);
    toast("Copied to clipboard");
  };

  const toggleTag = (tag) => {
    setActiveTags(ts => ts.includes(tag) ? ts.filter(t => t !== tag) : [...ts, tag]);
  };

  const exportJson = () => {
    const blob = new Blob([JSON.stringify(items, null, 2)], { type: "application/json" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `snibox-${new Date().toISOString().slice(0,10)}.json`;
    a.click();
    toast(`Exported ${items.length} items`);
  };
  const onImport = (incoming, mode) => {
    if (mode === "replace") setItems(incoming);
    else if (mode === "append") {
      const reIded = incoming.map(it => ({ ...it, id: "i" + Math.random().toString(36).slice(2, 9) }));
      setItems(list => [...reIded, ...list]);
    } else {
      // merge by id
      const map = new Map(items.map(i => [i.id, i]));
      incoming.forEach(it => map.set(it.id, { ...map.get(it.id), ...it }));
      setItems([...map.values()]);
    }
    setShowImport(false);
    toast(`Imported ${incoming.length} items`);
  };

  // ---- Keyboard shortcuts ----
  uE(() => {
    const onKey = (e) => {
      if (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA") {
        if (e.key === "Escape") e.target.blur();
        return;
      }
      if (e.key === "/") {
        e.preventDefault();
        searchRef.current?.focus();
      } else if (e.key === "n" && !e.metaKey && !e.ctrlKey) {
        e.preventDefault();
        setShowNew(true);
      } else if (e.key === "Escape") {
        setShowNew(false); setShowImport(false);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // ---- UI ----
  return (
    <div className={`app ${navCollapsed ? "nav-collapsed" : ""}`}>
      <NavRail
        view={view} setView={(v) => { setView(v); setActiveTags([]); }}
        counts={counts} allTags={allTags}
        activeTags={activeTags} toggleTag={toggleTag}
        collapsed={navCollapsed} setCollapsed={setNavCollapsed}
        openNew={() => setShowNew(true)}
        openImport={() => setShowImport(true)}
        openExport={exportJson}
        navOpen={navOpen} setNavOpen={setNavOpen}
      />

      <section className={`list ${selected ? "" : ""}`}>
        <ListHeader
          query={query} setQuery={(v) => { setQuery(v); if (v && view.kind !== "search") setView({ kind: "search" }); else if (!v && view.kind === "search") setView({ kind: "all" }); }}
          view={view}
          activeTags={activeTags} removeTag={(t) => setActiveTags(ts => ts.filter(x => x !== t))}
          typeFilter={typeFilter} setTypeFilter={setTypeFilter}
          langFilter={langFilter} setLangFilter={setLangFilter}
          availableLangs={availableLangs}
          total={filtered.length}
          searchRef={searchRef}
          setNavOpen={setNavOpen}
        />
        <ListMeta sort={sort} setSort={setSort}/>
        <div className="list-scroll">
          {filtered.length === 0 ? (
            <div className="list-empty">
              <div className="big">∅</div>
              <div>No items match.</div>
              <div style={{marginTop: 14}}>
                <button className="btn ghost-border" onClick={() => { setQuery(""); setActiveTags([]); setTypeFilter(null); setLangFilter(null); }}>
                  Clear filters
                </button>
              </div>
            </div>
          ) : filtered.map(item => (
            <ItemRow
              key={item.id}
              item={item}
              active={item.id === selectedId}
              onClick={() => { setSelectedId(item.id); }}
            />
          ))}
        </div>
      </section>

      <section className={`main ${selected ? "open" : ""}`}>
        {selected ? (
          <ItemPane
            item={selected}
            patch={(p) => patch(selected.id, p)}
            del={() => del(selected.id)}
            togglePin={() => togglePin(selected.id)}
            toggleArchive={() => toggleArchive(selected.id)}
            copyBody={() => copyBody(selected.id)}
            removeTag={(tag) => patch(selected.id, { tags: selected.tags.filter(x => x !== tag) })}
            addTag={(tag) => {
              const cleaned = tag.trim().replace(/^#/, "").toLowerCase();
              if (cleaned && !selected.tags.includes(cleaned)) {
                patch(selected.id, { tags: [...selected.tags, cleaned] });
              }
            }}
            setLanguage={(l) => patch(selected.id, { language: l })}
            lineNumbers={t.lineNumbers}
            wrap={t.wrap}
            mobileShowPreview={mobileShowPreview}
            setMobileShowPreview={setMobileShowPreview}
            availableLangs={["bash","yaml","go","javascript","typescript","python","sql","nginx","caddyfile","nix","ini","json","html","css","rust","dockerfile","plaintext"]}
            onBack={() => setSelectedId(null)}
          />
        ) : (
          <EmptyMain onNew={() => setShowNew(true)}/>
        )}
      </section>

      {showNew && <NewModal onCreate={create} onClose={() => setShowNew(false)} />}
      {showImport && <ImportModal onImport={onImport} onClose={() => setShowImport(false)} />}

      <ToastStack toasts={toasts}/>

      {/* Tweaks panel */}
      <TweaksPanel>
        <TweakSection title="Code display">
          <TweakToggle
            label="Show line numbers"
            value={t.lineNumbers}
            onChange={v => setTweak("lineNumbers", v)}
            hint="Snippet view"
          />
          <TweakToggle
            label="Soft wrap long lines"
            value={t.wrap}
            onChange={v => setTweak("wrap", v)}
          />
        </TweakSection>
      </TweaksPanel>
    </div>
  );
}

/* ---------- Item pane (header + body) ---------- */
function ItemPane({
  item, patch, del, togglePin, toggleArchive, copyBody,
  removeTag, addTag, setLanguage,
  lineNumbers, wrap,
  mobileShowPreview, setMobileShowPreview,
  availableLangs, onBack,
}) {
  const [title, setTitle] = uS(item.title);
  const [addingTag, setAddingTag] = uS(false);
  const [newTag, setNewTag] = uS("");

  uE(() => { setTitle(item.title); }, [item.id]);

  const onTitleBlur = () => {
    if (title.trim() && title !== item.title) patch({ title: title.trim() });
  };

  return (
    <>
      <div className="main-head">
        <button
          className="iconbtn mobile-toggle"
          style={{display: "none"}}
          onClick={onBack}
          aria-label="Back to list"
        ><I.ChevronLeft size={16}/></button>
        <div className="main-title-wrap">
          <div className="main-title-icon" style={{color: `var(--type-${item.type})`}}>
            {typeIcon(item.type)}
          </div>
          <input
            className="main-title"
            value={title}
            onChange={e => setTitle(e.target.value)}
            onBlur={onTitleBlur}
            onKeyDown={e => { if (e.key === "Enter") e.target.blur(); }}
            placeholder="Untitled"
          />
        </div>
        <div className="main-actions">
          {item.type === "note" && (
            <button
              className={`btn mobile-toggle ${mobileShowPreview ? "active" : ""}`}
              style={{display: "none"}}
              onClick={() => setMobileShowPreview(v => !v)}
              title="Toggle preview"
            ><I.Eye size={13}/></button>
          )}
          {item.type !== "link" && (
            <button className="btn" onClick={copyBody} title="Copy contents"><I.Copy size={13}/></button>
          )}
          <button className={`btn ${item.pinned ? "active" : ""}`} onClick={togglePin} title={item.pinned ? "Unpin" : "Pin"}>
            {item.pinned ? <I.PinFilled size={13}/> : <I.Pin size={13}/>}
          </button>
          <button className={`btn ${item.archived ? "active" : ""}`} onClick={toggleArchive} title={item.archived ? "Unarchive" : "Archive"}>
            <I.Archive size={13}/>
          </button>
          <button className="btn danger" onClick={() => { if (confirm("Delete this item?")) del(); }} title="Delete">
            <I.Trash size={13}/>
          </button>
        </div>
      </div>

      <div className="main-meta">
        <span className="type-badge">{typeIcon(item.type)} {item.type}</span>
        {item.type === "snippet" && (
          <select
            className="lang-pill"
            value={item.language || ""}
            onChange={e => setLanguage(e.target.value || null)}
            title="Language"
          >
            {availableLangs.map(l => <option key={l} value={l}>{l}</option>)}
          </select>
        )}
        <span className="tag-row">
          {item.tags.map(tag => (
            <span key={tag} className="tag-pill">
              {tag}
              <button className="x" onClick={() => removeTag(tag)} title="Remove tag">×</button>
            </span>
          ))}
          {addingTag ? (
            <input
              autoFocus
              className="tag-pill"
              style={{background: "var(--bg-2)", border: "1px solid var(--accent-border)", outline: "none", color: "var(--text)", fontFamily: "var(--font-mono)", fontSize: 11, padding: "2px 7px", width: 80}}
              value={newTag}
              onChange={e => setNewTag(e.target.value)}
              onBlur={() => { if (newTag) addTag(newTag); setNewTag(""); setAddingTag(false); }}
              onKeyDown={e => {
                if (e.key === "Enter") { addTag(newTag); setNewTag(""); setAddingTag(false); }
                if (e.key === "Escape") { setNewTag(""); setAddingTag(false); }
              }}
              placeholder="tag…"
            />
          ) : (
            <button className="add-tag" onClick={() => setAddingTag(true)}>+ tag</button>
          )}
        </span>
        <span style={{marginLeft: "auto", color: "var(--text-mute)"}}>
          edited {fmtRel(item.updated_at)}
        </span>
      </div>

      <div className="main-body">
        {item.type === "snippet" && (
          <SnippetEditor
            item={item}
            onChange={patch}
            lineNumbers={lineNumbers}
            wrap={wrap}
            onCopy={copyBody}
          />
        )}
        {item.type === "note" && (
          <NoteEditor
            item={item}
            onChange={patch}
            mobileShowPreview={mobileShowPreview}
          />
        )}
        {item.type === "link" && (
          <LinkEditor item={item} onChange={patch} />
        )}
      </div>
    </>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
